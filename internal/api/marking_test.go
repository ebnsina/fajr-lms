package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type sheetResponse struct {
	Attempt struct {
		ID             string `json:"id"`
		State          string `json:"state"`
		PointsPossible int32  `json:"points_possible"`
	} `json:"attempt"`
	Questions []struct {
		ID               string   `json:"id"`
		Kind             string   `json:"kind"`
		Points           int32    `json:"points"`
		CorrectOptionIDs []string `json:"correct_option_ids"`
		TextAnswer       string   `json:"text_answer"`
		PointsAwarded    *int32   `json:"points_awarded"`
		NeedsGrading     bool     `json:"needs_grading"`
	} `json:"questions"`
	Pending int64 `json:"pending"`
}

// essayAttempt seeds a quiz with an essay, sits it, and returns the attempt id.
func essayAttempt(t *testing.T, h http.Handler, owner, student actor) (attemptID, quizID string) {
	t.Helper()
	_, quizID = seedQuiz(t, h, owner, student, 2)
	if rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/questions", owner.token, owner.slug, map[string]any{
		"kind": "essay", "prompt": "اشرح الفرق", "points": 4,
	}); rec.Code != http.StatusCreated {
		t.Fatalf("add essay: got %d: %s", rec.Code, rec.Body)
	}

	attempt := startAttempt(t, h, student, quizID)
	if rec := do(t, h, "PUT", "/v1/attempts/"+attempt.Attempt.ID+"/answers", student.token, owner.slug,
		map[string]any{"question_id": attempt.Questions[0].ID, "option_ids": labelled(t, attempt, 0, "خمسة")}); rec.Code != http.StatusOK {
		t.Fatalf("answer mcq: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/v1/attempts/"+attempt.Attempt.ID+"/answers", student.token, owner.slug,
		map[string]any{"question_id": attempt.Questions[2].ID, "text": "إجابة مفصلة"}); rec.Code != http.StatusOK {
		t.Fatalf("answer essay: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/v1/attempts/"+attempt.Attempt.ID+"/submit", student.token, owner.slug, nil); rec.Code != http.StatusOK {
		t.Fatalf("submit: got %d: %s", rec.Code, rec.Body)
	}
	return attempt.Attempt.ID, quizID
}

func readSheet(t *testing.T, h http.Handler, a actor, attemptID string) sheetResponse {
	t.Helper()
	rec := do(t, h, "GET", "/v1/attempts/"+attemptID+"/sheet", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("sheet: got %d: %s", rec.Code, rec.Body)
	}
	var sheet sheetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &sheet); err != nil {
		t.Fatalf("decode sheet: %v", err)
	}
	return sheet
}

func TestMarking(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	attemptID, _ := essayAttempt(t, h, owner, student)

	t.Run("the queue lists what needs a teacher", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/marking", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Attempts []struct {
				FullName  string `json:"full_name"`
				QuizTitle string `json:"quiz_title"`
				Pending   int64  `json:"pending"`
			} `json:"attempts"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Attempts) != 1 || got.Attempts[0].Pending != 1 {
			t.Fatalf("got %+v, want one attempt with one pending answer", got.Attempts)
		}
		if got.Attempts[0].QuizTitle != "اختبار قصير" {
			t.Errorf("quiz title = %q", got.Attempts[0].QuizTitle)
		}
	})

	t.Run("a learner cannot see the queue or the answer key", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/marking", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("queue: got %d, want 403", rec.Code)
		}
		if rec := do(t, h, "GET", "/v1/attempts/"+attemptID+"/sheet", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("sheet: got %d, want 403", rec.Code)
		}
	})

	sheet := readSheet(t, h, owner, attemptID)
	var essayID string
	for _, q := range sheet.Questions {
		if q.Kind == "essay" {
			essayID = q.ID
		}
	}

	t.Run("the marker sees the answer key and what is outstanding", func(t *testing.T) {
		if sheet.Pending != 1 || sheet.Attempt.State != "submitted" {
			t.Fatalf("got pending=%d state=%q", sheet.Pending, sheet.Attempt.State)
		}
		if essayID == "" {
			t.Fatal("the essay is missing from the sheet")
		}
		for _, q := range sheet.Questions {
			if q.Kind == "mcq_single" && len(q.CorrectOptionIDs) != 1 {
				t.Errorf("the marker should see the correct option: %+v", q)
			}
			if q.Kind == "essay" && q.TextAnswer != "إجابة مفصلة" {
				t.Errorf("essay answer = %q", q.TextAnswer)
			}
		}
	})

	t.Run("releasing before marking is finished is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/attempts/"+attemptID+"/release", owner.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a mark cannot exceed what the question is worth", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/attempts/"+attemptID+"/questions/"+essayID+"/mark", owner.token, owner.slug,
			map[string]any{"points_awarded": 99})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
		rec = do(t, h, "PUT", "/v1/attempts/"+attemptID+"/questions/"+essayID+"/mark", owner.token, owner.slug,
			map[string]any{"points_awarded": -1})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("negative: got %d, want 422", rec.Code)
		}
		rec = do(t, h, "PUT", "/v1/attempts/"+attemptID+"/questions/"+essayID+"/mark", owner.token, owner.slug,
			map[string]any{"feedback": "no score"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("missing points: got %d, want 422", rec.Code)
		}
	})

	t.Run("marking then releasing gives the learner a result", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/attempts/"+attemptID+"/questions/"+essayID+"/mark", owner.token, owner.slug,
			map[string]any{"points_awarded": 3, "feedback": "جيد، لكن وضّح المثال"})
		if rec.Code != http.StatusOK {
			t.Fatalf("mark: got %d: %s", rec.Code, rec.Body)
		}

		if after := readSheet(t, h, owner, attemptID); after.Pending != 0 {
			t.Fatalf("pending = %d, want 0", after.Pending)
		}

		rec = do(t, h, "POST", "/v1/attempts/"+attemptID+"/release", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("release: got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Attempt struct {
				State         string `json:"state"`
				PointsAwarded int32  `json:"points_awarded"`
			} `json:"attempt"`
			Percent int  `json:"percent"`
			Passed  bool `json:"passed"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// 2 auto-graded plus 3 marked, out of 10.
		if got.Attempt.State != "graded" || got.Attempt.PointsAwarded != 5 {
			t.Fatalf("got %+v", got.Attempt)
		}
		if got.Percent != 50 || !got.Passed {
			t.Fatalf("got %d%% passed=%v, want 50%% and a pass", got.Percent, got.Passed)
		}
	})

	t.Run("a graded attempt cannot be marked or released again", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/attempts/"+attemptID+"/questions/"+essayID+"/mark", owner.token, owner.slug,
			map[string]any{"points_awarded": 4})
		if rec.Code != http.StatusConflict {
			t.Errorf("mark: got %d, want 409", rec.Code)
		}
		if rec := do(t, h, "POST", "/v1/attempts/"+attemptID+"/release", owner.token, owner.slug, nil); rec.Code != http.StatusConflict {
			t.Errorf("release: got %d, want 409", rec.Code)
		}
	})

	t.Run("the queue empties once released", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/marking", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Attempts []json.RawMessage `json:"attempts"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Attempts) != 0 {
			t.Errorf("got %d attempts still queued, want 0", len(got.Attempts))
		}
	})

	t.Run("an attempt in another tenant is not found", func(t *testing.T) {
		other := enrol(t, h, ch, store, "owner")
		if rec := do(t, h, "GET", "/v1/attempts/"+attemptID+"/sheet", other.token, other.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rec.Code)
		}
	})
}
