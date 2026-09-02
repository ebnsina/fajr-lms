package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/assessment"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/media"
)

type assignmentRequest struct {
	Title        string     `json:"title"`
	Instructions string     `json:"instructions"`
	Dir          string     `json:"dir"`
	Points       int32      `json:"points"`
	DueAt        *time.Time `json:"due_at"`
	AllowLate    *bool      `json:"allow_late"`
	LatePenalty  int16      `json:"late_penalty"`
	MaxFiles     *int16     `json:"max_files"`
}

func (s *Server) createAssignment(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body assignmentRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	if body.Points == 0 {
		body.Points = 100
	}
	if body.Points < 1 || body.Points > 100000 {
		return invalid("points", "Points must be between 1 and 100000.")
	}
	if body.LatePenalty < 0 || body.LatePenalty > 100 {
		return invalid("late_penalty", "A late penalty is a percentage between 0 and 100.")
	}
	maxFiles := int16(5)
	if body.MaxFiles != nil {
		if *body.MaxFiles < 0 || *body.MaxFiles > 20 {
			return invalid("max_files", "Allow between 0 and 20 files.")
		}
		maxFiles = *body.MaxFiles
	}

	tenant := CurrentTenant(r.Context())
	var assignment database.Assignment
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		assignment, err = q.CreateAssignment(r.Context(), database.CreateAssignmentParams{
			TenantID: tenant.ID, LessonID: lessonID, Title: title,
			Instructions: strings.TrimSpace(body.Instructions), Dir: dir, Points: body.Points,
			DueAt: timestamp(body.DueAt), AllowLate: body.AllowLate == nil || *body.AllowLate,
			LatePenalty: body.LatePenalty, MaxFiles: maxFiles,
		})
		if err != nil {
			return err
		}

		courseID, err := q.LessonCourse(r.Context(), lessonID)
		if err != nil {
			return err
		}
		_, err = q.CreateGradeItem(r.Context(), database.CreateGradeItemParams{
			TenantID: tenant.ID, CourseID: courseID, Source: database.GradeSourceAssignment,
			AssignmentID: uuid.NullUUID{UUID: assignment.ID, Valid: true}, Title: title,
			Category: "assignment", PointsPossible: assignment.Points, Weight: 100,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "assignment_exists", "This lesson already has an assignment.")
	}
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, assignment)
}

// assignmentForLearner returns the brief together with the learner's own work.
// updateAssignment applies only the fields present, so a teacher can move a
// due date without restating the brief.
func (s *Server) updateAssignment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body assignmentRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	params := database.UpdateAssignmentParams{ID: id}
	if raw := strings.TrimSpace(body.Title); raw != "" {
		title, err := requireText("title", raw, 200)
		if err != nil {
			return err
		}
		params.Title = &title
	}
	if body.Instructions != "" {
		instructions := strings.TrimSpace(body.Instructions)
		params.Instructions = &instructions
	}
	if body.DueAt != nil {
		params.DueAt = pgtype.Timestamptz{Time: *body.DueAt, Valid: true}
	}
	if body.AllowLate != nil {
		params.AllowLate = body.AllowLate
	}
	if body.LatePenalty < 0 || body.LatePenalty > 100 {
		return invalid("late_penalty", "A penalty is between 0 and 100 percent.")
	}
	params.LatePenalty = &body.LatePenalty

	var assignment database.Assignment
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		assignment, err = q.UpdateAssignment(r.Context(), params)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, assignment)
}

func (s *Server) assignmentForLearner(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	userID := Authenticated(r.Context()).UserID
	var (
		assignment database.Assignment
		submission *database.Submission
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if assignment, err = q.GetAssignmentByLesson(r.Context(), lessonID); err != nil {
			return err
		}
		mine, err := q.MySubmission(r.Context(), database.MySubmissionParams{
			AssignmentID: assignment.ID, UserID: userID,
		})
		if database.IsNotFound(err) {
			return nil
		}
		submission = &mine
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"assignment": assignment, "submission": submission,
	})
}

type submissionRequest struct {
	Body     string   `json:"body"`
	MediaIDs []string `json:"media_ids"`
	Submit   bool     `json:"submit"`
}

// submitWork saves a draft or hands work in. Lateness is decided here, from the
// server clock, so a device with a wrong date cannot beat a deadline.
func (s *Server) submitWork(w http.ResponseWriter, r *http.Request) error {
	assignmentID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body submissionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	mediaIDs, err := parseUUIDs(body.MediaIDs)
	if err != nil {
		return invalid("media_ids", "One of the attachments is not valid.")
	}
	if len(body.Body) > 100000 {
		return invalid("body", "That answer is too long.")
	}

	tenant := CurrentTenant(r.Context())
	userID := Authenticated(r.Context()).UserID
	var submission database.Submission

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		assignment, err := q.GetAssignment(r.Context(), assignmentID)
		if err != nil {
			return err
		}
		if int16(len(mediaIDs)) > assignment.MaxFiles {
			return invalid("media_ids", "That is more files than this assignment allows.")
		}
		if body.Submit && strings.TrimSpace(body.Body) == "" && len(mediaIDs) == 0 {
			return invalid("body", "Write something or attach a file before handing in.")
		}

		courseID, err := q.LessonCourse(r.Context(), assignment.LessonID)
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

		if existing, err := q.MySubmission(r.Context(), database.MySubmissionParams{
			AssignmentID: assignmentID, UserID: userID,
		}); err == nil && existing.State == database.SubmissionStateReturned {
			return httpx.Errorf(http.StatusConflict, "already_marked",
				"This work has been marked. Ask your teacher to reopen it.")
		} else if err != nil && !database.IsNotFound(err) {
			return err
		}

		late := assignment.DueAt.Valid && time.Now().After(assignment.DueAt.Time)
		if body.Submit && late && !assignment.AllowLate {
			return httpx.Errorf(http.StatusConflict, "past_due", "The deadline for this assignment has passed.")
		}

		state := database.SubmissionStateDraft
		if body.Submit {
			state = database.SubmissionStateSubmitted
		}
		submission, err = q.UpsertSubmission(r.Context(), database.UpsertSubmissionParams{
			TenantID: tenant.ID, AssignmentID: assignmentID, EnrollmentID: enrollment.ID,
			UserID: userID, State: state, Body: body.Body, MediaIds: mediaIDs,
			IsLate: late && body.Submit,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if isForeignKeyViolation(err) {
		return invalid("media_ids", "One of the attachments does not exist.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, submission)
}

func (s *Server) submissionQueue(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var rows []database.ListSubmissionsToGradeRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListSubmissionsToGrade(r.Context(), database.ListSubmissionsToGradeParams{
			PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"submissions": rows})
}

// submissionSheet is what a marker needs: the work, the brief, and playable
// links for whatever was attached.
type submissionSheet struct {
	Submission  database.Submission `json:"submission"`
	Assignment  database.Assignment `json:"assignment"`
	FullName    string              `json:"full_name"`
	Attachments []attachmentView    `json:"attachments"`
}

type attachmentView struct {
	MediaID uuid.UUID       `json:"media_id"`
	Kind    string          `json:"kind"`
	Title   string          `json:"title"`
	State   string          `json:"state"`
	Play    *media.Playback `json:"playback,omitempty"`
}

func (s *Server) submissionSheet(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var (
		row    database.SubmissionForMarkingRow
		assets []database.MediaAsset
	)
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		if row, err = q.SubmissionForMarking(r.Context(), id); err != nil {
			return err
		}
		for _, mediaID := range row.Submission.MediaIds {
			asset, err := q.GetMediaAsset(r.Context(), mediaID)
			if database.IsNotFound(err) {
				continue // deleted since it was attached
			}
			if err != nil {
				return err
			}
			assets = append(assets, asset)
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	viewer := media.Viewer{UserID: Authenticated(r.Context()).UserID.String(), TenantID: tenant.ID.String()}
	out := submissionSheet{
		Submission: row.Submission, Assignment: row.Assignment, FullName: row.FullName,
		Attachments: make([]attachmentView, 0, len(assets)),
	}
	for _, asset := range assets {
		view := attachmentView{
			MediaID: asset.ID, Kind: string(asset.Kind),
			Title: asset.Title, State: string(asset.State),
		}
		if provider, err := s.media.Get(asset.Provider); err == nil {
			if play, err := provider.Playback(r.Context(), media.Asset{
				ID: asset.ID.String(), TenantID: asset.TenantID.String(), ExternalRef: asset.ExternalRef,
				State: media.State(asset.State), ContentType: asset.ContentType,
			}, viewer); err == nil {
				view.Play = &play
			}
		}
		out.Attachments = append(out.Attachments, view)
	}
	return httpx.JSON(w, http.StatusOK, out)
}

type gradeSubmissionRequest struct {
	Points   *int32 `json:"points_awarded"`
	Feedback string `json:"feedback"`
}

type gradedSubmission struct {
	database.Submission
	PenaltyApplied int32 `json:"late_penalty_applied"`
}

// gradeWork records a mark, applying the late penalty once so the learner sees
// a single number rather than a raw score they have to adjust themselves.
func (s *Server) gradeWork(w http.ResponseWriter, r *http.Request) error {
	submissionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body gradeSubmissionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.Points == nil {
		return invalid("points_awarded", "Give the points this work earned.")
	}
	if *body.Points < 0 {
		return invalid("points_awarded", "Points cannot be negative.")
	}
	if len(body.Feedback) > 10000 {
		return invalid("feedback", "Keep feedback under 10000 characters.")
	}

	marker := Authenticated(r.Context()).UserID
	var out gradedSubmission

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		current, err := q.GetSubmission(r.Context(), submissionID)
		if err != nil {
			return err
		}
		assignment, err := q.GetAssignment(r.Context(), current.AssignmentID)
		if err != nil {
			return err
		}
		if *body.Points > assignment.Points {
			return invalid("points_awarded", "That is more than the assignment is worth.")
		}

		awarded := *body.Points
		if current.IsLate {
			reduced := assessment.LatePenalty(awarded, int(assignment.LatePenalty))
			out.PenaltyApplied, awarded = awarded-reduced, reduced
		}

		out.Submission, err = q.GradeSubmission(r.Context(), database.GradeSubmissionParams{
			ID: submissionID, PointsAwarded: &awarded, Feedback: strings.TrimSpace(body.Feedback),
			GradedBy: uuid.NullUUID{UUID: marker, Valid: true},
		})
		if database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "not_submitted",
				"Only work that has been handed in can be marked.")
		}
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	s.notifyUser(r.Context(), CurrentTenant(r.Context()).ID, out.UserID, "assignment.marked",
		"Your work has been marked",
		fmt.Sprintf("You scored %d.", derefInt(out.PointsAwarded)),
		map[string]any{"submission_id": out.ID, "points": out.PointsAwarded})
	return httpx.JSON(w, http.StatusOK, out)
}

func derefInt(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

func timestamp(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
