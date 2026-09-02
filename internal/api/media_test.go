package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/media"
)

// testRegistry adds the file provider only when an object store is configured,
// so the suite still runs without one.
func testRegistry(t *testing.T) *media.Registry {
	t.Helper()
	providers := []media.Provider{media.Embed{AllowedHosts: []string{"tube.madrasa.test"}}}

	if endpoint := os.Getenv("FAJR_S3_ENDPOINT"); endpoint != "" {
		store, err := media.NewObjectStore(context.Background(), media.ObjectStoreConfig{
			Endpoint: endpoint, Bucket: "fajr-test",
			AccessKey: os.Getenv("FAJR_S3_ACCESS_KEY"), SecretKey: os.Getenv("FAJR_S3_SECRET_KEY"),
			Region: "us-east-1", MaxBytes: 1 << 20,
		})
		if err != nil {
			t.Fatalf("build object store: %v", err)
		}
		providers = append(providers, store)
	}

	r, err := media.NewRegistry("embed", providers...)
	if err != nil {
		t.Fatalf("build media registry: %v", err)
	}
	return r
}

func TestFileUploadFlow(t *testing.T) {
	if os.Getenv("FAJR_S3_ENDPOINT") == "" {
		t.Skip("FAJR_S3_ENDPOINT not set")
	}
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")

	var created struct {
		ID     string        `json:"id"`
		State  string        `json:"state"`
		Upload *media.Upload `json:"upload"`
	}

	t.Run("asking to upload returns a direct upload target", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/media", owner.token, owner.slug, map[string]any{
			"provider": "file", "filename": "درس.mp4", "content_type": "video/mp4",
			"byte_size": 11, "title": "Lesson one",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if created.State != "pending" || created.Upload == nil || created.Upload.Method != "PUT" {
			t.Fatalf("got %+v", created)
		}
	})

	t.Run("completing before the bytes land is a conflict", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/media/"+created.ID+"/complete", owner.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("playback of a pending asset says not ready", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/media/"+created.ID+"/playback", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got media.Playback
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Kind != media.PlaybackNotReady {
			t.Errorf("got %+v, want not_ready", got)
		}
	})

	t.Run("uploading then completing makes it playable", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, created.Upload.URL, strings.NewReader("hello video"))
		if err != nil {
			t.Fatalf("build upload: %v", err)
		}
		req.Header.Set("Content-Type", "video/mp4")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("upload: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("upload: got %d", resp.StatusCode)
		}

		rec := do(t, h, "POST", "/v1/media/"+created.ID+"/complete", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("complete: got %d: %s", rec.Code, rec.Body)
		}
		var asset struct {
			State    string `json:"state"`
			ByteSize int64  `json:"byte_size"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &asset); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if asset.State != "ready" || asset.ByteSize != 11 {
			t.Fatalf("got %+v, want ready at 11 bytes", asset)
		}

		rec = do(t, h, "GET", "/v1/media/"+created.ID+"/playback", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("playback: got %d: %s", rec.Code, rec.Body)
		}
		var got media.Playback
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Kind != media.PlaybackFile || got.ExpiresAt == nil {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("an oversized or disallowed file is refused up front", func(t *testing.T) {
		big := do(t, h, "POST", "/v1/media", owner.token, owner.slug, map[string]any{
			"provider": "file", "filename": "huge.mp4", "content_type": "video/mp4", "byte_size": 1 << 30,
		})
		if big.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("oversized: got %d, want 413: %s", big.Code, big.Body)
		}
		bad := do(t, h, "POST", "/v1/media", owner.token, owner.slug, map[string]any{
			"provider": "file", "filename": "page.html", "content_type": "text/html", "byte_size": 10,
		})
		if bad.Code != http.StatusUnprocessableEntity {
			t.Errorf("disallowed type: got %d, want 422: %s", bad.Code, bad.Body)
		}
	})

	t.Run("an embed asset has nothing to complete", func(t *testing.T) {
		id := createdID(t, do(t, h, "POST", "/v1/media", owner.token, owner.slug,
			map[string]any{"url": "https://youtu.be/dQw4w9WgXcQ"}))
		rec := do(t, h, "POST", "/v1/media/"+id+"/complete", owner.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})
}

func TestMedia(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

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
		byName := map[string]media.Caps{}
		for _, c := range got.Providers {
			byName[c.Name] = c
		}
		if embed, ok := byName["embed"]; !ok || !embed.AcceptsURL || embed.AcceptsFile {
			t.Errorf("embed caps = %+v", embed)
		}
		if file, ok := byName["file"]; ok && (!file.AcceptsFile || !file.Offline) {
			t.Errorf("file caps = %+v", file)
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
			map[string]any{"url": "https://youtu.be/dQw4w9WgXcQ", "provider": "nonexistent"})
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
		other := enroll(t, h, ch, store, "owner")
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
