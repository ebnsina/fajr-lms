package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/api"
)

// stubDNS answers from a map, so a test never touches the network.
type stubDNS struct{ records map[string][]string }

func (s stubDNS) LookupTXT(_ context.Context, name string) ([]string, error) {
	values, ok := s.records[name]
	if !ok {
		return nil, fmt.Errorf("no such host")
	}
	return values, nil
}

func TestCustomDomain(t *testing.T) {
	h, server, ch, store := notifyHarness(t)
	dns := stubDNS{records: map[string][]string{}}
	server.UseDNS(dns)

	owner := enroll(t, h, ch, store, "owner")
	other := enroll(t, h, ch, store, "owner")
	domain := fmt.Sprintf("school%09d.example", rand.IntN(1_000_000_000))

	rec := do(t, h, "PUT", "/v1/site/domain", owner.token, owner.slug,
		map[string]any{"domain": "HTTPS://" + strings.ToUpper(domain) + "/"})
	if rec.Code != http.StatusOK {
		t.Fatalf("set domain: got %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Domain   string            `json:"domain"`
		Verified bool              `json:"verified"`
		Record   map[string]string `json:"record"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Domain != domain {
		t.Fatalf("the address was not tidied up: %q", out.Domain)
	}
	if out.Verified {
		t.Fatal("a domain nobody has proved should not be verified")
	}

	t.Run("an unproved domain does not resolve", func(t *testing.T) {
		if rec := do(t, h, "GET", "/site/resolve?host="+domain, "", "", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("verifying without the record fails", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/site/domain/verify", owner.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a wrong value fails", func(t *testing.T) {
		dns.records[out.Record["name"]] = []string{"fajr-site-verification=somebody-else"}
		rec := do(t, h, "POST", "/v1/site/domain/verify", owner.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("the right value verifies and the host resolves", func(t *testing.T) {
		dns.records[out.Record["name"]] = []string{"other stuff", out.Record["value"]}
		if rec := do(t, h, "POST", "/v1/site/domain/verify", owner.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("verify: got %d: %s", rec.Code, rec.Body)
		}

		rec := do(t, h, "GET", "/site/resolve?host="+domain+":443", "", "", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("resolve: got %d: %s", rec.Code, rec.Body)
		}
		var resolved struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resolved); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resolved.Slug != owner.slug {
			t.Errorf("resolved to %q, want %q", resolved.Slug, owner.slug)
		}
	})

	t.Run("two schools cannot hold one domain", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/site/domain", other.token, other.slug,
			map[string]any{"domain": domain})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("something that is not a domain is refused", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/site/domain", owner.token, owner.slug,
			map[string]any{"domain": "not a domain"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("dropping the domain stops the site answering there", func(t *testing.T) {
		if rec := do(t, h, "DELETE", "/v1/site/domain", owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("delete: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "GET", "/site/resolve?host="+domain, "", "", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	var _ api.DNSLookup = dns
}
