package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/api"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/notify"
)

// captureChannel keeps the last code so tests can complete a login.
type captureChannel struct{ last notify.Message }

func (c *captureChannel) Name() string { return "capture" }

func (c *captureChannel) Send(_ context.Context, m notify.Message) error {
	c.last = m
	return nil
}

func newHarness(t *testing.T) (http.Handler, *captureChannel, *database.Store) {
	t.Helper()
	url := os.Getenv("FAJR_DATABASE_URL")
	if url == "" {
		t.Skip("FAJR_DATABASE_URL not set")
	}
	store, err := database.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(store.Close)

	ch := &captureChannel{}
	return api.NewServer(store, identity.New(store, ch), testRegistry(t)).Routes(), ch, store
}

func do(t *testing.T, h http.Handler, method, path, token, tenant string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenant != "" {
		req.Header.Set("X-Fajr-Tenant", tenant)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func codeFrom(t *testing.T, body string) string {
	t.Helper()
	var code string
	if _, err := fmt.Sscanf(body, "Your Fajr code is %6s.", &code); err != nil {
		t.Fatalf("no code in %q: %v", body, err)
	}
	return code
}

func TestLoginFlow(t *testing.T) {
	h, ch, store := newHarness(t)
	ctx := context.Background()
	phone := fmt.Sprintf("+8809%08d", rand.IntN(100_000_000))
	slug := fmt.Sprintf("school-%09d", rand.IntN(1_000_000_000))

	t.Run("rejects a malformed destination", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/auth/otp", "", "", map[string]string{"destination": "not-a-phone"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects an unknown field", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/auth/otp", "", "", map[string]string{"destination": phone, "oops": "x"})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body)
		}
	})

	if rec := do(t, h, "POST", "/v1/auth/otp", "", "", map[string]string{"destination": phone}); rec.Code != http.StatusAccepted {
		t.Fatalf("request otp: got %d, want 202: %s", rec.Code, rec.Body)
	}
	code := codeFrom(t, ch.last.Body)

	t.Run("rejects a wrong code", func(t *testing.T) {
		wrong := "000000"
		if wrong == code {
			wrong = "111111"
		}
		rec := do(t, h, "POST", "/v1/auth/otp/verify", "", "",
			map[string]string{"destination": phone, "code": wrong})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401: %s", rec.Code, rec.Body)
		}
	})

	rec := do(t, h, "POST", "/v1/auth/otp/verify", "", "",
		map[string]string{"destination": phone, "code": code, "full_name": "আয়েশা রহমান"})
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: got %d, want 200: %s", rec.Code, rec.Body)
	}

	var session struct {
		Token string `json:"token"`
		User  struct {
			ID       string `json:"id"`
			FullName string `json:"full_name"`
		} `json:"user"`
		Memberships []identity.Membership `json:"memberships"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.Token == "" || len(session.Memberships) != 0 {
		t.Fatalf("want a token and no memberships, got %+v", session)
	}
	if session.User.FullName != "আয়েশা রহমান" {
		t.Errorf("bengali name round-trip: got %q", session.User.FullName)
	}

	t.Run("a used code cannot be replayed", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/auth/otp/verify", "", "", map[string]string{"destination": phone, "code": code})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects a missing or bogus token", func(t *testing.T) {
		for _, token := range []string{"", "not-a-real-token-but-long-enough-to-pass"} {
			if rec := do(t, h, "GET", "/v1/me", token, "", nil); rec.Code != http.StatusUnauthorized {
				t.Errorf("token %q: got %d, want 401", token, rec.Code)
			}
		}
	})

	if rec := do(t, h, "GET", "/v1/me", session.Token, "", nil); rec.Code != http.StatusOK {
		t.Fatalf("me: got %d, want 200: %s", rec.Code, rec.Body)
	}

	t.Run("no tenant header is a clear error", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/tenant", session.Token, "", nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d, want 400", rec.Code)
		}
	})

	t.Run("a non-member is forbidden", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/tenant", session.Token, slug, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403", rec.Code)
		}
	})

	// Enrol the user, then the same token works inside the tenant.
	userID := uuid.MustParse(session.User.ID)
	tenant, err := store.Unscoped().ProvisionTenant(ctx, database.ProvisionTenantParams{
		Slug: slug, Name: "Darul Uloom", Kind: database.TenantKindInstitution,
		DefaultDir: database.TextDirRtl, Locale: "ar", Currency: "BDT",
	})
	if err != nil {
		t.Fatalf("provision tenant: %v", err)
	}
	err = store.InTenant(ctx, tenant.ID, func(q *database.Queries) error {
		_, err := q.CreateMembership(ctx, database.CreateMembershipParams{
			TenantID: tenant.ID, UserID: userID, Role: database.MemberRoleInstructor,
		})
		return err
	})
	if err != nil {
		t.Fatalf("create membership: %v", err)
	}

	t.Run("a member sees the tenant", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/tenant", session.Token, slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body)
		}
		var got map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode tenant: %v", err)
		}
		if got["default_dir"] != "rtl" || got["role"] != "instructor" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("an instructor may list members", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/tenant/members", session.Token, slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects an out-of-range limit", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/tenant/members?limit=9999", session.Token, slug, nil)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422", rec.Code)
		}
	})

	t.Run("logout ends the session", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/auth/logout", session.Token, "", nil); rec.Code != http.StatusNoContent {
			t.Fatalf("logout: got %d, want 204", rec.Code)
		}
		if rec := do(t, h, "GET", "/v1/me", session.Token, "", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("revoked token still works: got %d", rec.Code)
		}
	})
}
