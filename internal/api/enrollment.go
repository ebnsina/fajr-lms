package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type enrollRequest struct {
	UserID string `json:"user_id"`
}

// enroll adds a learner to a course. Staff may enroll anyone; a learner may
// enroll themselves only in a published course that is not private.
func (s *Server) enroll(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body enrollRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	caller := Authenticated(r.Context()).UserID
	target := caller
	source := database.EnrollmentSourceSelf
	if raw := strings.TrimSpace(body.UserID); raw != "" {
		if target, err = uuid.Parse(raw); err != nil {
			return invalid("user_id", "Provide the id of the learner to enroll.")
		}
		source = database.EnrollmentSourceStaff
	}
	if target != caller && !staffRole(CurrentRole(r.Context())) {
		return httpx.ErrForbidden
	}

	tenant := CurrentTenant(r.Context())
	var enrollment database.Enrollment
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		course, err := q.GetCourse(r.Context(), courseID)
		if err != nil {
			return err
		}
		if source == database.EnrollmentSourceSelf && !selfEnrollable(course) {
			return httpx.ErrForbidden
		}
		enrollment, err = q.EnrollUser(r.Context(), database.EnrollUserParams{
			TenantID: tenant.ID, CourseID: courseID, UserID: target, Source: source,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if isForeignKeyViolation(err) {
		return invalid("user_id", "That learner does not exist.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, enrollment)
}

// selfEnrollable allows self-enrollment only into a free, published, listed course.
func selfEnrollable(c database.Course) bool {
	return c.Status == database.PublishStatusPublished &&
		c.Visibility != database.CourseVisibilityPrivate &&
		c.PriceMinor == 0
}

func staffRole(role string) bool {
	switch role {
	case "owner", "admin", "instructor", "assistant":
		return true
	default:
		return false
	}
}

func (s *Server) listMyEnrollments(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var rows []database.ListMyEnrollmentsRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListMyEnrollments(r.Context(), database.ListMyEnrollmentsParams{
			UserID: Authenticated(r.Context()).UserID, PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"enrollments": rows})
}

type rosterEntry struct {
	database.ListCourseRosterRow
	PercentComplete int `json:"percent_complete"`
}

func (s *Server) courseRoster(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}

	var (
		rows  []database.ListCourseRosterRow
		total int64
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if rows, err = q.ListCourseRoster(r.Context(), database.ListCourseRosterParams{
			CourseID: courseID, PageLimit: limit, PageOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountPublishedLessons(r.Context(), courseID)
		return err
	})
	if err != nil {
		return err
	}

	entries := make([]rosterEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, rosterEntry{
			ListCourseRosterRow: row,
			PercentComplete:     percent(row.LessonsCompleted, total),
		})
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"roster": entries, "published_lessons": total,
	})
}

type progressRequest struct {
	PositionS int32 `json:"position_s"`
	Completed bool  `json:"completed"`
}

type progressResponse struct {
	Progress        database.LessonProgress `json:"progress"`
	LessonsTotal    int64                   `json:"lessons_total"`
	LessonsDone     int64                   `json:"lessons_done"`
	PercentComplete int                     `json:"percent_complete"`
	CourseComplete  bool                    `json:"course_complete"`
}

// recordProgress merges a report from any device. Resume points move forward
// only and a completed lesson never reverts, so a late offline sync is safe.
func (s *Server) recordProgress(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body progressRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.PositionS < 0 {
		return invalid("position_s", "Position cannot be negative.")
	}

	tenant := CurrentTenant(r.Context())
	userID := Authenticated(r.Context()).UserID
	var out progressResponse

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		courseID, err := q.LessonCourse(r.Context(), lessonID)
		if err != nil {
			return err
		}
		enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
			CourseID: courseID, UserID: userID,
		})
		if err != nil {
			return err
		}
		if enrollment.Status == database.EnrollmentStatusCancelled {
			return httpx.ErrForbidden
		}

		if out.Progress, err = q.RecordProgress(r.Context(), database.RecordProgressParams{
			TenantID: tenant.ID, EnrollmentID: enrollment.ID, LessonID: lessonID,
			PositionS: body.PositionS, Completed: body.Completed,
		}); err != nil {
			return err
		}

		if out.LessonsTotal, err = q.CountPublishedLessons(r.Context(), courseID); err != nil {
			return err
		}
		if out.LessonsDone, err = q.CountCompletedLessons(r.Context(), enrollment.ID); err != nil {
			return err
		}

		out.PercentComplete = percent(out.LessonsDone, out.LessonsTotal)
		out.CourseComplete = out.LessonsTotal > 0 && out.LessonsDone >= out.LessonsTotal

		if body.Completed {
			s.award(r.Context(), q, tenant, userID, "lesson", lessonID, pointsPerLesson)
		}
		if out.CourseComplete && enrollment.Status != database.EnrollmentStatusCompleted {
			if _, err = q.CompleteEnrollment(r.Context(), enrollment.ID); err != nil {
				return err
			}
			s.award(r.Context(), q, tenant, userID, "course", courseID, pointsPerCourse)
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, out)
}

func (s *Server) myCourseProgress(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	var (
		rows       []database.LessonProgress
		enrollment database.Enrollment
		total      int64
		done       int64
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if enrollment, err = q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
			CourseID: courseID, UserID: Authenticated(r.Context()).UserID,
		}); err != nil {
			return err
		}
		if rows, err = q.ListProgressForEnrollment(r.Context(), enrollment.ID); err != nil {
			return err
		}
		if total, err = q.CountPublishedLessons(r.Context(), courseID); err != nil {
			return err
		}
		done, err = q.CountCompletedLessons(r.Context(), enrollment.ID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"enrollment": enrollment, "lessons": rows,
		"lessons_total": total, "lessons_done": done,
		"percent_complete": percent(done, total),
	})
}

func (s *Server) cancelEnrollment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.CancelEnrollment(r.Context(), id)
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	return httpx.NoContent(w)
}

func percent(done, total int64) int {
	if total <= 0 {
		return 0
	}
	if done >= total {
		return 100
	}
	return int(done * 100 / total)
}
