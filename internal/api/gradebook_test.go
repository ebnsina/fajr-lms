package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type gradebookResponse struct {
	Items []struct {
		ID       string `json:"id"`
		Source   string `json:"source"`
		Title    string `json:"title"`
		Possible int32  `json:"points_possible"`
		Weight   int32  `json:"weight"`
	} `json:"items"`
	Learners []struct {
		EnrollmentID string `json:"enrollment_id"`
		FullName     string `json:"full_name"`
		Percent      int    `json:"percent"`
		Graded       int    `json:"items_graded"`
		Scores       []struct {
			Points     *int32 `json:"points"`
			Percent    *int   `json:"percent"`
			Overridden bool   `json:"overridden"`
			Note       string `json:"note"`
		} `json:"scores"`
	} `json:"learners"`
}

func readGradebook(t *testing.T, h http.Handler, a actor, courseID string) gradebookResponse {
	t.Helper()
	rec := do(t, h, "GET", "/v1/courses/"+courseID+"/gradebook", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("gradebook: got %d: %s", rec.Code, rec.Body)
	}
	var out gradebookResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode gradebook: %v", err)
	}
	return out
}

// gradedCourse builds a course with a sat quiz and returns ids.
func gradedCourse(t *testing.T, h http.Handler, owner, student actor) (courseID, quizID string) {
	t.Helper()
	courseID = createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
		map[string]any{"title": "Usul al-Fiqh", "visibility": "public"}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/modules", owner.token, owner.slug,
		map[string]any{"title": "Unit"}))
	lessonID := createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", owner.token, owner.slug,
		map[string]any{"title": "Assessment", "kind": "quiz"}))
	if rec := do(t, h, "PATCH", "/v1/lessons/"+lessonID, owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish lesson: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/v1/courses/"+courseID+"/status", owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish course: got %d: %s", rec.Code, rec.Body)
	}

	quizID = createdID(t, do(t, h, "POST", "/v1/lessons/"+lessonID+"/quiz", owner.token, owner.slug,
		map[string]any{"title": "Quiz 1", "max_attempts": 2, "pass_percent": 50}))
	if rec := do(t, h, "POST", "/v1/quizzes/"+quizID+"/questions", owner.token, owner.slug, map[string]any{
		"kind": "mcq_single", "prompt": "Pick A", "points": 10,
		"options": []map[string]any{{"label": "A", "is_correct": true}, {"label": "B"}},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("add question: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
		t.Fatalf("enrol: got %d: %s", rec.Code, rec.Body)
	}
	return courseID, quizID
}

func TestGradebook(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	courseID, quizID := gradedCourse(t, h, owner, student)

	t.Run("a quiz appears as a column worth what its questions are worth", func(t *testing.T) {
		book := readGradebook(t, h, owner, courseID)
		if len(book.Items) != 1 || book.Items[0].Source != "quiz" {
			t.Fatalf("got %+v", book.Items)
		}
		if book.Items[0].Possible != 10 {
			t.Errorf("points_possible = %d, want 10 from the question", book.Items[0].Possible)
		}
		if len(book.Learners) != 1 || book.Learners[0].Graded != 0 {
			t.Errorf("nothing is sat yet: %+v", book.Learners)
		}
	})

	t.Run("sitting the quiz fills the column", func(t *testing.T) {
		attempt := startAttempt(t, h, student, quizID)
		if rec := do(t, h, "PUT", "/v1/attempts/"+attempt.Attempt.ID+"/answers", student.token, owner.slug,
			map[string]any{"question_id": attempt.Questions[0].ID, "option_ids": labelled(t, attempt, 0, "A")}); rec.Code != http.StatusOK {
			t.Fatalf("answer: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "POST", "/v1/attempts/"+attempt.Attempt.ID+"/submit", student.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("submit: got %d: %s", rec.Code, rec.Body)
		}

		book := readGradebook(t, h, owner, courseID)
		row := book.Learners[0]
		if row.Graded != 1 || row.Percent != 100 || *row.Scores[0].Points != 10 {
			t.Fatalf("got %+v", row)
		}
	})

	t.Run("the best attempt counts, not the latest", func(t *testing.T) {
		attempt := startAttempt(t, h, student, quizID)
		if rec := do(t, h, "PUT", "/v1/attempts/"+attempt.Attempt.ID+"/answers", student.token, owner.slug,
			map[string]any{"question_id": attempt.Questions[0].ID, "option_ids": labelled(t, attempt, 0, "B")}); rec.Code != http.StatusOK {
			t.Fatalf("answer: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "POST", "/v1/attempts/"+attempt.Attempt.ID+"/submit", student.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("submit: got %d: %s", rec.Code, rec.Body)
		}

		book := readGradebook(t, h, owner, courseID)
		if *book.Learners[0].Scores[0].Points != 10 {
			t.Errorf("a worse resit lowered the grade: %+v", book.Learners[0].Scores[0])
		}
	})

	var itemID, enrollmentID string
	t.Run("a manual item is weighted alongside the quiz", func(t *testing.T) {
		itemID = createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/grade-items", owner.token, owner.slug,
			map[string]any{"title": "Oral exam", "points_possible": 20, "weight": 300, "category": "exam"}))

		book := readGradebook(t, h, owner, courseID)
		if len(book.Items) != 2 {
			t.Fatalf("got %d items, want 2", len(book.Items))
		}
		enrollmentID = book.Learners[0].EnrollmentID
		// Only the quiz is graded, so the average is still 100.
		if book.Learners[0].Percent != 100 || book.Learners[0].Graded != 1 {
			t.Errorf("an unsat item should not drag the average down: %+v", book.Learners[0])
		}
	})

	t.Run("rejects a mark above what the item is worth", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/grade-items/"+itemID+"/learners/"+enrollmentID, owner.token, owner.slug,
			map[string]any{"points": 999})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a teacher's score changes the weighted average", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/grade-items/"+itemID+"/learners/"+enrollmentID, owner.token, owner.slug,
			map[string]any{"points": 10, "note": "hesitant on the second question"})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		book := readGradebook(t, h, owner, courseID)
		row := book.Learners[0]
		// 100% at weight 100 and 50% at weight 300 is 62.5, rounded to 63.
		if row.Percent != 63 || row.Graded != 2 {
			t.Fatalf("got %+v", row)
		}
		if !row.Scores[1].Overridden || row.Scores[1].Note == "" {
			t.Errorf("the manual score should be marked as a teacher's: %+v", row.Scores[1])
		}
	})

	t.Run("a teacher can override a quiz score and undo it", func(t *testing.T) {
		book := readGradebook(t, h, owner, courseID)
		quizItem := book.Items[0].ID

		if rec := do(t, h, "PUT", "/v1/grade-items/"+quizItem+"/learners/"+enrollmentID, owner.token, owner.slug,
			map[string]any{"points": 4, "note": "resat under supervision"}); rec.Code != http.StatusOK {
			t.Fatalf("override: got %d: %s", rec.Code, rec.Body)
		}
		if got := readGradebook(t, h, owner, courseID).Learners[0]; *got.Scores[0].Points != 4 || !got.Scores[0].Overridden {
			t.Fatalf("the override did not win: %+v", got.Scores[0])
		}

		if rec := do(t, h, "DELETE", "/v1/grade-items/"+quizItem+"/learners/"+enrollmentID, owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("clear: got %d: %s", rec.Code, rec.Body)
		}
		got := readGradebook(t, h, owner, courseID).Learners[0]
		if *got.Scores[0].Points != 10 || got.Scores[0].Overridden {
			t.Errorf("clearing should restore the quiz result: %+v", got.Scores[0])
		}
		if rec := do(t, h, "DELETE", "/v1/grade-items/"+quizItem+"/learners/"+enrollmentID, owner.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("clearing twice: got %d, want 404", rec.Code)
		}
	})

	t.Run("a learner sees their own grades but not the class", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/courses/"+courseID+"/gradebook", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("gradebook: got %d, want 403", rec.Code)
		}
		rec := do(t, h, "GET", "/v1/courses/"+courseID+"/grades", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("grades: got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Grades struct {
				Percent  int    `json:"percent"`
				FullName string `json:"full_name"`
			} `json:"grades"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Grades.Percent != 63 {
			t.Errorf("percent = %d, want 63", got.Grades.Percent)
		}
	})

	t.Run("a learner not enrolled has no grades", func(t *testing.T) {
		outsider := enrolIn(t, h, ch, store, owner.slug, "student")
		if rec := do(t, h, "GET", "/v1/courses/"+courseID+"/grades", outsider.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rec.Code)
		}
	})

	t.Run("a gradebook in another tenant is not found", func(t *testing.T) {
		other := enrol(t, h, ch, store, "owner")
		if rec := do(t, h, "GET", "/v1/courses/"+courseID+"/gradebook", other.token, other.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rec.Code)
		}
	})
}
