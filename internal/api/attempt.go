package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/assessment"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// myAttempt lets a learner resume: their own answers, never the answer key.
func (s *Server) myAttempt(w http.ResponseWriter, r *http.Request) error {
	attemptID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	userID := Authenticated(r.Context()).UserID
	var out attemptResponse
	var saved []savedAnswer

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		attempt, err := q.GetAttempt(r.Context(), attemptID)
		if err != nil {
			return err
		}
		if attempt.UserID != userID {
			return httpx.ErrNotFound
		}
		out.Attempt = attempt

		if err := s.fillPaper(r, q, &out); err != nil {
			return err
		}
		rows, err := q.ListAnswers(r.Context(), attemptID)
		if err != nil {
			return err
		}
		saved = make([]savedAnswer, 0, len(rows))
		for _, row := range rows {
			saved = append(saved, savedAnswer{
				QuestionID: row.QuestionID, OptionIDs: orEmpty(row.OptionIds), Text: row.TextAnswer,
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
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"attempt": out.Attempt, "questions": out.Questions,
		"expires_in_s": out.ExpiresIn, "answers": saved,
	})
}

type savedAnswer struct {
	QuestionID uuid.UUID   `json:"question_id"`
	OptionIDs  []uuid.UUID `json:"option_ids"`
	Text       string      `json:"text"`
}

type answerRequest struct {
	QuestionID string   `json:"question_id"`
	OptionIDs  []string `json:"option_ids"`
	Text       string   `json:"text"`
}

// saveAnswer stores one answer without grading it, so a learner can work
// through a paper and a phone can flush its queue when it reconnects.
func (s *Server) saveAnswer(w http.ResponseWriter, r *http.Request) error {
	attemptID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body answerRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	questionID, err := uuid.Parse(strings.TrimSpace(body.QuestionID))
	if err != nil {
		return invalid("question_id", "Name the question being answered.")
	}
	optionIDs, err := parseUUIDs(body.OptionIDs)
	if err != nil {
		return err
	}
	if len(body.Text) > 20000 {
		return invalid("text", "That answer is too long.")
	}

	tenant := CurrentTenant(r.Context())
	userID := Authenticated(r.Context()).UserID
	var answer database.AttemptAnswer

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		attempt, err := q.GetAttempt(r.Context(), attemptID)
		if err != nil {
			return err
		}
		if attempt.UserID != userID {
			return httpx.ErrNotFound
		}
		if attempt.State != database.AttemptStateInProgress {
			return httpx.Errorf(http.StatusConflict, "attempt_closed", "This attempt has been submitted.")
		}
		if expired(attempt) {
			return httpx.Errorf(http.StatusConflict, "attempt_expired", "Your time ran out.")
		}

		question, err := q.GetQuestion(r.Context(), questionID)
		if err != nil {
			return err
		}
		if question.QuizID != attempt.QuizID {
			return httpx.ErrNotFound
		}

		answer, err = q.SaveAnswer(r.Context(), database.SaveAnswerParams{
			TenantID: tenant.ID, AttemptID: attemptID, QuestionID: questionID,
			OptionIds: optionIDs, TextAnswer: strings.TrimSpace(body.Text),
			NeedsGrading: assessment.Kind(question.Kind).NeedsHuman(),
		})
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

type attemptResultResponse struct {
	Attempt   database.QuizAttempt `json:"attempt"`
	Percent   int                  `json:"percent"`
	Passed    bool                 `json:"passed"`
	Pending   bool                 `json:"awaiting_marking"`
	Breakdown []questionOutcome    `json:"breakdown,omitempty"`
}

type questionOutcome struct {
	QuestionID  uuid.UUID `json:"question_id"`
	Points      int32     `json:"points_awarded"`
	NeedsHuman  bool      `json:"awaiting_marking"`
	Explanation string    `json:"explanation,omitempty"`
}

// submitAttempt grades everything it can and leaves written answers for a
// marker. Grading reads the answer key from the database, never the request.
func (s *Server) submitAttempt(w http.ResponseWriter, r *http.Request) error {
	attemptID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	userID := Authenticated(r.Context()).UserID
	var out attemptResultResponse

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		attempt, err := q.GetAttempt(r.Context(), attemptID)
		if err != nil {
			return err
		}
		if attempt.UserID != userID {
			return httpx.ErrNotFound
		}
		if attempt.State != database.AttemptStateInProgress {
			return httpx.Errorf(http.StatusConflict, "attempt_closed", "This attempt has already been submitted.")
		}

		quiz, err := q.GetQuiz(r.Context(), attempt.QuizID)
		if err != nil {
			return err
		}
		questions, err := servedQuestions(r, q, attempt)
		if err != nil {
			return err
		}
		options, err := q.ListOptionsForQuiz(r.Context(), attempt.QuizID)
		if err != nil {
			return err
		}
		saved, err := q.ListAnswers(r.Context(), attemptID)
		if err != nil {
			return err
		}

		result := assessment.Grade(toQuestions(questions, options), toAnswers(saved))

		for _, verdict := range result.Verdicts {
			points := verdict.Points
			if err := q.GradeAnswer(r.Context(), database.GradeAnswerParams{
				AttemptID: attemptID, QuestionID: verdict.QuestionID,
				PointsAwarded: &points, NeedsGrading: verdict.NeedsHuman,
			}); err != nil {
				return err
			}
		}

		state := database.AttemptStateGraded
		if result.NeedsHuman {
			state = database.AttemptStateSubmitted
		}
		// A learner who ran out of time still gets what they earned.
		if expired(attempt) && !result.NeedsHuman {
			state = database.AttemptStateGraded
		}

		out.Attempt, err = q.FinishAttempt(r.Context(), database.FinishAttemptParams{
			ID: attemptID, State: state, PointsAwarded: result.PointsAwarded,
		})
		if err != nil {
			return err
		}

		out.Pending = result.NeedsHuman
		out.Percent = assessment.Percent(result.PointsAwarded, result.PointsPossible)
		out.Passed = !result.NeedsHuman && assessment.Passed(result.PointsAwarded, result.PointsPossible, int(quiz.PassPercent))

		// Paid once per quiz, however many attempts it took.
		if out.Passed {
			s.award(r.Context(), q, CurrentTenant(r.Context()), userID, "quiz", quiz.ID, pointsPerQuiz)
		}

		if quiz.RevealAnswers {
			explanations := make(map[uuid.UUID]string, len(questions))
			for _, question := range questions {
				explanations[question.ID] = question.Explanation
			}
			for _, verdict := range result.Verdicts {
				out.Breakdown = append(out.Breakdown, questionOutcome{
					QuestionID: verdict.QuestionID, Points: verdict.Points,
					NeedsHuman: verdict.NeedsHuman, Explanation: explanations[verdict.QuestionID],
				})
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
	return httpx.JSON(w, http.StatusOK, out)
}

func toQuestions(rows []database.Question, options []database.QuestionOption) []assessment.Question {
	correct := make(map[uuid.UUID][]uuid.UUID, len(rows))
	for _, o := range options {
		if o.IsCorrect {
			correct[o.QuestionID] = append(correct[o.QuestionID], o.ID)
		}
	}
	out := make([]assessment.Question, 0, len(rows))
	for _, row := range rows {
		out = append(out, assessment.Question{
			ID: row.ID, Kind: assessment.Kind(row.Kind), Points: row.Points, Correct: correct[row.ID],
		})
	}
	return out
}

func toAnswers(rows []database.AttemptAnswer) map[uuid.UUID]assessment.Answer {
	out := make(map[uuid.UUID]assessment.Answer, len(rows))
	for _, row := range rows {
		out[row.QuestionID] = assessment.Answer{
			QuestionID: row.QuestionID, OptionIDs: row.OptionIds, Text: row.TextAnswer,
		}
	}
	return out
}

func parseUUIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) > 50 {
		return nil, invalid("option_ids", "That is too many options.")
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, v := range raw {
		id, err := uuid.Parse(strings.TrimSpace(v))
		if err != nil {
			return nil, invalid("option_ids", "One of the selected options is not valid.")
		}
		out = append(out, id)
	}
	return out, nil
}
