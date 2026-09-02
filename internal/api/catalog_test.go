package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/api"
)

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Introduction to Tajweed", "introduction-to-tajweed"},
		{"  Spaced   Out  ", "spaced-out"},
		{"Grade 9 — Physics!", "grade-9-physics"},
	}
	for _, c := range cases {
		if got := api.Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// Arabic and Bengali titles have no ASCII, so they get a generated slug.
	for _, title := range []string{"مقدمة في التجويد", "নবম শ্রেণির পদার্থবিজ্ঞান", "!!!"} {
		got := api.Slugify(title)
		if len(got) < 3 || got[:2] != "c-" {
			t.Errorf("Slugify(%q) = %q, want a generated c- slug", title, got)
		}
	}
}

func TestCatalog(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	var course struct {
		ID    string `json:"id"`
		Slug  string `json:"slug"`
		Title string `json:"title"`
		Dir   string `json:"dir"`
	}

	t.Run("an instructor creates a course with an Arabic title", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses", owner.token, owner.slug, map[string]any{
			"title": "مقدمة في التجويد", "summary": "دورة تمهيدية", "dir": "rtl", "price_minor": 150000,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &course); err != nil {
			t.Fatalf("decode course: %v", err)
		}
		if course.Title != "مقدمة في التجويد" || course.Dir != "rtl" {
			t.Errorf("got %+v", course)
		}
		if course.Slug[:2] != "c-" {
			t.Errorf("slug %q should be generated for a non-ASCII title", course.Slug)
		}
	})

	t.Run("a student cannot create a course", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses", student.token, owner.slug, map[string]any{"title": "Sneaky"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects a blank title", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses", owner.token, owner.slug, map[string]any{"title": "   "})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects a duplicate slug", func(t *testing.T) {
		body := map[string]any{"title": "Duplicate", "slug": "taken-" + owner.slug}
		if rec := do(t, h, "POST", "/v1/courses", owner.token, owner.slug, body); rec.Code != http.StatusCreated {
			t.Fatalf("first create: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "POST", "/v1/courses", owner.token, owner.slug, body); rec.Code != http.StatusConflict {
			t.Fatalf("second create: got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects an unknown lesson kind", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+course.ID+"/modules", owner.token, owner.slug,
			map[string]any{"title": "Unit 1"})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create module: got %d: %s", rec.Code, rec.Body)
		}
		var module struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &module); err != nil {
			t.Fatalf("decode module: %v", err)
		}
		rec = do(t, h, "POST", "/v1/modules/"+module.ID+"/lessons", owner.token, owner.slug,
			map[string]any{"title": "Bad", "kind": "hologram"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("builds and reads back an outline in order", func(t *testing.T) {
		var moduleIDs []string
		for _, title := range []string{"الوحدة الأولى", "الوحدة الثانية"} {
			rec := do(t, h, "POST", "/v1/courses/"+course.ID+"/modules", owner.token, owner.slug,
				map[string]any{"title": title})
			if rec.Code != http.StatusCreated {
				t.Fatalf("create module: got %d: %s", rec.Code, rec.Body)
			}
			var m struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
				t.Fatalf("decode module: %v", err)
			}
			moduleIDs = append(moduleIDs, m.ID)
		}

		for _, title := range []string{"Lesson A", "Lesson B", "Lesson C"} {
			rec := do(t, h, "POST", "/v1/modules/"+moduleIDs[0]+"/lessons", owner.token, owner.slug,
				map[string]any{"title": title, "kind": "video", "duration_s": 600})
			if rec.Code != http.StatusCreated {
				t.Fatalf("create lesson %s: got %d: %s", title, rec.Code, rec.Body)
			}
		}

		got := readOutline(t, h, owner, course.Slug)
		if len(got.Modules) != 3 {
			t.Fatalf("got %d modules, want 3", len(got.Modules))
		}
		if len(got.Modules[1].Lessons) != 3 {
			t.Fatalf("got %d lessons in module 2, want 3", len(got.Modules[1].Lessons))
		}
		if got.Modules[1].Lessons[0].Title != "Lesson A" {
			t.Errorf("lessons out of order: %+v", got.Modules[1].Lessons)
		}

		// Move C between A and B; only that row changes.
		lessons := got.Modules[1].Lessons
		mid := (lessons[0].Position + lessons[1].Position) / 2
		rec := do(t, h, "PUT", "/v1/lessons/"+lessons[2].ID+"/position", owner.token, owner.slug,
			map[string]any{"module_id": moduleIDs[0], "position": mid})
		if rec.Code != http.StatusOK {
			t.Fatalf("move: got %d, want 200: %s", rec.Code, rec.Body)
		}

		got = readOutline(t, h, owner, course.Slug)
		titles := []string{}
		for _, l := range got.Modules[1].Lessons {
			titles = append(titles, l.Title)
		}
		want := []string{"Lesson A", "Lesson C", "Lesson B"}
		for i := range want {
			if titles[i] != want[i] {
				t.Fatalf("order after move = %v, want %v", titles, want)
			}
		}
	})

	t.Run("a student reads the outline but cannot delete a lesson", func(t *testing.T) {
		got := readOutline(t, h, student, course.Slug)
		if len(got.Modules) == 0 {
			t.Fatal("student sees no modules")
		}
		id := got.Modules[1].Lessons[0].ID
		if rec := do(t, h, "DELETE", "/v1/lessons/"+id, student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403", rec.Code)
		}
		if rec := do(t, h, "DELETE", "/v1/lessons/"+id, owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("owner delete: got %d, want 204", rec.Code)
		}
		if rec := do(t, h, "DELETE", "/v1/lessons/"+id, owner.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Fatalf("second delete: got %d, want 404", rec.Code)
		}
	})

	t.Run("publishing stamps a date", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/courses/"+course.ID+"/status", owner.token, owner.slug,
			map[string]any{"status": "published"})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body)
		}
		var got struct {
			Status      string  `json:"status"`
			PublishedAt *string `json:"published_at"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Status != "published" || got.PublishedAt == nil {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("a course in another tenant is not found", func(t *testing.T) {
		other := enroll(t, h, ch, store, "owner")
		rec := do(t, h, "GET", "/v1/courses/"+course.Slug, other.token, other.slug, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})
}
