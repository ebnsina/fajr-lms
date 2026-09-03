package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// DNSLookup reads the TXT records of a name. It is an interface so a test can
// answer without a network, and so a resolver of our own can replace it later.
type DNSLookup interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

type netLookup struct{}

func (netLookup) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return net.DefaultResolver.LookupTXT(ctx, name)
}

// UseDNS replaces the resolver used to check a domain belongs to a school.
func (s *Server) UseDNS(lookup DNSLookup) { s.dns = lookup }

func parseDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(raw))
	domain = strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	domain = strings.TrimSuffix(strings.TrimPrefix(domain, "."), "/")
	if domain == "" {
		return "", invalid("domain", "Enter the domain your site should answer on.")
	}
	if len(domain) > 253 || !strings.Contains(domain, ".") || strings.Contains(domain, " ") {
		return "", invalid("domain", "That does not look like a domain name.")
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 {
			return "", invalid("domain", "That does not look like a domain name.")
		}
	}
	return domain, nil
}

// domainRecord is what the school has to publish in DNS before we will serve
// their site on it, so nobody can point a domain at somebody else's school.
func domainRecord(domain, token string) map[string]string {
	return map[string]string{
		"type": "TXT", "name": "_fajr." + domain, "value": "fajr-site-verification=" + token,
	}
}

func (s *Server) setDomain(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Domain string `json:"domain"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	domain, err := parseDomain(body.Domain)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	token := strings.ToLower(rand.Text()[:20])
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		_, err := q.SetCustomDomain(r.Context(), database.SetCustomDomainParams{
			ID: tenant.ID, CustomDomain: &domain, DomainToken: token,
		})
		return err
	})
	if isUniqueViolation(err) {
		return &httpx.Error{Status: http.StatusConflict, Code: "domain_taken",
			Message: "Another school is already using that domain.", Field: "domain"}
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"domain": domain, "verified": false, "record": domainRecord(domain, token),
	})
}

// verifyDomain looks for the record and, finding it, starts serving the site
// there. It is safe to call again: a school will retry while DNS spreads.
func (s *Server) verifyDomain(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	if tenant.CustomDomain == nil || *tenant.CustomDomain == "" {
		return httpx.Errorf(http.StatusConflict, "no_domain", "Name a domain first.")
	}

	domain, token := *tenant.CustomDomain, tenant.DomainToken
	records, err := s.dns.LookupTXT(r.Context(), "_fajr."+domain)
	if err != nil {
		return httpx.Errorf(http.StatusConflict, "record_not_found",
			"No record found yet. DNS can take a while; try again in a few minutes.")
	}

	want := "fajr-site-verification=" + token
	found := false
	for _, record := range records {
		if strings.TrimSpace(record) == want {
			found = true
			break
		}
	}
	if !found {
		return httpx.Errorf(http.StatusConflict, "record_mismatch",
			fmt.Sprintf("The record at _fajr.%s does not carry this school's value yet.", domain))
	}

	var updated database.Tenant
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		updated, err = q.MarkDomainVerified(r.Context(), tenant.ID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"domain": domain, "verified": updated.DomainVerifiedAt.Valid,
	})
}

func (s *Server) clearDomain(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		_, err := q.ClearCustomDomain(r.Context(), tenant.ID)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) siteDomain(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	out := map[string]any{"domain": "", "verified": false}
	if tenant.CustomDomain != nil && *tenant.CustomDomain != "" {
		out["domain"] = *tenant.CustomDomain
		out["verified"] = tenant.DomainVerifiedAt.Valid
		out["record"] = domainRecord(*tenant.CustomDomain, tenant.DomainToken)
	}
	return httpx.JSON(w, http.StatusOK, out)
}

// resolveHost answers which school a hostname belongs to, for the front end to
// serve the right site. Unverified domains are not in the view it reads.
func (s *Server) resolveHost(w http.ResponseWriter, r *http.Request) error {
	host := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("host")))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "" {
		return invalid("host", "Name the host to resolve.")
	}

	tenant, err := s.store.Unscoped().ResolveDomain(r.Context(), host)
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"slug": tenant.Slug, "name": tenant.Name,
	})
}
