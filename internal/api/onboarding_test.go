package api_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
)

func TestCreateSchool(t *testing.T) {
	h, ch, _ := newHarness(t)
	token, _ := login(t, h, ch)

	// The slug is globally unique, so the name is too, run to run.
	name := fmt.Sprintf("Riyadh Institute %d", rand.IntN(1_000_000_000))
	rec := do(t, h, "POST", "/v1/tenants", token, "", map[string]any{
		"name": name, "kind": "institution", "dir": "rtl", "locale": "ar", "currency": "SAR",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create school: got %d: %s", rec.Code, rec.Body)
	}
	var tenant struct {
		Slug     string `json:"slug"`
		Name     string `json:"name"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tenant); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tenant.Slug == "" || tenant.Currency != "SAR" {
		t.Fatalf("unexpected school: %+v", tenant)
	}

	t.Run("the founder owns it straight away", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/tenant", token, tenant.Slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var current struct {
			Role string `json:"role"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &current); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if current.Role != "owner" {
			t.Errorf("role is %q, want owner", current.Role)
		}
	})

	t.Run("the address cannot be taken twice", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/tenants", token, "", map[string]any{
			"name": "Another school", "slug": tenant.Slug,
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a stranger cannot open a school", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/tenants", "", "", map[string]any{"name": "Nobody's school"})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401: %s", rec.Code, rec.Body)
		}
	})
}
