package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/media"
)

func testRegistry(t *testing.T) *media.Registry {
	t.Helper()
	r, err := media.NewRegistry("embed", media.Embed{AllowedHosts: []string{"tube.madrasa.test"}})
	if err != nil {
		t.Fatalf("build media registry: %v", err)
	}
	return r
}

func TestMedia(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")

	t.Run("lists provider capabilities", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/media/providers", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Providers []media.Caps `json:"providers"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Providers) != 1 || got.Providers[0].Name != "embed" || !got.Providers[0].AcceptsURL {
			t.Errorf("got %+v", got.Providers)
		}
	})

	var asset struct {
		ID       string `json:"id"`
		Provider string `json:"provider"`
		State    string `json:"state"`
		Kind     string `json:"kind"`
	}

	t.Run("ingests a pasted video link", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/media", owner.token, owner.slug, map[string]any{
			"url": "https://youtu.be/dQw4w9WgXcQ", "title": "الدرس الأول",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &asset); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if asset.Provider != "embed" || asset.State != "ready" || asset.Kind != "video" {
			t.Errorf("got %+v", asset)
		}
	})

	t.Run("rejects a link no provider handles", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/media", owner.token, owner.slug,
			map[string]any{"url": "https://evil.example.com/clip.mp4"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects an unconfigured provider by name", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/media", owner.token, owner.slug,
			map[string]any{"url": "https://youtu.be/dQw4w9WgXcQ", "provider": "transcoder"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a student cannot ingest media", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/media", student.token, owner.slug,
			map[string]any{"url": "https://youtu.be/dQw4w9WgXcQ"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403", rec.Code)
		}
	})

	t.Run("a student may play it back", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/media/"+asset.ID+"/playback", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got media.Playback
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Kind != media.PlaybackEmbed || got.URL != "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("attaches to a lesson and detaches again", func(t *testing.T) {
		lessonID := seedLesson(t, h, owner)

		rec := do(t, h, "PUT", "/v1/lessons/"+lessonID+"/media", owner.token, owner.slug,
			map[string]any{"media_id": asset.ID})
		if rec.Code != http.StatusOK {
			t.Fatalf("attach: got %d: %s", rec.Code, rec.Body)
		}
		var lesson struct {
			MediaID *string `json:"media_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &lesson); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if lesson.MediaID == nil || *lesson.MediaID != asset.ID {
			t.Fatalf("got %+v, want %s", lesson.MediaID, asset.ID)
		}

		rec = do(t, h, "PUT", "/v1/lessons/"+lessonID+"/media", owner.token, owner.slug,
			map[string]any{"media_id": ""})
		if rec.Code != http.StatusOK {
			t.Fatalf("detach: got %d: %s", rec.Code, rec.Body)
		}
	})

	t.Run("media from another tenant is not found", func(t *testing.T) {
		other := enrol(t, h, ch, store, "owner")
		rec := do(t, h, "GET", "/v1/media/"+asset.ID+"/playback", other.token, other.slug, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("owners see delivery counted per day", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/media/usage", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Days []struct {
				Requests int64 `json:"requests"`
			} `json:"days"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Days) != 1 || got.Days[0].Requests < 1 {
			t.Errorf("got %+v, want at least one delivery today", got.Days)
		}
		if rec := do(t, h, "GET", "/v1/media/usage", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Errorf("student usage: got %d, want 403", rec.Code)
		}
	})
}

// seedLesson creates a course, module and lesson, returning the lesson id.
func seedLesson(t *testing.T, h http.Handler, a actor) string {
	t.Helper()
	courseID := createdID(t, do(t, h, "POST", "/v1/courses", a.token, a.slug, map[string]any{"title": "Media Course"}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/modules", a.token, a.slug, map[string]any{"title": "Unit"}))
	return createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", a.token, a.slug,
		map[string]any{"title": "Lesson", "kind": "video"}))
}
