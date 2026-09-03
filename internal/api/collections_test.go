package api_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
)

// TestTopicsAndCollections covers filing courses under topics, and putting
// several together as a path to work through or a bundle to buy.
func TestTopicsAndCollections(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	first, _ := publishedCourse(t, h, owner, 1)
	second := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
		map[string]any{"title": fmt.Sprintf("Second course %d", rand.IntN(100000)), "visibility": "public"}))

	unique := rand.IntN(100000)
	topicID := createdID(t, do(t, h, "POST", "/v1/topics", owner.token, owner.slug,
		map[string]any{"name": fmt.Sprintf("Arabic %d", unique)}))

	t.Run("a learner cannot invent topics", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/topics", student.token, owner.slug,
			map[string]any{"name": "Anything"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("filing a course replaces what it was filed under", func(t *testing.T) {
		other := createdID(t, do(t, h, "POST", "/v1/topics", owner.token, owner.slug,
			map[string]any{"name": fmt.Sprintf("Fiqh %d", unique)}))

		if rec := do(t, h, "PUT", "/v1/courses/"+first+"/topics", owner.token, owner.slug,
			map[string]any{"topic_ids": []string{topicID, other}}); rec.Code != http.StatusOK {
			t.Fatalf("tag: got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "PUT", "/v1/courses/"+first+"/topics", owner.token, owner.slug,
			map[string]any{"topic_ids": []string{other}})
		if rec.Code != http.StatusOK {
			t.Fatalf("retag: got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Topics []struct {
				ID string `json:"id"`
			} `json:"topics"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Topics) != 1 || out.Topics[0].ID != other {
			t.Fatalf("got %s, want only the second topic", rec.Body)
		}
	})

	t.Run("a path is worked through, so it carries no price", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/collections", owner.token, owner.slug, map[string]any{
			"kind": "path", "title": "Grammar from the start", "price_minor": 5000,
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	pathSlug := fmt.Sprintf("grammar-path-%d", unique)
	pathID := createdID(t, do(t, h, "POST", "/v1/collections", owner.token, owner.slug,
		map[string]any{"kind": "path", "slug": pathSlug, "title": "Grammar from the start"}))

	t.Run("courses keep the order they were added in", func(t *testing.T) {
		for _, course := range []string{first, second} {
			if rec := do(t, h, "POST", "/v1/collections/"+pathID+"/courses", owner.token, owner.slug,
				map[string]any{"course_id": course}); rec.Code != http.StatusNoContent {
				t.Fatalf("add: got %d: %s", rec.Code, rec.Body)
			}
		}
		rec := do(t, h, "GET", "/v1/collections/"+pathSlug, student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("read: got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Courses []struct {
				Course struct {
					ID string `json:"id"`
				} `json:"course"`
			} `json:"courses"`
			CoursesDone int `json:"courses_done"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Courses) != 2 || out.Courses[0].Course.ID != first {
			t.Fatalf("got %s", rec.Body)
		}
		if out.CoursesDone != 0 {
			t.Fatalf("a learner who has done nothing is %d courses in", out.CoursesDone)
		}
	})

	t.Run("a course can be taken back out", func(t *testing.T) {
		if rec := do(t, h, "DELETE", "/v1/collections/"+pathID+"/courses/"+second,
			owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "DELETE", "/v1/collections/"+pathID+"/courses/"+second,
			owner.token, owner.slug, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("removing it twice: got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a bundle carries a price and lists as a bundle", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/collections", owner.token, owner.slug, map[string]any{
			"kind": "bundle", "slug": fmt.Sprintf("everything-%d", unique),
			"title": "Everything", "price_minor": 250000,
		}); rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "GET", "/v1/collections?kind=bundle", owner.token, owner.slug, nil)
		var out struct {
			Collections []struct {
				Kind       string `json:"kind"`
				PriceMinor int64  `json:"price_minor"`
			} `json:"collections"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Collections) == 0 {
			t.Fatal("the bundle is not in the list")
		}
		for _, row := range out.Collections {
			if row.Kind != "bundle" {
				t.Fatalf("asking for bundles returned a %s", row.Kind)
			}
		}
	})

	t.Run("two collections cannot share an address", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/collections", owner.token, owner.slug, map[string]any{
			"kind": "path", "slug": pathSlug, "title": "Another one",
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})
}
