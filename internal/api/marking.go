package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/assessment"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

func (s *Server) markingQueue(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var rows []database.ListAttemptsForMarkingRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListAttemptsForMarking(r.Context(), database.ListAttemptsForMarkingParams{
			PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"attempts": rows})
}

// markerSheet is the teacher's view: the answer key is included here, and only here.
type markerSheet struct {
	Attempt   database.QuizAttempt `json:"attempt"`
	Questions []markerQuestion     `json:"questions"`
	Pending   int64                `json:"pending"`
}

type markerQuestion struct {
	database.Question
	CorrectOptionIDs []uuid.UUID `json:"correct_option_ids"`
	OptionIDs        []uuid.UUID `json:"answer_option_ids"`
	TextAnswer       string      `json:"text_answer"`
	PointsAwarded    *int32      `json:"points_awarded"`
	NeedsGrading     bool        `json:"needs_grading"`
	Feedback         string      `json:"feedback"`
}

func (s *Server) attemptSheet(w http.ResponseWriter, r *http.Request) error {
	attemptID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	var sheet markerSheet
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		attempt, err := q.GetAttempt(r.Context(), attemptID)
		if err != nil {
			return err
		}
		rows, err := q.AttemptSheet(r.Context(), database.AttemptSheetParams{
			AttemptID: attemptID, QuizID: attempt.QuizID,
		})
		if err != nil {
			return err
		}
		options, err := q.ListOptionsForQuiz(r.Context(), attempt.QuizID)
		if err != nil {
			return err
		}
		totals, err := q.SumAwardedPoints(r.Context(), attemptID)
		if err != nil {
			return err
		}

		correct := make(map[uuid.UUID][]uuid.UUID, len(rows))
		for _, o := range options {
			if o.IsCorrect {
				correct[o.QuestionID] = append(correct[o.QuestionID], o.ID)
			}
		}

		sheet.Attempt, sheet.Pending = attempt, totals.Pending
		sheet.Questions = make([]markerQuestion, 0, len(rows))
		for _, row := range rows {
			sheet.Questions = append(sheet.Questions, markerQuestion{
				Question: row.Question, CorrectOptionIDs: orEmpty(correct[row.Question.ID]),
				OptionIDs: orEmpty(row.OptionIds), TextAnswer: deref(row.TextAnswer),
				PointsAwarded: row.PointsAwarded, NeedsGrading: deref(row.NeedsGrading),
				Feedback: deref(row.Feedback),
			})
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, sheet)
}

type markRequest struct {
	Points   *int32 `json:"points_awarded"`
	Feedback string `json:"feedback"`
}

// markAnswer records a teacher's score for one written answer, capped at what
// the question is worth so a slip cannot push a learner over 100 percent.
func (s *Server) markAnswer(w http.ResponseWriter, r *http.Request) error {
	attemptID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	questionID, err := pathUUID(r, "questionId")
	if err != nil {
		return err
	}
	var body markRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.Points == nil {
		return invalid("points_awarded", "Give the points this answer earned.")
	}
	if *body.Points < 0 {
		return invalid("points_awarded", "Points cannot be negative.")
	}
	if len(body.Feedback) > 5000 {
		return invalid("feedback", "Keep feedback under 5000 characters.")
	}

	marker := Authenticated(r.Context()).UserID
	var answer database.AttemptAnswer

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		attempt, err := q.GetAttempt(r.Context(), attemptID)
		if err != nil {
			return err
		}
		if attempt.State != database.AttemptStateSubmitted {
			return httpx.Errorf(http.StatusConflict, "attempt_not_markable",
				"Only a submitted attempt can be marked.")
		}
		question, err := q.GetQuestion(r.Context(), questionID)
		if err != nil {
			return err
		}
		if question.QuizID != attempt.QuizID {
			return httpx.ErrNotFound
		}
		if *body.Points > question.Points {
			return invalid("points_awarded", "That is more than the question is worth.")
		}

		answer, err = q.MarkAnswer(r.Context(), database.MarkAnswerParams{
			AttemptID: attemptID, QuestionID: questionID, PointsAwarded: body.Points,
			Feedback: strings.TrimSpace(body.Feedback),
			GradedBy: uuid.NullUUID{UUID: marker, Valid: true},
		})
		if database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "not_answered", "The learner left this question blank.")
		}
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, answer)
}

type finalizeResponse struct {
	Attempt database.QuizAttempt `json:"attempt"`
	Percent int                  `json:"percent"`
	Passed  bool                 `json:"passed"`
}

// releaseAttempt totals the marks and hands the result back to the learner.
func (s *Server) releaseAttempt(w http.ResponseWriter, r *http.Request) error {
	attemptID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	var out finalizeResponse
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		attempt, err := q.GetAttempt(r.Context(), attemptID)
		if err != nil {
			return err
		}
		if attempt.State != database.AttemptStateSubmitted {
			return httpx.Errorf(http.StatusConflict, "attempt_not_markable",
				"Only a submitted attempt can be released.")
		}
		totals, err := q.SumAwardedPoints(r.Context(), attemptID)
		if err != nil {
			return err
		}
		if totals.Pending > 0 {
			return httpx.Errorf(http.StatusConflict, "marking_incomplete",
				"Some answers are still waiting to be marked.")
		}
		quiz, err := q.GetQuiz(r.Context(), attempt.QuizID)
		if err != nil {
			return err
		}

		out.Attempt, err = q.FinalizeAttempt(r.Context(), database.FinalizeAttemptParams{
			ID: attemptID, PointsAwarded: totals.Total,
		})
		if err != nil {
			return err
		}
		out.Percent = percentOf(totals.Total, out.Attempt.PointsPossible)
		out.Passed = out.Percent >= int(quiz.PassPercent) && out.Attempt.PointsPossible > 0
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	s.notifyUser(r.Context(), CurrentTenant(r.Context()).ID, out.Attempt.UserID, "quiz.result",
		"Your quiz result is ready",
		fmt.Sprintf("You scored %d%%.", out.Percent),
		map[string]any{"attempt_id": out.Attempt.ID, "percent": out.Percent, "passed": out.Passed})
	return httpx.JSON(w, http.StatusOK, out)
}

func percentOf(awarded, possible int32) int { return assessment.Percent(awarded, possible) }

// deref reads a column that is null when the learner skipped the question.
func deref[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}

func orEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}
