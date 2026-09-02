package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/assessment"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type quizRequest struct {
	Title         string `json:"title"`
	Instructions  string `json:"instructions"`
	Dir           string `json:"dir"`
	TimeLimitS    int32  `json:"time_limit_s"`
	MaxAttempts   int16  `json:"max_attempts"`
	PassPercent   int16  `json:"pass_percent"`
	Shuffle       bool   `json:"shuffle"`
	RevealAnswers *bool  `json:"reveal_answers"`
}

func (s *Server) createQuiz(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body quizRequest
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
	if body.TimeLimitS < 0 || body.TimeLimitS > 24*3600 {
		return invalid("time_limit_s", "A time limit must be between 0 and 24 hours.")
	}
	if body.MaxAttempts == 0 {
		body.MaxAttempts = 1
	}
	if body.MaxAttempts < 1 || body.MaxAttempts > 100 {
		return invalid("max_attempts", "Allow between 1 and 100 attempts.")
	}
	if body.PassPercent < 0 || body.PassPercent > 100 {
		return invalid("pass_percent", "The pass mark must be between 0 and 100.")
	}
	reveal := body.RevealAnswers == nil || *body.RevealAnswers

	tenant := CurrentTenant(r.Context())
	var quiz database.Quiz
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		quiz, err = q.CreateQuiz(r.Context(), database.CreateQuizParams{
			TenantID: tenant.ID, LessonID: lessonID, Title: title,
			Instructions: strings.TrimSpace(body.Instructions), Dir: dir,
			TimeLimitS: body.TimeLimitS, MaxAttempts: body.MaxAttempts,
			PassPercent: body.PassPercent, Shuffle: body.Shuffle, RevealAnswers: reveal,
		})
		if err != nil {
			return err
		}

		courseID, err := q.LessonCourse(r.Context(), lessonID)
		if err != nil {
			return err
		}
		_, err = q.CreateGradeItem(r.Context(), database.CreateGradeItemParams{
			TenantID: tenant.ID, CourseID: courseID, Source: database.GradeSourceQuiz,
			QuizID: uuid.NullUUID{UUID: quiz.ID, Valid: true}, Title: title,
			Category: "quiz", PointsPossible: 1, Weight: 100,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "quiz_exists", "This lesson already has a quiz.")
	}
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, quiz)
}

type questionRequest struct {
	Kind        string          `json:"kind"`
	Prompt      string          `json:"prompt"`
	Dir         string          `json:"dir"`
	Points      int32           `json:"points"`
	Explanation string          `json:"explanation"`
	Options     []optionRequest `json:"options"`
}

type optionRequest struct {
	Label     string `json:"label"`
	IsCorrect bool   `json:"is_correct"`
}

// addQuestion writes a question and its options together, rejecting any shape
// that could not be graded.
func (s *Server) addQuestion(w http.ResponseWriter, r *http.Request) error {
	quizID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body questionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	kind := assessment.Kind(strings.TrimSpace(body.Kind))
	if !kind.Valid() {
		return invalid("kind", "Unknown question type.")
	}
	prompt, err := requireText("prompt", body.Prompt, 5000)
	if err != nil {
		return err
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	if body.Points == 0 {
		body.Points = 1
	}
	if body.Points < 1 || body.Points > 1000 {
		return invalid("points", "Points must be between 1 and 1000.")
	}
	if err := validateOptions(kind, body.Options); err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var (
		question database.Question
		options  []database.QuestionOption
	)
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		question, err = q.CreateQuestion(r.Context(), database.CreateQuestionParams{
			TenantID: tenant.ID, QuizID: quizID, Kind: database.QuestionKind(kind),
			Prompt: prompt, Dir: dir, Points: body.Points,
			Explanation: strings.TrimSpace(body.Explanation),
		})
		if err != nil {
			return err
		}
		for _, opt := range body.Options {
			created, err := q.CreateOption(r.Context(), database.CreateOptionParams{
				TenantID: tenant.ID, QuestionID: question.ID,
				Label: strings.TrimSpace(opt.Label), IsCorrect: opt.IsCorrect,
			})
			if err != nil {
				return err
			}
			options = append(options, created)
		}
		// Keep the gradebook column worth what the paper is worth.
		return q.SyncQuizItemPoints(r.Context(), quizID)
	})
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, map[string]any{"question": question, "options": options})
}

// validateOptions refuses questions that could never be answered correctly.
func validateOptions(kind assessment.Kind, options []optionRequest) error {
	if kind.NeedsHuman() {
		if len(options) > 0 {
			return invalid("options", "A written answer takes no options.")
		}
		return nil
	}

	if len(options) < 2 {
		return invalid("options", "Give at least two options.")
	}
	if len(options) > 20 {
		return invalid("options", "Twenty options is the limit.")
	}

	correct := 0
	for _, opt := range options {
		if strings.TrimSpace(opt.Label) == "" {
			return invalid("options", "Every option needs a label.")
		}
		if opt.IsCorrect {
			correct++
		}
	}
	switch {
	case correct == 0:
		return invalid("options", "Mark at least one option correct.")
	case kind != assessment.MCQMulti && correct > 1:
		return invalid("options", "This question type allows only one correct option.")
	case kind == assessment.TrueFalse && len(options) != 2:
		return invalid("options", "A true or false question has exactly two options.")
	}
	return nil
}

// learnerQuestion deliberately omits which option is correct.
type learnerQuestion struct {
	ID      uuid.UUID       `json:"id"`
	Kind    string          `json:"kind"`
	Prompt  string          `json:"prompt"`
	Dir     string          `json:"dir"`
	Points  int32           `json:"points"`
	Options []learnerOption `json:"options"`
}

type learnerOption struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label"`
}

// quizForLearner returns the paper without the answer key.
func (s *Server) quizForLearner(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	var (
		quiz      database.Quiz
		questions []learnerQuestion
		attempts  []database.QuizAttempt
	)
	userID := Authenticated(r.Context()).UserID

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if quiz, err = q.GetQuizByLesson(r.Context(), lessonID); err != nil {
			return err
		}
		questions, err = s.learnerPaper(r, q, quiz.ID)
		if err != nil {
			return err
		}
		attempts, err = q.ListMyAttempts(r.Context(), database.ListMyAttemptsParams{
			QuizID: quiz.ID, UserID: userID,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"quiz": quiz, "questions": questions, "attempts": attempts,
	})
}

// quizSheet is the staff view of a quiz: the same paper with the answer key,
// which the learner endpoint deliberately withholds.
func (s *Server) quizSheet(w http.ResponseWriter, r *http.Request) error {
	quizID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	type option struct {
		ID        uuid.UUID `json:"id"`
		Label     string    `json:"label"`
		IsCorrect bool      `json:"is_correct"`
	}
	type question struct {
		database.Question
		Options []option `json:"options"`
	}

	var (
		quiz      database.Quiz
		questions []question
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if quiz, err = q.GetQuiz(r.Context(), quizID); err != nil {
			return err
		}
		rows, err := q.ListQuestions(r.Context(), quizID)
		if err != nil {
			return err
		}
		options, err := q.ListOptionsForQuiz(r.Context(), quizID)
		if err != nil {
			return err
		}
		byQuestion := make(map[uuid.UUID][]option, len(rows))
		for _, o := range options {
			byQuestion[o.QuestionID] = append(byQuestion[o.QuestionID],
				option{ID: o.ID, Label: o.Label, IsCorrect: o.IsCorrect})
		}
		questions = make([]question, 0, len(rows))
		for _, row := range rows {
			opts := byQuestion[row.ID]
			if opts == nil {
				opts = []option{}
			}
			questions = append(questions, question{Question: row, Options: opts})
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"quiz": quiz, "questions": questions})
}

func (s *Server) deleteQuestion(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.DeleteQuestion(r.Context(), id)
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) learnerPaper(r *http.Request, q *database.Queries, quizID uuid.UUID) ([]learnerQuestion, error) {
	rows, err := q.ListQuestions(r.Context(), quizID)
	if err != nil {
		return nil, err
	}
	options, err := q.ListOptionsForQuiz(r.Context(), quizID)
	if err != nil {
		return nil, err
	}

	byQuestion := make(map[uuid.UUID][]learnerOption, len(rows))
	for _, o := range options {
		byQuestion[o.QuestionID] = append(byQuestion[o.QuestionID], learnerOption{ID: o.ID, Label: o.Label})
	}

	out := make([]learnerQuestion, 0, len(rows))
	for _, row := range rows {
		opts := byQuestion[row.ID]
		if opts == nil {
			opts = []learnerOption{}
		}
		out = append(out, learnerQuestion{
			ID: row.ID, Kind: string(row.Kind), Prompt: row.Prompt,
			Dir: string(row.Dir), Points: row.Points, Options: opts,
		})
	}
	return out, nil
}

type attemptResponse struct {
	Attempt   database.QuizAttempt `json:"attempt"`
	Questions []learnerQuestion    `json:"questions"`
	ExpiresIn int                  `json:"expires_in_s,omitempty"`
}

// startAttempt opens an attempt, enforcing the attempt limit server-side.
func (s *Server) startAttempt(w http.ResponseWriter, r *http.Request) error {
	quizID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	userID := Authenticated(r.Context()).UserID
	var out attemptResponse

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		quiz, err := q.GetQuiz(r.Context(), quizID)
		if err != nil {
			return err
		}
		lesson, err := q.GetLesson(r.Context(), quiz.LessonID)
		if err != nil {
			return err
		}
		courseID, err := q.LessonCourse(r.Context(), lesson.ID)
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

		// Resume rather than start again, so a dropped connection costs nothing.
		if open, err := q.OpenAttempt(r.Context(), database.OpenAttemptParams{
			QuizID: quizID, UserID: userID,
		}); err == nil {
			if expired(open) {
				return httpx.Errorf(http.StatusConflict, "attempt_expired",
					"Your time ran out. Submit to see your result.")
			}
			out.Attempt = open
			return s.fillPaper(r, q, quizID, &out)
		} else if !database.IsNotFound(err) {
			return err
		}

		used, err := q.CountAttempts(r.Context(), database.CountAttemptsParams{QuizID: quizID, UserID: userID})
		if err != nil {
			return err
		}
		if used >= int64(quiz.MaxAttempts) {
			return httpx.Errorf(http.StatusConflict, "no_attempts_left",
				"You have used all your attempts at this quiz.")
		}

		questions, err := q.ListQuestions(r.Context(), quizID)
		if err != nil {
			return err
		}
		if len(questions) == 0 {
			return httpx.Errorf(http.StatusConflict, "quiz_empty", "This quiz has no questions yet.")
		}
		var possible int32
		for _, question := range questions {
			possible += question.Points
		}

		out.Attempt, err = q.StartAttempt(r.Context(), database.StartAttemptParams{
			TenantID: tenant.ID, QuizID: quizID, EnrollmentID: enrollment.ID, UserID: userID,
			AttemptNo: int16(used + 1), TimeLimitS: quiz.TimeLimitS, PointsPossible: possible,
		})
		if err != nil {
			return err
		}
		return s.fillPaper(r, q, quizID, &out)
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, out)
}

func (s *Server) fillPaper(r *http.Request, q *database.Queries, quizID uuid.UUID, out *attemptResponse) error {
	questions, err := s.learnerPaper(r, q, quizID)
	if err != nil {
		return err
	}
	out.Questions = questions
	if out.Attempt.ExpiresAt.Valid {
		out.ExpiresIn = max(0, int(time.Until(out.Attempt.ExpiresAt.Time).Seconds()))
	}
	return nil
}

func expired(a database.QuizAttempt) bool {
	return a.ExpiresAt.Valid && a.ExpiresAt.Time.Before(time.Now())
}
