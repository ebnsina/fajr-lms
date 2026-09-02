package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
)

// actor is a logged-in user together with the tenant they act in.
type actor struct {
	token  string
	userID uuid.UUID
	slug   string
}

func login(t *testing.T, h http.Handler, ch *captureChannel) (string, uuid.UUID) {
	t.Helper()
	phone := fmt.Sprintf("+8809%08d", rand.IntN(100_000_000))

	if rec := do(t, h, "POST", "/v1/auth/otp", "", "", map[string]string{"destination": phone}); rec.Code != http.StatusAccepted {
		t.Fatalf("request otp: got %d: %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "POST", "/v1/auth/otp/verify", "", "",
		map[string]string{"destination": phone, "code": codeFrom(t, ch.last.Body), "full_name": "Test " + phone})
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: got %d: %s", rec.Code, rec.Body)
	}

	var session struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	return session.Token, uuid.MustParse(session.User.ID)
}

// enrol creates a fresh tenant and a user holding the given role in it.
func enrol(t *testing.T, h http.Handler, ch *captureChannel, store *database.Store, role string) actor {
	t.Helper()
	ctx := context.Background()
	slug := fmt.Sprintf("t-%09d", rand.IntN(1_000_000_000))

	tenant, err := store.Unscoped().ProvisionTenant(ctx, database.ProvisionTenantParams{
		Slug: slug, Name: "Tenant " + slug, Kind: database.TenantKindInstitution,
		DefaultDir: database.TextDirAuto, Locale: "en", Currency: "BDT",
	})
	if err != nil {
		t.Fatalf("provision tenant: %v", err)
	}

	token, userID := login(t, h, ch)
	addMember(t, store, tenant.ID, userID, role)
	return actor{token: token, userID: userID, slug: slug}
}

// enrolIn adds a new user to an existing tenant.
func enrolIn(t *testing.T, h http.Handler, ch *captureChannel, store *database.Store, slug, role string) actor {
	t.Helper()
	tenant, err := store.Unscoped().ResolveTenant(context.Background(), slug)
	if err != nil {
		t.Fatalf("resolve tenant %s: %v", slug, err)
	}
	token, userID := login(t, h, ch)
	addMember(t, store, tenant.ID, userID, role)
	return actor{token: token, userID: userID, slug: slug}
}

func addMember(t *testing.T, store *database.Store, tenantID, userID uuid.UUID, role string) {
	t.Helper()
	ctx := context.Background()
	err := store.InTenant(ctx, tenantID, func(q *database.Queries) error {
		_, err := q.CreateMembership(ctx, database.CreateMembershipParams{
			TenantID: tenantID, UserID: userID, Role: database.MemberRole(role),
		})
		return err
	})
	if err != nil {
		t.Fatalf("create %s membership: %v", role, err)
	}
}

type outlineResponse struct {
	Course struct {
		ID     string `json:"id"`
		Slug   string `json:"slug"`
		Status string `json:"status"`
	} `json:"course"`
	Modules []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Lessons []struct {
			ID       string  `json:"id"`
			Title    string  `json:"title"`
			Position float64 `json:"position"`
		} `json:"lessons"`
	} `json:"modules"`
}

func readOutline(t *testing.T, h http.Handler, a actor, slug string) outlineResponse {
	t.Helper()
	rec := do(t, h, "GET", "/v1/courses/"+slug, a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("outline: got %d: %s", rec.Code, rec.Body)
	}
	var out outlineResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode outline: %v", err)
	}
	return out
}

// createdID reads the id from a 201 response, failing the test otherwise.
func createdID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
	}
	var got struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode id: %v", err)
	}
	return got.ID
}

var errBroken = errors.New("gateway unreachable")

func osGetenv(key string) string { return os.Getenv(key) }
