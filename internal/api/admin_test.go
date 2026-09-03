package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAdmin covers the back office: who may open it, and what it shows.
func TestAdmin(t *testing.T) {
	h, ch, store := newHarness(t)

	// A lead to look at, from the demo the back office exists to measure.
	if rec := do(t, h, "POST", "/v1/demo", "", "", map[string]any{
		"full_name": "Rashida Khan", "email": "rashida@example.school",
		"organisation": "Darul Uloom", "runs": "madrasah",
	}); rec.Code != http.StatusOK {
		t.Fatalf("demo: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("a wrong password says nothing about who is staff", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/admin/login", "", "",
			map[string]any{"email": "sina@fajrlabs.com", "password": "not it"})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401: %s", rec.Code, rec.Body)
		}
		other := do(t, h, "POST", "/v1/admin/login", "", "",
			map[string]any{"email": "nobody@example.com", "password": "not it"})
		if other.Body.String() != rec.Body.String() {
			t.Fatal("the answer differs for an address that is not staff")
		}
	})

	rec := do(t, h, "POST", "/v1/admin/login", "", "",
		map[string]any{"email": "sina@fajrlabs.com", "password": "fajrlabs@313"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: got %d: %s", rec.Code, rec.Body)
	}
	var signed struct {
		Token string `json:"token"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &signed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if signed.Token == "" || signed.Role != "owner" {
		t.Fatalf("got %s", rec.Body)
	}

	t.Run("somebody who is not staff cannot find the back office", func(t *testing.T) {
		owner := enroll(t, h, ch, store, "owner")
		if got := do(t, h, "GET", "/v1/admin/overview", owner.token, "", nil); got.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", got.Code, got.Body)
		}
		if got := do(t, h, "GET", "/v1/admin/leads", "", "", nil); got.Code != http.StatusUnauthorized {
			t.Fatalf("signed out got %d, want 401", got.Code)
		}
	})

	t.Run("the numbers count what is there", func(t *testing.T) {
		got := do(t, h, "GET", "/v1/admin/overview", signed.Token, "", nil)
		if got.Code != http.StatusOK {
			t.Fatalf("got %d: %s", got.Code, got.Body)
		}
		var numbers map[string]any
		if err := json.Unmarshal(got.Body.Bytes(), &numbers); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, key := range []string{"schools", "demo_schools", "leads", "leads_converted", "people"} {
			if _, ok := numbers[key]; !ok {
				t.Fatalf("%s is missing: %s", key, got.Body)
			}
		}
		if numbers["leads"].(float64) < 1 || numbers["demo_schools"].(float64) < 1 {
			t.Fatalf("nothing counted: %s", got.Body)
		}
	})

	t.Run("every school is listed, demo ones included", func(t *testing.T) {
		got := do(t, h, "GET", "/v1/admin/schools", signed.Token, "", nil)
		if got.Code != http.StatusOK {
			t.Fatalf("got %d: %s", got.Code, got.Body)
		}
		var out struct {
			Schools []struct {
				ID   string `json:"id"`
				Slug string `json:"slug"`
				Demo bool   `json:"demo"`
			} `json:"schools"`
		}
		if err := json.Unmarshal(got.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		var demo string
		for _, school := range out.Schools {
			if school.Slug == "demo-madrasah" && school.Demo {
				demo = school.ID
			}
		}
		if demo == "" {
			t.Fatalf("the demo school is not listed: %s", got.Body)
		}

		one := do(t, h, "GET", "/v1/admin/schools/"+demo, signed.Token, "", nil)
		if one.Code != http.StatusOK {
			t.Fatalf("one school: got %d: %s", one.Code, one.Body)
		}
		if !strings.Contains(one.Body.String(), "\"members\"") {
			t.Fatalf("no members on the school page: %s", one.Body)
		}
	})

	t.Run("a lead can be worked, and taken away as a file", func(t *testing.T) {
		got := do(t, h, "GET", "/v1/admin/leads", signed.Token, "", nil)
		if got.Code != http.StatusOK {
			t.Fatalf("got %d: %s", got.Code, got.Body)
		}
		var out struct {
			Leads []struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				State string `json:"state"`
			} `json:"leads"`
		}
		if err := json.Unmarshal(got.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Leads) == 0 || out.Leads[0].Email != "rashida@example.school" {
			t.Fatalf("got %s", got.Body)
		}
		if out.Leads[0].State != "new" {
			t.Fatalf("a fresh lead is %q", out.Leads[0].State)
		}

		worked := do(t, h, "PUT", "/v1/admin/leads/"+out.Leads[0].ID, signed.Token, "",
			map[string]any{"state": "contacted", "note": "Called Tuesday. Wants a trial term."})
		if worked.Code != http.StatusOK {
			t.Fatalf("work: got %d: %s", worked.Code, worked.Body)
		}

		filtered := do(t, h, "GET", "/v1/admin/leads?state=contacted", signed.Token, "", nil)
		if !strings.Contains(filtered.Body.String(), "Called Tuesday") {
			t.Fatalf("the note did not stick: %s", filtered.Body)
		}
		if bad := do(t, h, "GET", "/v1/admin/leads?state=maybe", signed.Token, "",
			nil); bad.Code != http.StatusUnprocessableEntity {
			t.Fatalf("a made-up state got %d", bad.Code)
		}

		req := httptest.NewRequest("GET", "/v1/admin/leads.csv", nil)
		req.Header.Set("Authorization", "Bearer "+signed.Token)
		file := httptest.NewRecorder()
		h.ServeHTTP(file, req)
		if file.Code != http.StatusOK {
			t.Fatalf("csv: got %d", file.Code)
		}
		if !strings.HasPrefix(file.Body.String(), "when,name,email") {
			t.Fatalf("csv looks wrong: %s", file.Body)
		}
		if !strings.Contains(file.Body.String(), "Darul Uloom") {
			t.Fatal("the lead is not in the file")
		}
	})
}
