package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestDemo covers the public demo: what it asks, what it seeds, and that
// nobody can change what they were shown.
func TestDemo(t *testing.T) {
	h, _, _ := newHarness(t)

	ask := func(t *testing.T, body map[string]any) (int, map[string]any) {
		t.Helper()
		rec := do(t, h, "POST", "/v1/demo", "", "", body)
		var out map[string]any
		if rec.Body.Len() > 0 {
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
				t.Fatalf("decode: %v: %s", err, rec.Body)
			}
		}
		return rec.Code, out
	}

	t.Run("the form asks what they run, and refuses a kind we have no school for", func(t *testing.T) {
		code, _ := ask(t, map[string]any{
			"full_name": "Rashida Khan", "email": "rashida@example.school", "runs": "spaceship",
		})
		if code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422", code)
		}
	})

	t.Run("an address we cannot reach is refused", func(t *testing.T) {
		code, _ := ask(t, map[string]any{
			"full_name": "Rashida Khan", "email": "not-an-address", "runs": "madrasah",
		})
		if code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422", code)
		}
	})

	code, out := ask(t, map[string]any{
		"full_name": "Rashida Khan", "email": "rashida@example.school", "phone": "+8801711000000",
		"organisation": "Darul Uloom", "role": "Principal", "learners": "200-500",
		"runs": "madrasah", "note": "Looking to move off paper.",
	})
	if code != http.StatusOK {
		t.Fatalf("demo: got %d: %v", code, out)
	}
	token, _ := out["token"].(string)
	slug, _ := out["tenant"].(string)
	if token == "" || slug != "demo-madrasah" {
		t.Fatalf("got %v", out)
	}

	t.Run("the demo school is already full of work", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/courses", token, slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("courses: got %d: %s", rec.Code, rec.Body)
		}
		var courses struct {
			Courses []struct {
				Title  string `json:"title"`
				Status string `json:"status"`
			} `json:"courses"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &courses); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(courses.Courses) < 3 {
			t.Fatalf("only %d courses seeded: %s", len(courses.Courses), rec.Body)
		}

		roster := do(t, h, "GET", "/v1/tenant/members", token, slug, nil)
		if roster.Code != http.StatusOK {
			t.Fatalf("members: got %d: %s", roster.Code, roster.Body)
		}
		var members struct {
			Members []map[string]any `json:"members"`
		}
		if err := json.Unmarshal(roster.Body.Bytes(), &members); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(members.Members) < 7 {
			t.Fatalf("only %d members seeded", len(members.Members))
		}
	})

	t.Run("nothing in a demo school can be changed", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses", token, slug,
			map[string]any{"title": "Mine now", "summary": "No."})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
		var out struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Error.Code != "demo_read_only" {
			t.Fatalf("refused as %q", out.Error.Code)
		}
	})

	t.Run("the second visitor joins the school the first one was shown", func(t *testing.T) {
		code, out := ask(t, map[string]any{
			"full_name": "Imran Chowdhury", "email": "imran@example.school", "runs": "madrasah",
		})
		if code != http.StatusOK {
			t.Fatalf("got %d: %v", code, out)
		}
		if out["tenant"] != slug {
			t.Fatalf("sent to %v, want %s", out["tenant"], slug)
		}
	})

	t.Run("each kind gets its own school", func(t *testing.T) {
		code, out := ask(t, map[string]any{
			"full_name": "Amina Begum", "email": "amina@example.com", "runs": "creator",
		})
		if code != http.StatusOK {
			t.Fatalf("got %d: %v", code, out)
		}
		if out["tenant"] != "demo-creator" {
			t.Fatalf("got %v", out["tenant"])
		}
	})
}
