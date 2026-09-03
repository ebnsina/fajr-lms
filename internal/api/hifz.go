package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// Hifz: the daily record a teacher keeps of what a student has memorised. The
// three kinds are the ones every madrasah already uses — sabaq is today's new
// lesson, sabqi the recent revision, manzil the older revision kept warm.

type hifzRequest struct {
	StudentID string `json:"student_id"`
	OnDate    string `json:"on_date"`
	Kind      string `json:"kind"`
	FromSurah int16  `json:"from_surah"`
	FromAyah  int16  `json:"from_ayah"`
	ToSurah   int16  `json:"to_surah"`
	ToAyah    int16  `json:"to_ayah"`
	Quality   string `json:"quality"`
	Mistakes  int16  `json:"mistakes"`
	Note      string `json:"note"`
}

func parseHifzKind(raw string) (database.HifzKind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sabaq":
		return database.HifzKindSabaq, nil
	case "sabqi":
		return database.HifzKindSabqi, nil
	case "manzil":
		return database.HifzKindManzil, nil
	}
	return "", invalid("kind", "A sitting is sabaq, sabqi or manzil.")
}

func parseQuality(raw string) (database.HifzQuality, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "good":
		return database.HifzQualityGood, nil
	case "excellent":
		return database.HifzQualityExcellent, nil
	case "fair":
		return database.HifzQualityFair, nil
	case "weak":
		return database.HifzQualityWeak, nil
	}
	return "", invalid("quality", "Mark it excellent, good, fair or weak.")
}

// recordHifz writes one sitting. The range is checked against the real ayah
// counts, so a slip of the finger cannot record an ayah that does not exist.
func (s *Server) recordHifz(w http.ResponseWriter, r *http.Request) error {
	var body hifzRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	studentID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
	if err != nil {
		return invalid("student_id", "Name the student this sitting is for.")
	}
	kind, err := parseHifzKind(body.Kind)
	if err != nil {
		return err
	}
	quality, err := parseQuality(body.Quality)
	if err != nil {
		return err
	}
	if body.Mistakes < 0 || body.Mistakes > 999 {
		return invalid("mistakes", "That is not a number of mistakes.")
	}
	if len(body.Note) > 500 {
		return invalid("note", "Keep the note under 500 characters.")
	}
	if body.FromSurah < 1 || body.FromSurah > 114 || body.ToSurah < 1 || body.ToSurah > 114 {
		return invalid("from_surah", "The Qur'an has 114 surahs.")
	}
	if body.FromAyah < 1 || body.ToAyah < 1 {
		return invalid("from_ayah", "Ayahs are counted from one.")
	}
	if body.ToSurah < body.FromSurah ||
		(body.ToSurah == body.FromSurah && body.ToAyah < body.FromAyah) {
		return invalid("to_ayah", "The range runs forwards, from the first ayah to the last.")
	}

	onDate := pgtype.Date{Time: time.Now(), Valid: true}
	if strings.TrimSpace(body.OnDate) != "" {
		if onDate, err = parseDay("on_date", body.OnDate); err != nil {
			return err
		}
	}

	tenant := CurrentTenant(r.Context())
	teacher := uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true}
	var entry database.HifzEntry

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		// Both ends are checked against the surah's own length.
		for _, end := range []struct {
			surah, ayah int16
			field       string
		}{
			{body.FromSurah, body.FromAyah, "from_ayah"},
			{body.ToSurah, body.ToAyah, "to_ayah"},
		} {
			count, err := q.SurahAyahCount(r.Context(), end.surah)
			if err != nil {
				return err
			}
			if end.ayah > count {
				return invalid(end.field, "That surah does not have that many ayahs.")
			}
		}

		var err error
		entry, err = q.RecordHifz(r.Context(), database.RecordHifzParams{
			TenantID: tenant.ID, StudentID: studentID, TeacherID: teacher, OnDate: onDate,
			Kind: kind, FromSurah: body.FromSurah, FromAyah: body.FromAyah,
			ToSurah: body.ToSurah, ToAyah: body.ToAyah, Quality: quality,
			Mistakes: body.Mistakes, Note: strings.TrimSpace(body.Note),
		})
		return err
	})
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, entry)
}

// studentHifz is one student's record. A learner may read their own, a
// guardian their child's, and staff anybody's in the school.
func (s *Server) studentHifz(w http.ResponseWriter, r *http.Request) error {
	studentID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	if err := s.mayReadAbout(r, studentID); err != nil {
		return err
	}

	var rows []database.ListHifzForStudentRow
	var summary database.HifzSummaryRow
	var surahs []database.Surah
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if rows, err = q.ListHifzForStudent(r.Context(), database.ListHifzForStudentParams{
			StudentID: studentID, PageLimit: limit, PageOffset: offset,
		}); err != nil {
			return err
		}
		if summary, err = q.HifzSummary(r.Context(), studentID); err != nil {
			return err
		}
		surahs, err = q.ListSurahs(r.Context())
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"entries": rows, "ayahs_memorised": summary.Ayahs, "lessons": summary.Lessons,
		"total_ayahs": 6236, "surahs": surahs,
	})
}

// hifzOnDate is the teacher's own page: everybody they heard on one day.
func (s *Server) hifzOnDate(w http.ResponseWriter, r *http.Request) error {
	onDate := pgtype.Date{Time: time.Now(), Valid: true}
	if raw := r.URL.Query().Get("on"); raw != "" {
		var err error
		if onDate, err = parseDay("on", raw); err != nil {
			return err
		}
	}
	var rows []database.HifzOnDateRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.HifzOnDate(r.Context(), onDate)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"on": onDate.Time.Format("2006-01-02"), "entries": rows,
	})
}

func (s *Server) deleteHifz(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteHifzEntry(r.Context(), id)
	})
}

// mayReadAbout gates one person's record: staff read anybody in their school,
// a learner reads their own, and a guardian reads their own child's. The
// guardianship is checked on every read rather than assumed from a role.
func (s *Server) mayReadAbout(r *http.Request, studentID uuid.UUID) error {
	caller := Authenticated(r.Context()).UserID
	if caller == studentID {
		return nil
	}
	switch CurrentRole(r.Context()) {
	case "owner", "admin", "instructor", "assistant":
		return nil
	}

	var guardian bool
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		guardian, err = q.IsGuardianOf(r.Context(), database.IsGuardianOfParams{
			GuardianID: caller, StudentID: studentID,
		})
		return err
	})
	if err != nil {
		return err
	}
	if !guardian {
		return httpx.ErrForbidden
	}
	return nil
}
