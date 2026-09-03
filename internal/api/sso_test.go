package api_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ebnsina/fajr-lms/internal/api"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/sso"
)

// fakeIDP is an OpenID provider that signs one person in.
type fakeIDP struct {
	url     string
	client  *http.Client
	nonce   string
	email   string
	subject string
}

func newFakeIDP(t *testing.T) *fakeIDP {
	t.Helper()
	// A run of its own, so an earlier run's person is not signed in again.
	unique := rand.IntN(1_000_000)
	idp := &fakeIDP{
		email:   fmt.Sprintf("fatima%06d@school.edu.bd", unique),
		subject: fmt.Sprintf("sub-%06d", unique),
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 idp.url,
			"authorization_endpoint": idp.url + "/authorize",
			"token_endpoint":         idp.url + "/token",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		claims, err := json.Marshal(map[string]any{
			"iss": idp.url, "sub": idp.subject, "aud": "client-1", "nonce": idp.nonce,
			"email": idp.email, "name": "Fatima Rahman", "exp": time.Now().Add(time.Hour).Unix(),
		})
		if err != nil {
			http.Error(w, "encode", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"id_token": fmt.Sprintf("h.%s.s", base64.RawURLEncoding.EncodeToString(claims)),
		})
	})

	server := httptest.NewTLSServer(mux)
	idp.client = server.Client()
	idp.url = server.URL
	t.Cleanup(server.Close)
	return idp
}

// newSSOHarness is the usual harness with an OpenID client that trusts the
// test's own provider.
func newSSOHarness(t *testing.T, client *http.Client) (http.Handler, *captureChannel, *database.Store) {
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
	server := api.NewServer(store, identity.New(store, ch), testRegistry(t), testPayments(t),
		"https://fajr.test")
	server.UseSSO(&sso.Client{HTTP: client})
	return server.Routes(), ch, store
}

func TestSignInWithASchoolAccount(t *testing.T) {
	idp := newFakeIDP(t)
	h, ch, store := newSSOHarness(t, idp.client)
	owner := enroll(t, h, ch, store, "owner")
	const redirect = "https://fajr.test/login/sso"

	t.Run("a school without SSO offers none", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/auth/sso/"+owner.slug, "", "", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if got := rec.Body.String(); !jsonHas(got, `"available":false`) {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("only the office can set it up", func(t *testing.T) {
		learner := enrollIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "PUT", "/v1/sso", learner.token, owner.slug, map[string]any{
			"issuer": idp.url, "client_id": "client-1",
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("an issuer that is not https is refused", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/sso", owner.token, owner.slug, map[string]any{
			"issuer": "http://insecure.example", "client_id": "client-1",
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("the office points the school at its provider", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/sso", owner.token, owner.slug, map[string]any{
			"issuer": idp.url, "client_id": "client-1", "client_secret": "secret-1",
			"allowed_domains": []string{"school.edu.bd"}, "label": "School account",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "secret-1") {
			t.Fatal("the client secret came back out of the API")
		}
	})

	t.Run("the school now offers it", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/auth/sso/"+owner.slug, "", "", nil)
		if !jsonHas(rec.Body.String(), `"available":true`) {
			t.Fatalf("got %s", rec.Body)
		}
	})

	var state string
	t.Run("starting hands back where to send the browser", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/auth/sso/"+owner.slug+"/start", "", "",
			map[string]any{"redirect_uri": redirect})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			URL   string `json:"url"`
			State string `json:"state"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		parsed, err := url.Parse(out.URL)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		idp.nonce = parsed.Query().Get("nonce")
		state = out.State
		if state == "" || idp.nonce == "" {
			t.Fatalf("got %s", rec.Body)
		}
	})

	t.Run("an address off the school's domains is refused", func(t *testing.T) {
		theirs := idp.email
		idp.email = "someone@gmail.com"
		defer func() { idp.email = theirs }()
		rec := do(t, h, "POST", "/v1/auth/sso/finish", "", "",
			map[string]any{"code": "code-1", "state": state})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("signing in joins the school and returns a session", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/auth/sso/"+owner.slug+"/start", "", "",
			map[string]any{"redirect_uri": redirect})
		var start struct {
			URL   string `json:"url"`
			State string `json:"state"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &start); err != nil {
			t.Fatalf("decode: %v", err)
		}
		parsed, err := url.Parse(start.URL)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		idp.nonce = parsed.Query().Get("nonce")

		rec = do(t, h, "POST", "/v1/auth/sso/finish", "", "",
			map[string]any{"code": "code-1", "state": start.State})
		if rec.Code != http.StatusOK {
			t.Fatalf("finish: got %d: %s", rec.Code, rec.Body)
		}
		var session struct {
			Token       string `json:"token"`
			Memberships []struct {
				Role string `json:"role"`
			} `json:"memberships"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if session.Token == "" || len(session.Memberships) != 1 ||
			session.Memberships[0].Role != "student" {
			t.Fatalf("got %s", rec.Body)
		}

		// The session works like any other.
		if rec := do(t, h, "GET", "/v1/me", session.Token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("me: got %d: %s", rec.Code, rec.Body)
		}

		t.Run("the same state cannot be used twice", func(t *testing.T) {
			rec := do(t, h, "POST", "/v1/auth/sso/finish", "", "",
				map[string]any{"code": "code-1", "state": start.State})
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("got %d, want 401: %s", rec.Code, rec.Body)
			}
		})
	})

	t.Run("a redirect address that is not ours is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/auth/sso/"+owner.slug+"/start", "", "",
			map[string]any{"redirect_uri": "https://evil.example/steal"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})
}

func jsonHas(body, fragment string) bool {
	return strings.Contains(strings.ReplaceAll(body, " ", ""), fragment)
}
