package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

type rollResponse struct {
	Session struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"session"`
	Roll []struct {
		EnrollmentID string  `json:"enrollment_id"`
		UserID       string  `json:"user_id"`
		FullName     string  `json:"full_name"`
		Status       *string `json:"status"`
	} `json:"roll"`
}

func readRoll(t *testing.T, h http.Handler, a actor, sessionID string) rollResponse {
	t.Helper()
	rec := do(t, h, "GET", "/v1/sessions/"+sessionID+"/roll", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("roll: got %d: %s", rec.Code, rec.Body)
	}
	var out rollResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode roll: %v", err)
	}
	return out
}

func TestAttendance(t *testing.T) {
	h, _, ch, store := notifyHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	guardian := enrollIn(t, h, ch, store, owner.slug, "parent")

	courseID, _ := publishedCourse(t, h, owner, 1)
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
		t.Fatalf("enroll: got %d: %s", rec.Code, rec.Body)
	}

	start := time.Now().Add(-time.Hour)
	sessionID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/sessions", owner.token, owner.slug,
		map[string]any{"title": "صباح الاثنين", "location": "Room 2", "starts_at": start}))

	t.Run("rejects a class that ends before it starts", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/sessions", owner.token, owner.slug,
			map[string]any{"title": "Bad", "starts_at": start, "ends_at": start.Add(-time.Minute)})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a student cannot create a class or take the roll", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/sessions", student.token, owner.slug,
			map[string]any{"title": "Mine", "starts_at": start}); rec.Code != http.StatusForbidden {
			t.Errorf("create: got %d, want 403", rec.Code)
		}
		if rec := do(t, h, "GET", "/v1/sessions/"+sessionID+"/roll", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("read roll: got %d, want 403", rec.Code)
		}
	})

	roll := readRoll(t, h, owner, sessionID)
	if len(roll.Roll) != 1 || roll.Roll[0].Status != nil {
		t.Fatalf("an unmarked roll should list the learner with no status: %+v", roll.Roll)
	}
	enrollmentID := roll.Roll[0].EnrollmentID

	t.Run("rejects an unknown status", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/sessions/"+sessionID+"/roll", owner.token, owner.slug, map[string]any{
			"entries": []map[string]any{{"enrollment_id": enrollmentID, "status": "maybe"}},
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects an empty roll", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/sessions/"+sessionID+"/roll", owner.token, owner.slug,
			map[string]any{"entries": []map[string]any{}})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("the whole class is marked in one request", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/sessions/"+sessionID+"/roll", owner.token, owner.slug, map[string]any{
			"entries": []map[string]any{{"enrollment_id": enrollmentID, "status": "present"}},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Marked int `json:"marked"`
			Absent int `json:"absent"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Marked != 1 || got.Absent != 0 {
			t.Fatalf("got %+v", got)
		}
		if after := readRoll(t, h, owner, sessionID); *after.Roll[0].Status != "present" {
			t.Errorf("status = %v, want present", *after.Roll[0].Status)
		}
	})

	t.Run("a correction replaces the earlier mark", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/sessions/"+sessionID+"/roll", owner.token, owner.slug, map[string]any{
			"entries": []map[string]any{{"enrollment_id": enrollmentID, "status": "late", "note": "bus"}},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if after := readRoll(t, h, owner, sessionID); *after.Roll[0].Status != "late" {
			t.Errorf("status = %v, want late", *after.Roll[0].Status)
		}
	})

	t.Run("a guardian and the learner both hear about an absence", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/guardians", owner.token, owner.slug, map[string]any{
			"guardian_id": guardian.userID.String(), "student_id": student.userID.String(), "relation": "father",
		}); rec.Code != http.StatusCreated {
			t.Fatalf("link guardian: got %d: %s", rec.Code, rec.Body)
		}

		second := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/sessions", owner.token, owner.slug,
			map[string]any{"title": "Tuesday", "starts_at": start.Add(24 * time.Hour)}))
		rec := do(t, h, "PUT", "/v1/sessions/"+second+"/roll", owner.token, owner.slug, map[string]any{
			"entries": []map[string]any{{"enrollment_id": enrollmentID, "status": "absent"}},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("mark absent: got %d: %s", rec.Code, rec.Body)
		}

		for _, who := range []struct {
			name string
			a    actor
		}{{"learner", student}, {"guardian", guardian}} {
			rec := do(t, h, "GET", "/v1/notifications", who.a.token, owner.slug, nil)
			var got struct {
				Notifications []struct {
					Kind string `json:"kind"`
				} `json:"notifications"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(got.Notifications) == 0 || got.Notifications[0].Kind != "attendance.absent" {
				t.Errorf("%s was not told: %+v", who.name, got.Notifications)
			}
		}
	})

	t.Run("the learner sees their own record and rate", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/courses/"+courseID+"/attendance", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Sessions []json.RawMessage `json:"sessions"`
			Summary  struct {
				Present  int64 `json:"present"`
				Late     int64 `json:"late"`
				Absent   int64 `json:"absent"`
				Rate     int   `json:"rate_percent"`
				Recorded int64 `json:"sessions_recorded"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// One late and one absent: a late arrival still counts as attended.
		if got.Summary.Late != 1 || got.Summary.Absent != 1 || got.Summary.Recorded != 2 {
			t.Fatalf("got %+v", got.Summary)
		}
		if got.Summary.Rate != 50 {
			t.Errorf("rate = %d, want 50", got.Summary.Rate)
		}
		if len(got.Sessions) != 2 {
			t.Errorf("got %d sessions, want 2", len(got.Sessions))
		}
	})

	t.Run("excused leave does not damage the rate", func(t *testing.T) {
		third := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/sessions", owner.token, owner.slug,
			map[string]any{"title": "Wednesday", "starts_at": start.Add(48 * time.Hour)}))
		if rec := do(t, h, "PUT", "/v1/sessions/"+third+"/roll", owner.token, owner.slug, map[string]any{
			"entries": []map[string]any{{"enrollment_id": enrollmentID, "status": "excused", "note": "medical"}},
		}); rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}

		rec := do(t, h, "GET", "/v1/courses/"+courseID+"/attendance", student.token, owner.slug, nil)
		var got struct {
			Summary struct {
				Excused int64 `json:"excused"`
				Rate    int   `json:"rate_percent"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Summary.Excused != 1 || got.Summary.Rate != 50 {
			t.Errorf("got %+v, want the rate unchanged at 50", got.Summary)
		}
	})

	t.Run("only staff manage guardians", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/guardians", student.token, owner.slug, map[string]any{
			"guardian_id": student.userID.String(), "student_id": guardian.userID.String(),
		})
		if rec.Code != http.StatusForbidden {
			t.Errorf("got %d, want 403", rec.Code)
		}
	})

	t.Run("a learner cannot be their own guardian", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/guardians", owner.token, owner.slug, map[string]any{
			"guardian_id": student.userID.String(), "student_id": student.userID.String(),
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a session in another tenant is not found", func(t *testing.T) {
		other := enroll(t, h, ch, store, "owner")
		if rec := do(t, h, "GET", "/v1/sessions/"+sessionID+"/roll", other.token, other.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rec.Code)
		}
	})
}
