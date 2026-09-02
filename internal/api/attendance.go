package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type sessionRequest struct {
	Title    string     `json:"title"`
	Location string     `json:"location"`
	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
}

func (s *Server) createClassSession(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body sessionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	if body.StartsAt == nil {
		return invalid("starts_at", "Say when the class starts.")
	}
	if body.EndsAt != nil && !body.EndsAt.After(*body.StartsAt) {
		return invalid("ends_at", "A class must end after it starts.")
	}

	tenant := CurrentTenant(r.Context())
	var session database.ClassSession
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		if _, err := q.GetCourse(r.Context(), courseID); err != nil {
			return err
		}
		var err error
		session, err = q.CreateClassSession(r.Context(), database.CreateClassSessionParams{
			TenantID: tenant.ID, CourseID: courseID, Title: title,
			Location: strings.TrimSpace(body.Location),
			StartsAt: timestamp(body.StartsAt), EndsAt: timestamp(body.EndsAt),
			CreatedBy: uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, session)
}

func (s *Server) listClassSessions(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var sessions []database.ClassSession
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		sessions, err = q.ListClassSessions(r.Context(), database.ListClassSessionsParams{
			CourseID: courseID, PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) sessionRoll(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	var (
		session database.ClassSession
		roll    []database.SessionRollRow
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if session, err = q.GetClassSession(r.Context(), sessionID); err != nil {
			return err
		}
		roll, err = q.SessionRoll(r.Context(), database.SessionRollParams{
			SessionID: sessionID, CourseID: session.CourseID,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"session": session, "roll": roll})
}

type rollEntry struct {
	EnrollmentID string `json:"enrollment_id"`
	Status       string `json:"status"`
	Note         string `json:"note"`
}

type takeRollRequest struct {
	Entries []rollEntry `json:"entries"`
}

// takeRoll marks a whole class in one request, because a teacher calls the
// register once, not once per learner.
func (s *Server) takeRoll(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body takeRollRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if len(body.Entries) == 0 {
		return invalid("entries", "Mark at least one learner.")
	}
	if len(body.Entries) > 500 {
		return invalid("entries", "Mark at most 500 learners at a time.")
	}

	type mark struct {
		enrollment uuid.UUID
		status     database.AttendanceStatus
		note       string
	}
	marks := make([]mark, 0, len(body.Entries))
	for _, entry := range body.Entries {
		enrollmentID, err := uuid.Parse(strings.TrimSpace(entry.EnrollmentID))
		if err != nil {
			return invalid("entries", "One of the learners is not valid.")
		}
		status, err := parseAttendance(entry.Status)
		if err != nil {
			return err
		}
		if len(entry.Note) > 500 {
			return invalid("entries", "A note must be under 500 characters.")
		}
		marks = append(marks, mark{enrollment: enrollmentID, status: status, note: strings.TrimSpace(entry.Note)})
	}

	tenant := CurrentTenant(r.Context())
	marker := Authenticated(r.Context()).UserID
	var (
		session   database.ClassSession
		absentees []uuid.UUID
	)

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		if session, err = q.GetClassSession(r.Context(), sessionID); err != nil {
			return err
		}
		absentees = absentees[:0]
		for _, m := range marks {
			row, err := q.MarkAttendance(r.Context(), database.MarkAttendanceParams{
				TenantID: tenant.ID, SessionID: sessionID, EnrollmentID: m.enrollment,
				Status: m.status, Note: m.note,
				MarkedBy: uuid.NullUUID{UUID: marker, Valid: true},
			})
			if err != nil {
				return err
			}
			if row.Status == database.AttendanceStatusAbsent {
				absentees = append(absentees, row.EnrollmentID)
			}
		}
		return nil
	})
	if database.IsNotFound(err) || isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	s.announceAbsences(r, tenant.ID, session, absentees)
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"marked": len(marks), "absent": len(absentees),
	})
}

// announceAbsences tells the learner and anyone responsible for them. This is
// the message a madrasa actually wants: a parent hears the same day.
func (s *Server) announceAbsences(r *http.Request, tenantID uuid.UUID, session database.ClassSession, absentees []uuid.UUID) {
	if s.notifier == nil || len(absentees) == 0 {
		return
	}

	var recipients []uuid.UUID
	err := s.store.InTenant(r.Context(), tenantID, func(q *database.Queries) error {
		for _, enrollmentID := range absentees {
			enrollment, err := q.GetEnrollmentByID(r.Context(), enrollmentID)
			if err != nil {
				return err
			}
			recipients = append(recipients, enrollment.UserID)

			guardians, err := q.GuardiansOf(r.Context(), enrollment.UserID)
			if err != nil {
				return err
			}
			for _, guardian := range guardians {
				recipients = append(recipients, guardian.GuardianID)
			}
		}
		return nil
	})
	if err != nil {
		return
	}

	title := "Absent from " + session.Title
	body := "Marked absent for " + session.Title + " on " + session.StartsAt.Time.Format("2 January 2006") + "."
	for _, userID := range recipients {
		s.notifyUser(r.Context(), tenantID, userID, "attendance.absent", title, body,
			map[string]any{"session_id": session.ID, "starts_at": session.StartsAt.Time})
	}
}

type attendanceSummary struct {
	Present  int64 `json:"present"`
	Late     int64 `json:"late"`
	Absent   int64 `json:"absent"`
	Excused  int64 `json:"excused"`
	Rate     int   `json:"rate_percent"`
	Recorded int64 `json:"sessions_recorded"`
}

func (s *Server) myAttendance(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}

	var (
		rows    []database.MyAttendanceRow
		summary attendanceSummary
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
			CourseID: courseID, UserID: Authenticated(r.Context()).UserID,
		})
		if err != nil {
			return err
		}
		if rows, err = q.MyAttendance(r.Context(), database.MyAttendanceParams{
			EnrollmentID: enrollment.ID, PageLimit: limit, PageOffset: offset,
		}); err != nil {
			return err
		}
		totals, err := q.AttendanceSummary(r.Context(), database.AttendanceSummaryParams{
			CourseID: courseID, EnrollmentID: enrollment.ID,
		})
		if err != nil {
			return err
		}
		summary = attendanceSummary{
			Present: totals.Present, Late: totals.Late, Absent: totals.Absent, Excused: totals.Excused,
		}
		summary.Recorded = totals.Present + totals.Late + totals.Absent + totals.Excused
		summary.Rate = attendanceRate(totals.Present, totals.Late, totals.Absent)
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"sessions": rows, "summary": summary})
}

// attendanceRate counts a late arrival as attended and ignores excused
// absences, so authorised leave does not damage a learner's record.
func attendanceRate(present, late, absent int64) int {
	counted := present + late + absent
	if counted == 0 {
		return 0
	}
	return int(((present+late)*200 + counted) / (counted * 2))
}

func parseAttendance(v string) (database.AttendanceStatus, error) {
	switch database.AttendanceStatus(strings.TrimSpace(v)) {
	case database.AttendanceStatusPresent:
		return database.AttendanceStatusPresent, nil
	case database.AttendanceStatusLate:
		return database.AttendanceStatusLate, nil
	case database.AttendanceStatusAbsent:
		return database.AttendanceStatusAbsent, nil
	case database.AttendanceStatusExcused:
		return database.AttendanceStatusExcused, nil
	default:
		return "", invalid("status", "Status must be present, late, absent or excused.")
	}
}

type guardianRequest struct {
	GuardianID string `json:"guardian_id"`
	StudentID  string `json:"student_id"`
	Relation   string `json:"relation"`
}

func (s *Server) addGuardian(w http.ResponseWriter, r *http.Request) error {
	var body guardianRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	guardianID, err := uuid.Parse(strings.TrimSpace(body.GuardianID))
	if err != nil {
		return invalid("guardian_id", "Name the guardian's account.")
	}
	studentID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
	if err != nil {
		return invalid("student_id", "Name the learner's account.")
	}
	if guardianID == studentID {
		return invalid("guardian_id", "A learner cannot be their own guardian.")
	}
	if len(body.Relation) > 50 {
		return invalid("relation", "Keep the relation short.")
	}

	tenant := CurrentTenant(r.Context())
	var link database.Guardianship
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		link, err = q.AddGuardian(r.Context(), database.AddGuardianParams{
			TenantID: tenant.ID, GuardianID: guardianID, StudentID: studentID,
			Relation: strings.TrimSpace(body.Relation),
		})
		return err
	})
	if isForeignKeyViolation(err) {
		return invalid("guardian_id", "One of those accounts does not exist.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, link)
}

func (s *Server) listGuardians(w http.ResponseWriter, r *http.Request) error {
	var rows []database.ListGuardianshipsRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListGuardianships(r.Context())
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"guardianships": rows})
}
