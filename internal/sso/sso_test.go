package sso

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeProvider is an OpenID provider that answers just enough to sign in.
type fakeProvider struct {
	server  *httptest.Server
	nonce   string
	aud     string
	issuer  string
	noEmail bool
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	p := &fakeProvider{aud: "client-123"}
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 p.issuer,
			"authorization_endpoint": p.issuer + "/authorize",
			"token_endpoint":         p.issuer + "/token",
			"userinfo_endpoint":      p.issuer + "/userinfo",
		})
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("code_verifier") == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
			return
		}
		email := "fatima@school.edu.bd"
		if p.noEmail {
			email = ""
		}
		json.NewEncoder(w).Encode(map[string]string{
			"id_token": idToken(map[string]any{
				"iss": p.issuer, "sub": "provider-subject-1", "aud": p.aud,
				"nonce": p.nonce, "email": email, "name": "Fatima Rahman",
				"exp": time.Now().Add(time.Hour).Unix(),
			}),
			"access_token": "at-1",
		})
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"email": "fatima@school.edu.bd", "name": "Fatima Rahman",
		})
	})

	p.server = httptest.NewServer(mux)
	p.issuer = p.server.URL
	t.Cleanup(p.server.Close)
	return p
}

func idToken(claims map[string]any) string {
	body, err := json.Marshal(claims)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("header.%s.signature", base64.RawURLEncoding.EncodeToString(body))
}

func TestSignIn(t *testing.T) {
	provider := newFakeProvider(t)
	client := &Client{}
	config := Provider{Issuer: provider.issuer, ClientID: "client-123", ClientSecret: "shh"}
	const redirect = "https://fajr.test/login/sso"

	login, err := client.Start(context.Background(), config, redirect)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	t.Run("the browser is sent with everything it needs back", func(t *testing.T) {
		parsed, err := url.Parse(login.URL)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		query := parsed.Query()
		for _, name := range []string{"state", "nonce", "code_challenge", "client_id", "redirect_uri"} {
			if query.Get(name) == "" {
				t.Fatalf("%s is missing from %s", name, login.URL)
			}
		}
		if query.Get("code_challenge_method") != "S256" {
			t.Fatalf("challenge method is %q", query.Get("code_challenge_method"))
		}
		if query.Get("state") != login.State || query.Get("nonce") != login.Nonce {
			t.Fatal("the URL does not carry the values the callback will check")
		}
	})

	t.Run("the code comes back as a person", func(t *testing.T) {
		provider.nonce = login.Nonce
		person, err := client.Exchange(context.Background(), config, "code-1",
			login.Verifier, login.Nonce, redirect)
		if err != nil {
			t.Fatalf("exchange: %v", err)
		}
		if person.Subject != "provider-subject-1" || person.Email != "fatima@school.edu.bd" {
			t.Fatalf("got %+v", person)
		}
	})

	t.Run("an answer to a different sign-in is refused", func(t *testing.T) {
		provider.nonce = "somebody else's nonce"
		_, err := client.Exchange(context.Background(), config, "code-1",
			login.Verifier, login.Nonce, redirect)
		if err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Fatalf("got %v, want the nonce to be refused", err)
		}
	})

	t.Run("a token meant for another application is refused", func(t *testing.T) {
		provider.nonce = login.Nonce
		provider.aud = "some-other-client"
		defer func() { provider.aud = "client-123" }()
		_, err := client.Exchange(context.Background(), config, "code-1",
			login.Verifier, login.Nonce, redirect)
		if err == nil || !strings.Contains(err.Error(), "another application") {
			t.Fatalf("got %v, want the audience to be refused", err)
		}
	})

	t.Run("an address kept back is read from userinfo", func(t *testing.T) {
		provider.nonce, provider.noEmail = login.Nonce, true
		defer func() { provider.noEmail = false }()
		person, err := client.Exchange(context.Background(), config, "code-1",
			login.Verifier, login.Nonce, redirect)
		if err != nil {
			t.Fatalf("exchange: %v", err)
		}
		if person.Email != "fatima@school.edu.bd" {
			t.Fatalf("got %q from userinfo", person.Email)
		}
	})

	t.Run("a provider calling itself something else is refused", func(t *testing.T) {
		other := &Client{}
		_, err := other.Start(context.Background(),
			Provider{Issuer: provider.issuer + "/tenant-two", ClientID: "client-123"}, redirect)
		if err == nil {
			t.Fatal("an issuer that does not match its own document was accepted")
		}
	})
}

func TestAddressAllowed(t *testing.T) {
	cases := []struct {
		email   string
		domains []string
		want    bool
	}{
		{"fatima@school.edu.bd", nil, true},
		{"fatima@school.edu.bd", []string{"school.edu.bd"}, true},
		{"fatima@SCHOOL.edu.bd", []string{"school.edu.bd"}, true},
		{"fatima@gmail.com", []string{"school.edu.bd"}, false},
		{"fatima@sub.school.edu.bd", []string{"school.edu.bd"}, false},
		{"nonsense", []string{"school.edu.bd"}, false},
	}
	for _, c := range cases {
		if got := AddressAllowed(c.email, c.domains); got != c.want {
			t.Errorf("%q against %v: got %v, want %v", c.email, c.domains, got, c.want)
		}
	}
}
