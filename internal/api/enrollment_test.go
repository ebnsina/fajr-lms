package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// publishedCourse creates a free, published course with n published lessons.
func publishedCourse(t *testing.T, h http.Handler, a actor, lessons int) (courseID string, lessonIDs []string) {
	t.Helper()
	courseID = createdID(t, do(t, h, "POST", "/v1/courses", a.token, a.slug,
		map[string]any{"title": "Tajweed Basics", "visibility": "public"}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/modules", a.token, a.slug,
		map[string]any{"title": "Unit 1"}))

	for range lessons {
		id := createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", a.token, a.slug,
			map[string]any{"title": "Lesson", "kind": "video", "duration_s": 600}))
		if rec := do(t, h, "PATCH", "/v1/lessons/"+id, a.token, a.slug,
			map[string]any{"status": "published"}); rec.Code != http.StatusOK {
			t.Fatalf("publish lesson: got %d: %s", rec.Code, rec.Body)
		}
		lessonIDs = append(lessonIDs, id)
	}
	if rec := do(t, h, "PUT", "/v1/courses/"+courseID+"/status", a.token, a.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish course: got %d: %s", rec.Code, rec.Body)
	}
	return courseID, lessonIDs
}

func TestEnrollmentAndProgress(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	other := enrollIn(t, h, ch, store, owner.slug, "student")

	courseID, lessons := publishedCourse(t, h, owner, 2)

	t.Run("a learner enrolls themselves in a free public course", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
		var got struct {
			Source string `json:"source"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Source != "self" || got.Status != "active" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("enrolling twice is idempotent", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a learner cannot enroll somebody else", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug,
			map[string]any{"user_id": other.userID.String()})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("staff enroll on someone's behalf", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", owner.token, owner.slug,
			map[string]any{"user_id": other.userID.String()})
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
	})

	t.Run("self-enrollment into a private course is refused", func(t *testing.T) {
		privateID := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
			map[string]any{"title": "Staff Only"}))
		rec := do(t, h, "POST", "/v1/courses/"+privateID+"/enrollments", student.token, owner.slug, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("progress without an enrollment is not found", func(t *testing.T) {
		outsider := enrollIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "PUT", "/v1/lessons/"+lessons[0]+"/progress", outsider.token, owner.slug,
			map[string]any{"position_s": 10})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a resume point moves forward but never back", func(t *testing.T) {
		report := func(pos int, done bool) progressBody {
			rec := do(t, h, "PUT", "/v1/lessons/"+lessons[0]+"/progress", student.token, owner.slug,
				map[string]any{"position_s": pos, "completed": done})
			if rec.Code != http.StatusOK {
				t.Fatalf("progress %d: got %d: %s", pos, rec.Code, rec.Body)
			}
			var got progressBody
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			return got
		}

		if got := report(120, false); got.Progress.PositionS != 120 {
			t.Fatalf("position = %d, want 120", got.Progress.PositionS)
		}
		if got := report(300, false); got.Progress.PositionS != 300 {
			t.Fatalf("position = %d, want 300", got.Progress.PositionS)
		}
		// A stale report arriving from an offline device must not rewind.
		if got := report(45, false); got.Progress.PositionS != 300 {
			t.Errorf("stale sync rewound position to %d, want 300", got.Progress.PositionS)
		}

		got := report(300, true)
		if got.Progress.State != "completed" || got.PercentComplete != 50 {
			t.Fatalf("got %+v, want completed at 50%%", got)
		}
		// A later in-progress report must not un-complete the lesson.
		if got := report(310, false); got.Progress.State != "completed" {
			t.Errorf("stale sync un-completed the lesson: %+v", got.Progress)
		}
	})

	t.Run("finishing every lesson completes the enrollment", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/lessons/"+lessons[1]+"/progress", student.token, owner.slug,
			map[string]any{"position_s": 600, "completed": true})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got progressBody
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !got.CourseComplete || got.PercentComplete != 100 {
			t.Fatalf("got %+v, want complete at 100%%", got)
		}

		rec = do(t, h, "GET", "/v1/courses/"+courseID+"/progress", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("progress: got %d: %s", rec.Code, rec.Body)
		}
		var course struct {
			Enrollment struct {
				Status string `json:"status"`
			} `json:"enrollment"`
			PercentComplete int `json:"percent_complete"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &course); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if course.Enrollment.Status != "completed" || course.PercentComplete != 100 {
			t.Errorf("got %+v", course)
		}
	})

	t.Run("the roster shows completion per learner", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/courses/"+courseID+"/roster", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Roster []struct {
				PercentComplete int `json:"percent_complete"`
			} `json:"roster"`
			PublishedLessons int64 `json:"published_lessons"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.PublishedLessons != 2 || len(got.Roster) != 2 {
			t.Fatalf("got %d lessons and %d learners, want 2 and 2", got.PublishedLessons, len(got.Roster))
		}
		if got.Roster[0].PercentComplete+got.Roster[1].PercentComplete != 100 {
			t.Errorf("one learner should be at 100%% and one at 0%%: %+v", got.Roster)
		}
		if rec := do(t, h, "GET", "/v1/courses/"+courseID+"/roster", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("student roster: got %d, want 403", rec.Code)
		}
	})

	t.Run("my enrolments list what I joined", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/enrollments", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Enrollments []struct {
				Title string `json:"title"`
			} `json:"enrollments"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Enrollments) != 1 || got.Enrollments[0].Title != "Tajweed Basics" {
			t.Errorf("got %+v", got.Enrollments)
		}
	})

	t.Run("cancelling stops further progress", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/courses/"+courseID+"/progress", other.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("progress: got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Enrollment struct {
				ID string `json:"id"`
			} `json:"enrollment"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if rec := do(t, h, "DELETE", "/v1/enrollments/"+got.Enrollment.ID, owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("cancel: got %d, want 204: %s", rec.Code, rec.Body)
		}
		rec = do(t, h, "PUT", "/v1/lessons/"+lessons[0]+"/progress", other.token, owner.slug,
			map[string]any{"position_s": 30})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("progress after cancel: got %d, want 403: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "DELETE", "/v1/enrollments/"+got.Enrollment.ID, owner.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("second cancel: got %d, want 404", rec.Code)
		}
	})
}

type progressBody struct {
	Progress struct {
		PositionS int32  `json:"position_s"`
		State     string `json:"state"`
	} `json:"progress"`
	PercentComplete int  `json:"percent_complete"`
	CourseComplete  bool `json:"course_complete"`
}
