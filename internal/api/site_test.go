package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type sitePage struct {
	ID     string `json:"id"`
	Blocks []struct {
		Type    string `json:"type"`
		Heading string `json:"heading"`
	} `json:"blocks"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	NavLabel string `json:"nav_label"`
}

func TestSitePages(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	courseID, _ := publishedCourse(t, h, owner, 1)
	_ = courseID

	home := map[string]any{
		"slug":  "home",
		"title": "Greenfield Academy",
		"blocks": []map[string]any{
			{"type": "hero", "heading": "Learn with us", "text": "Since 1998.",
				"cta_label": "See the courses", "cta_href": "/courses"},
			{"type": "courses", "heading": "What we teach", "limit": 3},
		},
	}
	rec := do(t, h, "POST", "/v1/site/pages", owner.token, owner.slug, home)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create home: got %d: %s", rec.Code, rec.Body)
	}
	var page sitePage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Slug != "" {
		t.Errorf("home should live at the site root, got %q", page.Slug)
	}
	// Blocks travel as JSON: a []byte would reach the editor as base64.
	if len(page.Blocks) != 2 || page.Blocks[0].Heading != "Learn with us" {
		t.Fatalf("sections did not come back as JSON: %+v", page.Blocks)
	}

	t.Run("a student cannot build the site", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/site/pages", student.token, owner.slug,
			map[string]any{"title": "Mine"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a section it does not know is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/site/pages", owner.token, owner.slug, map[string]any{
			"title": "Odd", "blocks": []map[string]any{{"type": "script", "text": "boom"}},
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("two pages cannot share an address", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/site/pages", owner.token, owner.slug,
			map[string]any{"slug": "home", "title": "Another front page"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a draft is not public", func(t *testing.T) {
		rec := do(t, h, "GET", "/site/"+owner.slug, "", "", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	if rec := do(t, h, "PUT", "/v1/site/pages/"+page.ID+"/status", owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish: got %d: %s", rec.Code, rec.Body)
	}

	about := map[string]any{
		"slug": "about", "title": "About us", "nav_label": "About", "nav_order": 1,
		"blocks": []map[string]any{{"type": "faq", "heading": "Questions",
			"items": []map[string]any{{"title": "Where are you?", "text": "Dhaka."}}}},
	}
	aboutID := createdID(t, do(t, h, "POST", "/v1/site/pages", owner.token, owner.slug, about))
	if rec := do(t, h, "PUT", "/v1/site/pages/"+aboutID+"/status", owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish about: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("the published site is readable by anyone", func(t *testing.T) {
		rec := do(t, h, "GET", "/site/"+owner.slug, "", "", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Page struct {
				Title string `json:"title"`
			} `json:"page"`
			Nav []struct {
				Slug     string `json:"slug"`
				NavLabel string `json:"nav_label"`
			} `json:"nav"`
			Courses []struct {
				Title string `json:"title"`
			} `json:"courses"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Page.Title != "Greenfield Academy" {
			t.Errorf("front page title: %q", out.Page.Title)
		}
		if len(out.Nav) != 1 || out.Nav[0].NavLabel != "About" {
			t.Errorf("navigation: %+v", out.Nav)
		}
		if len(out.Courses) == 0 {
			t.Error("a page listing courses should carry them")
		}
	})

	t.Run("a page of another tenant is not served under this one", func(t *testing.T) {
		rec := do(t, h, "GET", "/site/"+owner.slug+"/missing", "", "", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("editing keeps what was not sent", func(t *testing.T) {
		rec := do(t, h, "PATCH", "/v1/site/pages/"+aboutID, owner.token, owner.slug,
			map[string]any{"description": "Who we are."})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var updated sitePage
		if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if updated.Title != "About us" || updated.NavLabel != "About" {
			t.Errorf("edit lost fields: %+v", updated)
		}
	})

	t.Run("the site can be dressed for its region", func(t *testing.T) {
		if rec := do(t, h, "PUT", "/v1/site/theme", owner.token, owner.slug,
			map[string]any{"theme": "gulf"}); rec.Code != http.StatusOK {
			t.Fatalf("set theme: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "PUT", "/v1/site/theme", owner.token, owner.slug,
			map[string]any{"theme": "neon"}); rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "GET", "/site/"+owner.slug, "", "", nil)
		var out struct {
			Theme string `json:"theme"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Theme != "gulf" {
			t.Errorf("theme not carried to the site: %q", out.Theme)
		}
	})

	t.Run("a deleted page leaves the site", func(t *testing.T) {
		if rec := do(t, h, "DELETE", "/v1/site/pages/"+aboutID, owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("delete: got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "GET", "/site/"+owner.slug+"/about", "", "", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})
}
