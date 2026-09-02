package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/assessment"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type gradeItemRequest struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Possible int32  `json:"points_possible"`
	Weight   *int32 `json:"weight"`
}

func (s *Server) createGradeItem(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body gradeItemRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	if body.Possible < 1 || body.Possible > 100000 {
		return invalid("points_possible", "Points must be between 1 and 100000.")
	}
	weight := int32(100)
	if body.Weight != nil {
		if *body.Weight < 0 || *body.Weight > 10000 {
			return invalid("weight", "Weight must be between 0 and 10000.")
		}
		weight = *body.Weight
	}

	tenant := CurrentTenant(r.Context())
	var item database.GradeItem
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		if _, err := q.GetCourse(r.Context(), courseID); err != nil {
			return err
		}
		var err error
		item, err = q.CreateGradeItem(r.Context(), database.CreateGradeItemParams{
			TenantID: tenant.ID, CourseID: courseID, Source: database.GradeSourceManual,
			Title: title, Category: strings.TrimSpace(body.Category),
			PointsPossible: body.Possible, Weight: weight,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, item)
}

// gradebook returns every learner against every item, in one pass.
func (s *Server) gradebook(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	var (
		items   []database.GradeItem
		reports []assessment.Report
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		items, reports, err = s.buildGradebook(r, q, courseID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "learners": reports})
}

// myGrades is the same computation narrowed to the caller.
func (s *Server) myGrades(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	userID := Authenticated(r.Context()).UserID
	var (
		items  []database.GradeItem
		report assessment.Report
		found  bool
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
			CourseID: courseID, UserID: userID,
		})
		if err != nil {
			return err
		}
		reports := []assessment.Report(nil)
		if items, reports, err = s.buildGradebook(r, q, courseID); err != nil {
			return err
		}
		for _, candidate := range reports {
			if candidate.EnrollmentID == enrollment.ID {
				report, found = candidate, true
			}
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !found {
		return httpx.ErrNotFound
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "grades": report})
}

func (s *Server) buildGradebook(r *http.Request, q *database.Queries, courseID uuid.UUID) ([]database.GradeItem, []assessment.Report, error) {
	if _, err := q.GetCourse(r.Context(), courseID); err != nil {
		return nil, nil, err
	}
	items, err := q.ListGradeItems(r.Context(), courseID)
	if err != nil {
		return nil, nil, err
	}
	learners, err := q.ListCourseEnrollments(r.Context(), courseID)
	if err != nil {
		return nil, nil, err
	}
	quizScores, err := q.BestQuizScores(r.Context(), courseID)
	if err != nil {
		return nil, nil, err
	}
	workScores, err := q.BestAssignmentScores(r.Context(), courseID)
	if err != nil {
		return nil, nil, err
	}
	overrides, err := q.ListCourseOverrides(r.Context(), courseID)
	if err != nil {
		return nil, nil, err
	}
	computed := append(toQuizScores(quizScores), toAssignmentScores(workScores)...)
	return items, assessment.BuildGradebook(
		toItems(items), toLearners(learners), computed, toOverrides(overrides),
	), nil
}

type overrideRequest struct {
	Points *int32 `json:"points"`
	Note   string `json:"note"`
}

// setGrade records a teacher's score, which always wins over a computed one.
func (s *Server) setGrade(w http.ResponseWriter, r *http.Request) error {
	itemID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	enrollmentID, err := pathUUID(r, "enrollmentId")
	if err != nil {
		return err
	}
	var body overrideRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.Points == nil {
		return invalid("points", "Give the points to record.")
	}
	if *body.Points < 0 {
		return invalid("points", "Points cannot be negative.")
	}
	if len(body.Note) > 2000 {
		return invalid("note", "Keep the note under 2000 characters.")
	}

	tenant := CurrentTenant(r.Context())
	var override database.GradeOverride
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		item, err := q.GetGradeItem(r.Context(), itemID)
		if err != nil {
			return err
		}
		if *body.Points > item.PointsPossible {
			return invalid("points", "That is more than the item is worth.")
		}
		override, err = q.SetGradeOverride(r.Context(), database.SetGradeOverrideParams{
			TenantID: tenant.ID, GradeItemID: itemID, EnrollmentID: enrollmentID,
			Points: *body.Points, Note: strings.TrimSpace(body.Note),
			SetBy: uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
		})
		return err
	})
	if database.IsNotFound(err) || isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, override)
}

// clearGrade removes a teacher's score so the computed one applies again.
func (s *Server) clearGrade(w http.ResponseWriter, r *http.Request) error {
	itemID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	enrollmentID, err := pathUUID(r, "enrollmentId")
	if err != nil {
		return err
	}

	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ClearGradeOverride(r.Context(), database.ClearGradeOverrideParams{
			GradeItemID: itemID, EnrollmentID: enrollmentID,
		})
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

func toItems(rows []database.GradeItem) []assessment.Item {
	out := make([]assessment.Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, assessment.Item{
			ID: row.ID, QuizID: row.QuizID.UUID, AssignmentID: row.AssignmentID.UUID, Title: row.Title,
			Category: row.Category, Possible: row.PointsPossible, Weight: row.Weight,
		})
	}
	return out
}

func toLearners(rows []database.ListCourseEnrollmentsRow) []assessment.Learner {
	out := make([]assessment.Learner, 0, len(rows))
	for _, row := range rows {
		out = append(out, assessment.Learner{EnrollmentID: row.ID, FullName: row.FullName})
	}
	return out
}

func toQuizScores(rows []database.BestQuizScoresRow) []assessment.SourceScore {
	out := make([]assessment.SourceScore, 0, len(rows))
	for _, row := range rows {
		out = append(out, assessment.SourceScore{
			SourceID: row.QuizID, EnrollmentID: row.EnrollmentID, Points: row.PointsAwarded,
		})
	}
	return out
}

func toAssignmentScores(rows []database.BestAssignmentScoresRow) []assessment.SourceScore {
	out := make([]assessment.SourceScore, 0, len(rows))
	for _, row := range rows {
		if row.PointsAwarded == nil {
			continue
		}
		out = append(out, assessment.SourceScore{
			SourceID: row.AssignmentID, EnrollmentID: row.EnrollmentID, Points: *row.PointsAwarded,
		})
	}
	return out
}

func toOverrides(rows []database.GradeOverride) []assessment.Override {
	out := make([]assessment.Override, 0, len(rows))
	for _, row := range rows {
		out = append(out, assessment.Override{
			ItemID: row.GradeItemID, EnrollmentID: row.EnrollmentID, Points: row.Points, Note: row.Note,
		})
	}
	return out
}
