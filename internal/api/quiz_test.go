package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type paperResponse struct {
	Quiz struct {
		ID          string `json:"id"`
		MaxAttempts int    `json:"max_attempts"`
	} `json:"quiz"`
	Questions []struct {
		ID      string `json:"id"`
		Kind    string `json:"kind"`
		Prompt  string `json:"prompt"`
		Points  int32  `json:"points"`
		Options []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"options"`
	} `json:"questions"`
}

// seedQuiz builds a two-question quiz on a published lesson and enrolls student.
func seedQuiz(t *testing.T, h http.Handler, owner, student actor, maxAttempts int) (lessonID, quizID string) {
	t.Helper()
	courseID := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
		map[string]any{"title": "Fiqh", "visibility": "public"}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/modules", owner.token, owner.slug,
		map[string]any{"title": "Unit"}))
	lessonID = createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", owner.token, owner.slug,
		map[string]any{"title": "Quiz lesson", "kind": "quiz"}))
	if rec := do(t, h, "PATCH", "/v1/lessons/"+lessonID, owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish lesson: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/v1/courses/"+courseID+"/status", owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish course: got %d: %s", rec.Code, rec.Body)
	}

	quizID = createdID(t, do(t, h, "POST", "/v1/lessons/"+lessonID+"/quiz", owner.token, owner.slug,
		map[string]any{"title": "اختبار قصير", "dir": "rtl", "max_attempts": maxAttempts, "pass_percent": 50}))

	if rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/questions", owner.token, owner.slug, map[string]any{
		"kind": "mcq_single", "prompt": "كم عدد أركان الإسلام؟", "points": 2, "explanation": "خمسة",
		"options": []map[string]any{
			{"label": "أربعة"}, {"label": "خمسة", "is_correct": true}, {"label": "ستة"},
		},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("add question: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/questions", owner.token, owner.slug, map[string]any{
		"kind": "mcq_multi", "prompt": "Pick the two correct", "points": 4,
		"options": []map[string]any{
			{"label": "A", "is_correct": true}, {"label": "B", "is_correct": true},
			{"label": "C"}, {"label": "D"},
		},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("add question: got %d: %s", rec.Code, rec.Body)
	}

	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
		t.Fatalf("enroll: got %d: %s", rec.Code, rec.Body)
	}
	return lessonID, quizID
}

func readPaper(t *testing.T, h http.Handler, a actor, lessonID string) paperResponse {
	t.Helper()
	rec := do(t, h, "GET", "/v1/lessons/"+lessonID+"/quiz", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("read quiz: got %d: %s", rec.Code, rec.Body)
	}
	var paper paperResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &paper); err != nil {
		t.Fatalf("decode paper: %v", err)
	}
	return paper
}

func TestQuizAuthoringValidation(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	lessonID, quizID := seedQuiz(t, h, owner, student, 2)

	t.Run("a lesson has at most one quiz", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/lessons/"+lessonID+"/quiz", owner.token, owner.slug,
			map[string]any{"title": "Second"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a student cannot author", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/questions", student.token, owner.slug,
			map[string]any{"kind": "mcq_single", "prompt": "x", "options": []map[string]any{{"label": "a", "is_correct": true}, {"label": "b"}}})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403", rec.Code)
		}
	})

	t.Run("unanswerable questions are refused", func(t *testing.T) {
		cases := map[string]map[string]any{
			"no correct option": {"kind": "mcq_single", "prompt": "p", "options": []map[string]any{{"label": "a"}, {"label": "b"}}},
			"only one option":   {"kind": "mcq_single", "prompt": "p", "options": []map[string]any{{"label": "a", "is_correct": true}}},
			"two correct on single": {"kind": "mcq_single", "prompt": "p", "options": []map[string]any{
				{"label": "a", "is_correct": true}, {"label": "b", "is_correct": true}}},
			"three-way true/false": {"kind": "true_false", "prompt": "p", "options": []map[string]any{
				{"label": "t", "is_correct": true}, {"label": "f"}, {"label": "maybe"}}},
			"essay with options": {"kind": "essay", "prompt": "p", "options": []map[string]any{
				{"label": "a", "is_correct": true}, {"label": "b"}}},
			"blank option label": {"kind": "mcq_single", "prompt": "p", "options": []map[string]any{
				{"label": "  ", "is_correct": true}, {"label": "b"}}},
			"unknown kind": {"kind": "matching", "prompt": "p"},
			"blank prompt": {"kind": "essay", "prompt": "   "},
		}
		for name, body := range cases {
			rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/questions", owner.token, owner.slug, body)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("%s: got %d, want 422: %s", name, rec.Code, rec.Body)
			}
		}
	})
}

func TestQuizNeverLeaksAnswers(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	lessonID, _ := seedQuiz(t, h, owner, student, 2)

	rec := do(t, h, "GET", "/v1/lessons/"+lessonID+"/quiz", student.token, owner.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	// The answer key must not appear anywhere in the payload.
	for _, leak := range []string{"is_correct", "explanation"} {
		if bytesContain(rec.Body.Bytes(), leak) {
			t.Errorf("the learner's paper contains %q: %s", leak, rec.Body)
		}
	}

	paper := readPaper(t, h, student, lessonID)
	if len(paper.Questions) != 2 || len(paper.Questions[0].Options) != 3 {
		t.Fatalf("got %+v", paper.Questions)
	}
}

func bytesContain(haystack []byte, needle string) bool {
	return len(haystack) >= len(needle) && json.Valid(haystack) &&
		stringsContains(string(haystack), needle)
}

func stringsContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestQuizSheetIsStaffOnly(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	_, quizID := seedQuiz(t, h, owner, student, 2)

	rec := do(t, h, "GET", "/v1/quizzes/"+quizID, owner.token, owner.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("read sheet: got %d: %s", rec.Code, rec.Body)
	}
	var sheet struct {
		Questions []struct {
			Prompt  string `json:"prompt"`
			Options []struct {
				Label     string `json:"label"`
				IsCorrect bool   `json:"is_correct"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sheet); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sheet.Questions) != 2 {
		t.Fatalf("want 2 questions, got %d", len(sheet.Questions))
	}
	correct := 0
	for _, option := range sheet.Questions[0].Options {
		if option.IsCorrect {
			correct++
		}
	}
	if correct != 1 {
		t.Errorf("the answer key did not come back: %+v", sheet.Questions[0].Options)
	}

	t.Run("a student cannot read the answer key", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/quizzes/"+quizID, student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a question can be removed", func(t *testing.T) {
		questionID := ""
		rec := do(t, h, "GET", "/v1/quizzes/"+quizID, owner.token, owner.slug, nil)
		var listed struct {
			Questions []struct {
				ID string `json:"id"`
			} `json:"questions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
			t.Fatalf("decode: %v", err)
		}
		questionID = listed.Questions[1].ID
		if rec := do(t, h, "DELETE", "/v1/questions/"+questionID, owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("delete: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "DELETE", "/v1/questions/"+questionID, owner.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Fatalf("second delete: got %d, want 404: %s", rec.Code, rec.Body)
		}
	})
}
