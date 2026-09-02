package api_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"
)

type submissionBody struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	IsLate  bool   `json:"is_late"`
	Points  *int32 `json:"points_awarded"`
	Penalty int32  `json:"late_penalty_applied"`
}

// assignmentCourse publishes a course with one assignment lesson.
func assignmentCourse(t *testing.T, h http.Handler, owner, student actor, body map[string]any) (courseID, assignmentID, lessonID string) {
	t.Helper()
	courseID = createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
		map[string]any{"title": fmt.Sprintf("Hadith Studies %09d", rand.IntN(1_000_000_000)), "visibility": "public"}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/modules", owner.token, owner.slug,
		map[string]any{"title": "Unit"}))
	lessonID = createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", owner.token, owner.slug,
		map[string]any{"title": "Essay task", "kind": "assignment"}))
	if rec := do(t, h, "PATCH", "/v1/lessons/"+lessonID, owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish lesson: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/v1/courses/"+courseID+"/status", owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish course: got %d: %s", rec.Code, rec.Body)
	}

	assignmentID = createdID(t, do(t, h, "POST", "/v1/lessons/"+lessonID+"/assignment", owner.token, owner.slug, body))
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
		t.Fatalf("enrol: got %d: %s", rec.Code, rec.Body)
	}
	return courseID, assignmentID, lessonID
}

func hand(t *testing.T, h http.Handler, a actor, assignmentID string, body map[string]any) *httpRec {
	t.Helper()
	rec := do(t, h, "PUT", "/v1/assignments/"+assignmentID+"/submission", a.token, a.slug, body)
	return &httpRec{code: rec.Code, body: rec.Body.Bytes()}
}

type httpRec struct {
	code int
	body []byte
}

func (r *httpRec) decode(t *testing.T) submissionBody {
	t.Helper()
	var out submissionBody
	if err := json.Unmarshal(r.body, &out); err != nil {
		t.Fatalf("decode submission: %v", err)
	}
	return out
}

func TestAssignmentFlow(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	due := time.Now().Add(time.Hour)
	courseID, assignmentID, lessonID := assignmentCourse(t, h, owner, student, map[string]any{
		"title": "واجب الأسبوع", "dir": "rtl", "points": 50, "due_at": due, "late_penalty": 20,
	})

	t.Run("a lesson has at most one assignment", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/lessons/"+lessonID+"/assignment", owner.token, owner.slug,
			map[string]any{"title": "Second"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a student cannot set an assignment", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/lessons/"+lessonID+"/assignment", student.token, owner.slug,
			map[string]any{"title": "Mine"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403", rec.Code)
		}
	})

	t.Run("handing in nothing is refused", func(t *testing.T) {
		if got := hand(t, h, student, assignmentID, map[string]any{"submit": true}); got.code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", got.code, got.body)
		}
	})

	t.Run("a draft saves without handing in", func(t *testing.T) {
		got := hand(t, h, student, assignmentID, map[string]any{"body": "مسودة"})
		if got.code != http.StatusOK {
			t.Fatalf("got %d: %s", got.code, got.body)
		}
		if s := got.decode(t); s.State != "draft" {
			t.Errorf("state = %q, want draft", s.State)
		}
	})

	t.Run("too many attachments are refused", func(t *testing.T) {
		media := make([]string, 6)
		for i := range media {
			media[i] = "00000000-0000-0000-0000-00000000000" + string(rune('1'+i))
		}
		got := hand(t, h, student, assignmentID, map[string]any{"body": "x", "media_ids": media})
		if got.code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", got.code, got.body)
		}
	})

	var submissionID string
	t.Run("handing in on time is not late", func(t *testing.T) {
		got := hand(t, h, student, assignmentID, map[string]any{"body": "الإجابة النهائية", "submit": true})
		if got.code != http.StatusOK {
			t.Fatalf("got %d: %s", got.code, got.body)
		}
		s := got.decode(t)
		if s.State != "submitted" || s.IsLate {
			t.Fatalf("got %+v", s)
		}
		submissionID = s.ID
	})

	t.Run("the queue shows work waiting to be marked", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/submissions", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("student: got %d, want 403", rec.Code)
		}
		rec := do(t, h, "GET", "/v1/submissions", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Submissions []struct {
				FullName        string `json:"full_name"`
				AssignmentTitle string `json:"assignment_title"`
			} `json:"submissions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Submissions) != 1 || got.Submissions[0].AssignmentTitle != "واجب الأسبوع" {
			t.Errorf("got %+v", got.Submissions)
		}
	})

	t.Run("a mark above the assignment's worth is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/submissions/"+submissionID+"/grade", owner.token, owner.slug,
			map[string]any{"points_awarded": 500})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("marking returns the work and fills the gradebook", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/submissions/"+submissionID+"/grade", owner.token, owner.slug,
			map[string]any{"points_awarded": 40, "feedback": "أحسنت"})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got submissionBody
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.State != "returned" || *got.Points != 40 || got.Penalty != 0 {
			t.Fatalf("got %+v", got)
		}

		book := readGradebook(t, h, owner, courseID)
		if len(book.Items) != 1 || book.Items[0].Source != "assignment" {
			t.Fatalf("got %+v", book.Items)
		}
		if book.Items[0].Possible != 50 {
			t.Errorf("points_possible = %d, want 50", book.Items[0].Possible)
		}
		if book.Learners[0].Percent != 80 {
			t.Errorf("percent = %d, want 80", book.Learners[0].Percent)
		}
	})

	t.Run("marked work cannot be resubmitted or marked twice", func(t *testing.T) {
		if got := hand(t, h, student, assignmentID, map[string]any{"body": "again", "submit": true}); got.code != http.StatusConflict {
			t.Errorf("resubmit: got %d, want 409: %s", got.code, got.body)
		}
		rec := do(t, h, "POST", "/v1/submissions/"+submissionID+"/grade", owner.token, owner.slug,
			map[string]any{"points_awarded": 50})
		if rec.Code != http.StatusConflict {
			t.Errorf("regrade: got %d, want 409", rec.Code)
		}
	})
}

func TestLateSubmissions(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	past := time.Now().Add(-time.Hour)

	t.Run("late work is accepted and penalised once", func(t *testing.T) {
		_, assignmentID, _ := assignmentCourse(t, h, owner, student, map[string]any{
			"title": "Late allowed", "points": 100, "due_at": past, "late_penalty": 25,
		})
		got := hand(t, h, student, assignmentID, map[string]any{"body": "sorry", "submit": true})
		if got.code != http.StatusOK {
			t.Fatalf("got %d: %s", got.code, got.body)
		}
		s := got.decode(t)
		if !s.IsLate {
			t.Fatal("work handed in after the deadline should be marked late")
		}

		rec := do(t, h, "POST", "/v1/submissions/"+s.ID+"/grade", owner.token, owner.slug,
			map[string]any{"points_awarded": 80})
		if rec.Code != http.StatusOK {
			t.Fatalf("grade: got %d: %s", rec.Code, rec.Body)
		}
		var graded submissionBody
		if err := json.Unmarshal(rec.Body.Bytes(), &graded); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if *graded.Points != 60 || graded.Penalty != 20 {
			t.Errorf("got %d points with a %d penalty, want 60 and 20", *graded.Points, graded.Penalty)
		}
	})

	t.Run("a closed deadline refuses late work", func(t *testing.T) {
		other := enrolIn(t, h, ch, store, owner.slug, "student")
		_, assignmentID, _ := assignmentCourse(t, h, owner, other, map[string]any{
			"title": "No late", "due_at": past, "allow_late": false,
		})
		got := hand(t, h, other, assignmentID, map[string]any{"body": "too late", "submit": true})
		if got.code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", got.code, got.body)
		}
		// A draft can still be saved, so nothing the learner typed is lost.
		if draft := hand(t, h, other, assignmentID, map[string]any{"body": "kept"}); draft.code != http.StatusOK {
			t.Errorf("draft after deadline: got %d, want 200: %s", draft.code, draft.body)
		}
	})
}

func TestSubmissionSheet(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	_, assignmentID, _ := assignmentCourse(t, h, owner, student, map[string]any{
		"title": "Reflection", "points": 30,
	})

	mediaID := createdID(t, do(t, h, "POST", "/v1/media", owner.token, owner.slug,
		map[string]any{"url": "https://youtu.be/dQw4w9WgXcQ", "kind": "link"}))
	got := hand(t, h, student, assignmentID, map[string]any{
		"body": "my reflection", "media_ids": []string{mediaID}, "submit": true,
	})
	if got.code != http.StatusOK {
		t.Fatalf("hand in: got %d: %s", got.code, got.body)
	}
	submissionID := got.decode(t).ID

	t.Run("a marker sees the work, the brief and the attachments", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/submissions/"+submissionID, owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var sheet struct {
			Submission struct {
				Body string `json:"body"`
			} `json:"submission"`
			Assignment struct {
				Points int32 `json:"points"`
			} `json:"assignment"`
			FullName    string `json:"full_name"`
			Attachments []struct {
				MediaID  string `json:"media_id"`
				Playback *struct {
					URL string `json:"url"`
				} `json:"playback"`
			} `json:"attachments"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &sheet); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if sheet.Submission.Body != "my reflection" || sheet.Assignment.Points != 30 {
			t.Fatalf("got %+v", sheet)
		}
		if sheet.FullName == "" {
			t.Error("the marker needs to know whose work this is")
		}
		if len(sheet.Attachments) != 1 || sheet.Attachments[0].Playback == nil {
			t.Fatalf("attachment did not come back playable: %+v", sheet.Attachments)
		}
	})

	t.Run("a learner cannot open the sheet", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/submissions/"+submissionID, student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("got %d, want 403", rec.Code)
		}
	})

	t.Run("a submission in another tenant is not found", func(t *testing.T) {
		other := enrol(t, h, ch, store, "owner")
		if rec := do(t, h, "GET", "/v1/submissions/"+submissionID, other.token, other.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rec.Code)
		}
	})
}
