package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type attemptStart struct {
	Attempt struct {
		ID        string `json:"id"`
		AttemptNo int    `json:"attempt_no"`
		State     string `json:"state"`
	} `json:"attempt"`
	Questions []struct {
		ID      string `json:"id"`
		Options []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"options"`
	} `json:"questions"`
}

type attemptResult struct {
	Attempt struct {
		State          string `json:"state"`
		PointsAwarded  int32  `json:"points_awarded"`
		PointsPossible int32  `json:"points_possible"`
	} `json:"attempt"`
	Percent   int  `json:"percent"`
	Passed    bool `json:"passed"`
	Pending   bool `json:"awaiting_marking"`
	Breakdown []struct {
		Points      int32  `json:"points_awarded"`
		Explanation string `json:"explanation"`
	} `json:"breakdown"`
}

func startAttempt(t *testing.T, h http.Handler, a actor, quizID string) attemptStart {
	t.Helper()
	rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/attempts", a.token, a.slug, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start attempt: got %d: %s", rec.Code, rec.Body)
	}
	var out attemptStart
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode attempt: %v", err)
	}
	return out
}

func labelled(t *testing.T, q attemptStart, question int, labels ...string) []string {
	t.Helper()
	var ids []string
	for _, label := range labels {
		found := false
		for _, opt := range q.Questions[question].Options {
			if opt.Label == label {
				ids = append(ids, opt.ID)
				found = true
			}
		}
		if !found {
			t.Fatalf("option %q not offered", label)
		}
	}
	return ids
}

func TestAttemptLifecycle(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	_, quizID := seedQuiz(t, h, owner, student, 2)

	t.Run("a learner who is not enrolled cannot attempt", func(t *testing.T) {
		outsider := enrolIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/attempts", outsider.token, owner.slug, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	first := startAttempt(t, h, student, quizID)
	if first.Attempt.AttemptNo != 1 || first.Attempt.State != "in_progress" {
		t.Fatalf("got %+v", first.Attempt)
	}

	t.Run("starting again resumes the same attempt", func(t *testing.T) {
		again := startAttempt(t, h, student, quizID)
		if again.Attempt.ID != first.Attempt.ID {
			t.Errorf("got a new attempt %s, want to resume %s", again.Attempt.ID, first.Attempt.ID)
		}
	})

	t.Run("answering another learner's attempt is not found", func(t *testing.T) {
		other := enrolIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "PUT", "/v1/attempts/"+first.Attempt.ID+"/answers", other.token, owner.slug,
			map[string]any{"question_id": first.Questions[0].ID, "option_ids": []string{}})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a question from another quiz is refused", func(t *testing.T) {
		otherOwner := enrol(t, h, ch, store, "owner")
		otherStudent := enrolIn(t, h, ch, store, otherOwner.slug, "student")
		_, otherQuiz := seedQuiz(t, h, otherOwner, otherStudent, 1)
		foreign := startAttempt(t, h, otherStudent, otherQuiz)

		rec := do(t, h, "PUT", "/v1/attempts/"+first.Attempt.ID+"/answers", student.token, owner.slug,
			map[string]any{"question_id": foreign.Questions[0].ID})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("answers are saved without being graded", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/attempts/"+first.Attempt.ID+"/answers", student.token, owner.slug,
			map[string]any{"question_id": first.Questions[0].ID, "option_ids": labelled(t, first, 0, "خمسة")})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var saved struct {
			PointsAwarded *int32 `json:"points_awarded"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if saved.PointsAwarded != nil {
			t.Errorf("an answer was graded on save: %v", *saved.PointsAwarded)
		}
	})

	t.Run("changing an answer replaces it", func(t *testing.T) {
		for _, label := range []string{"أربعة", "خمسة"} {
			rec := do(t, h, "PUT", "/v1/attempts/"+first.Attempt.ID+"/answers", student.token, owner.slug,
				map[string]any{"question_id": first.Questions[0].ID, "option_ids": labelled(t, first, 0, label)})
			if rec.Code != http.StatusOK {
				t.Fatalf("got %d: %s", rec.Code, rec.Body)
			}
		}
	})

	// Answer the multi correctly; the single is already right.
	if rec := do(t, h, "PUT", "/v1/attempts/"+first.Attempt.ID+"/answers", student.token, owner.slug,
		map[string]any{"question_id": first.Questions[1].ID, "option_ids": labelled(t, first, 1, "A", "B")}); rec.Code != http.StatusOK {
		t.Fatalf("answer multi: got %d: %s", rec.Code, rec.Body)
	}

	var result attemptResult
	t.Run("submitting grades everything at once", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/attempts/"+first.Attempt.ID+"/submit", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result.Attempt.State != "graded" || result.Attempt.PointsAwarded != 6 || result.Attempt.PointsPossible != 6 {
			t.Fatalf("got %+v", result.Attempt)
		}
		if result.Percent != 100 || !result.Passed || result.Pending {
			t.Fatalf("got %+v", result)
		}
		if len(result.Breakdown) != 2 || result.Breakdown[0].Explanation != "خمسة" {
			t.Errorf("explanations should be revealed after submission: %+v", result.Breakdown)
		}
	})

	t.Run("a submitted attempt is closed", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/attempts/"+first.Attempt.ID+"/submit", student.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Errorf("resubmit: got %d, want 409", rec.Code)
		}
		rec = do(t, h, "PUT", "/v1/attempts/"+first.Attempt.ID+"/answers", student.token, owner.slug,
			map[string]any{"question_id": first.Questions[0].ID, "option_ids": labelled(t, first, 0, "خمسة")})
		if rec.Code != http.StatusConflict {
			t.Errorf("answer after submit: got %d, want 409", rec.Code)
		}
	})

	t.Run("a second attempt scores independently", func(t *testing.T) {
		second := startAttempt(t, h, student, quizID)
		if second.Attempt.AttemptNo != 2 {
			t.Fatalf("attempt no = %d, want 2", second.Attempt.AttemptNo)
		}
		// Answer only the cheap question, and get it wrong.
		if rec := do(t, h, "PUT", "/v1/attempts/"+second.Attempt.ID+"/answers", student.token, owner.slug,
			map[string]any{"question_id": second.Questions[0].ID, "option_ids": labelled(t, second, 0, "ستة")}); rec.Code != http.StatusOK {
			t.Fatalf("answer: got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "POST", "/v1/attempts/"+second.Attempt.ID+"/submit", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("submit: got %d: %s", rec.Code, rec.Body)
		}
		var got attemptResult
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Attempt.PointsAwarded != 0 || got.Percent != 0 || got.Passed {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("the attempt limit is enforced", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/attempts", student.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})
}

func TestWrittenAnswersWaitForMarking(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	_, quizID := seedQuiz(t, h, owner, student, 3)

	if rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/questions", owner.token, owner.slug, map[string]any{
		"kind": "essay", "prompt": "اشرح بإيجاز", "points": 4,
	}); rec.Code != http.StatusCreated {
		t.Fatalf("add essay: got %d: %s", rec.Code, rec.Body)
	}

	attempt := startAttempt(t, h, student, quizID)
	if len(attempt.Questions) != 3 {
		t.Fatalf("got %d questions, want 3", len(attempt.Questions))
	}
	if rec := do(t, h, "PUT", "/v1/attempts/"+attempt.Attempt.ID+"/answers", student.token, owner.slug,
		map[string]any{"question_id": attempt.Questions[0].ID, "option_ids": labelled(t, attempt, 0, "خمسة")}); rec.Code != http.StatusOK {
		t.Fatalf("answer mcq: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/v1/attempts/"+attempt.Attempt.ID+"/answers", student.token, owner.slug,
		map[string]any{"question_id": attempt.Questions[2].ID, "text": "الجواب المفصل هنا"}); rec.Code != http.StatusOK {
		t.Fatalf("answer essay: got %d: %s", rec.Code, rec.Body)
	}

	rec := do(t, h, "POST", "/v1/attempts/"+attempt.Attempt.ID+"/submit", student.token, owner.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit: got %d: %s", rec.Code, rec.Body)
	}
	var got attemptResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Attempt.State != "submitted" || !got.Pending {
		t.Fatalf("an essay should hold the attempt for marking: %+v", got)
	}
	if got.Passed {
		t.Error("an attempt awaiting marking must not be reported as passed")
	}
	if got.Attempt.PointsAwarded != 2 {
		t.Errorf("auto-graded points = %d, want 2", got.Attempt.PointsAwarded)
	}
}
