package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestLiveClassLinks(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")

	courseID, _ := publishedCourse(t, h, owner, 1)
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
		t.Fatalf("enrol: got %d: %s", rec.Code, rec.Body)
	}
	now := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/sessions", owner.token, owner.slug,
		map[string]any{"title": "Live tafsir", "starts_at": time.Now()}))

	t.Run("a class with no link yet says so", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/sessions/"+now+"/join", student.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("only accepted platforms may be linked", func(t *testing.T) {
		for _, bad := range []string{
			"", "not a url", "http://meet.google.com/abc-defg-hij",
			"https://evil.example.com/room", "javascript:alert(1)",
		} {
			rec := do(t, h, "PUT", "/v1/sessions/"+now+"/link", owner.token, owner.slug,
				map[string]any{"join_url": bad})
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("%q: got %d, want 422", bad, rec.Code)
			}
		}
	})

	t.Run("a student cannot set the link", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/sessions/"+now+"/link", student.token, owner.slug,
			map[string]any{"join_url": "https://meet.google.com/abc-defg-hij"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403", rec.Code)
		}
	})

	t.Run("Meet and Zoom links are recognised", func(t *testing.T) {
		cases := map[string]string{
			"https://meet.google.com/abc-defg-hij": "google_meet",
			"https://us05web.zoom.us/j/123456789":  "zoom",
		}
		for link, provider := range cases {
			rec := do(t, h, "PUT", "/v1/sessions/"+now+"/link", owner.token, owner.slug,
				map[string]any{"join_url": link})
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: got %d: %s", link, rec.Code, rec.Body)
			}
			var got struct {
				Provider string `json:"provider"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.Provider != provider {
				t.Errorf("%s recognised as %q, want %q", link, got.Provider, provider)
			}
		}
	})

	t.Run("an enrolled learner joins around the scheduled time", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/sessions/"+now+"/join", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got joinBody
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.JoinURL == "" || got.Provider != "zoom" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("a learner who is not enrolled gets nothing", func(t *testing.T) {
		outsider := enrolIn(t, h, ch, store, owner.slug, "student")
		if rec := do(t, h, "GET", "/v1/sessions/"+now+"/join", outsider.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rec.Code)
		}
	})

	t.Run("a link is not handed out days in advance", func(t *testing.T) {
		future := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/sessions", owner.token, owner.slug,
			map[string]any{"title": "Next week", "starts_at": time.Now().Add(72 * time.Hour)}))
		if rec := do(t, h, "PUT", "/v1/sessions/"+future+"/link", owner.token, owner.slug,
			map[string]any{"join_url": "https://meet.google.com/xyz-abcd-efg"}); rec.Code != http.StatusOK {
			t.Fatalf("set link: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "GET", "/v1/sessions/"+future+"/join", student.token, owner.slug, nil); rec.Code != http.StatusConflict {
			t.Errorf("learner: got %d, want 409", rec.Code)
		}
		// A teacher may open the room whenever they like.
		if rec := do(t, h, "GET", "/v1/sessions/"+future+"/join", owner.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Errorf("teacher: got %d, want 200", rec.Code)
		}
	})

	t.Run("the teacher gets the host link when there is one", func(t *testing.T) {
		if rec := do(t, h, "PUT", "/v1/sessions/"+now+"/link", owner.token, owner.slug, map[string]any{
			"join_url": "https://meet.google.com/abc-defg-hij",
			"host_url": "https://meet.google.com/abc-defg-hij?authuser=1",
		}); rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		teacher := joinAs(t, h, owner, now)
		learner := joinAs(t, h, student, now)
		if teacher.JoinURL == learner.JoinURL {
			t.Errorf("teacher and learner got the same link: %q", teacher.JoinURL)
		}
	})

	t.Run("a recording is attached back to the class", func(t *testing.T) {
		mediaID := createdID(t, do(t, h, "POST", "/v1/media", owner.token, owner.slug,
			map[string]any{"url": "https://youtu.be/dQw4w9WgXcQ", "title": "Recording"}))
		if rec := do(t, h, "PUT", "/v1/sessions/"+now+"/recording", owner.token, owner.slug,
			map[string]any{"media_id": mediaID}); rec.Code != http.StatusOK {
			t.Fatalf("attach: got %d: %s", rec.Code, rec.Body)
		}
		if got := joinAs(t, h, student, now); got.Recording == nil || *got.Recording != mediaID {
			t.Errorf("recording = %v, want %s", got.Recording, mediaID)
		}
	})
}

type joinBody struct {
	Provider  string  `json:"provider"`
	JoinURL   string  `json:"join_url"`
	Recording *string `json:"recording_media_id"`
}

func joinAs(t *testing.T, h http.Handler, a actor, sessionID string) joinBody {
	t.Helper()
	rec := do(t, h, "GET", "/v1/sessions/"+sessionID+"/join", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("join: got %d: %s", rec.Code, rec.Body)
	}
	var out joinBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode join: %v", err)
	}
	return out
}
