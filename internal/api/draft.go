package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ebnsina/fajr-lms/internal/ai"
	"github.com/ebnsina/fajr-lms/internal/assessment"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// draftQuestions suggests questions from what a teacher wrote in the lesson.
// Nothing is saved: the teacher reads the draft, changes it, and adds what they
// want. A quiz is a judgement about a class, not something to hand to a model.
func (s *Server) draftQuestions(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	count := 5
	if raw := r.URL.Query().Get("count"); raw != "" {
		count, err = strconv.Atoi(raw)
		if err != nil || count < 1 || count > 20 {
			return invalid("count", "Ask for between 1 and 20 questions.")
		}
	}

	var lesson database.Lesson
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		lesson, err = q.GetLesson(r.Context(), lessonID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if len(lesson.Body) < 200 {
		return httpx.Errorf(http.StatusConflict, "lesson_too_short",
			"There is not enough written in this lesson to ask questions about yet.")
	}

	drafted, err := s.ai.DraftQuestions(r.Context(), ai.Lesson{
		Title: lesson.Title, Body: lesson.Body, Dir: string(lesson.Dir),
	}, count)
	if errors.Is(err, ai.ErrOff) {
		return httpx.Errorf(http.StatusServiceUnavailable, "ai_off",
			"Fajr AI is not switched on for this installation.")
	}
	if err != nil {
		return httpx.Errorf(http.StatusBadGateway, "ai_failed", err.Error())
	}

	// A draft that could not be graded is worse than no draft, so anything the
	// quiz builder would refuse is dropped here rather than shown.
	kept := make([]ai.Question, 0, len(drafted))
	for _, question := range drafted {
		if usable(question) {
			kept = append(kept, question)
		}
	}
	if len(kept) == 0 {
		return httpx.Errorf(http.StatusBadGateway, "ai_unusable",
			"The draft came back in a shape we could not use. Try again.")
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"questions": kept})
}

func usable(q ai.Question) bool {
	kind := assessment.Kind(q.Kind)
	if !kind.Valid() || kind.NeedsHuman() || q.Prompt == "" || q.Points < 1 || q.Points > 5 {
		return false
	}
	correct := 0
	for _, option := range q.Options {
		if option.Label == "" {
			return false
		}
		if option.IsCorrect {
			correct++
		}
	}
	switch kind {
	case assessment.MCQSingle:
		return len(q.Options) >= 3 && correct == 1
	case assessment.TrueFalse:
		return len(q.Options) == 2 && correct == 1
	case assessment.MCQMulti:
		return len(q.Options) >= 3 && correct >= 2
	}
	return false
}
