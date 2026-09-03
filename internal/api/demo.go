package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/demo"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
)

// The demo hands out a look at a school that already has a term's work in it.
// One school per kind of customer, shared by everyone who asks and read-only
// for all of them, so nobody's clicking spoils the next visitor's look.

type demoRequest struct {
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Organisation string `json:"organisation"`
	Role         string `json:"role"`
	Learners     string `json:"learners"`
	Runs         string `json:"runs"`
	Note         string `json:"note"`
}

// ponytail: nothing throttles this yet; a lead and a user row are all an abuser
// gets. Rate limit by IP if the leads table ever fills with noise.
func (s *Server) startDemo(w http.ResponseWriter, r *http.Request) error {
	var body demoRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	name, err := requireText("full_name", body.FullName, 200)
	if err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if _, err := identity.ParseDestination(email); err != nil || !strings.Contains(email, "@") {
		return invalid("email", "Enter a work email address we can reach you at.")
	}
	kind, ok := demo.Find(body.Runs)
	if !ok {
		return invalid("runs", "Tell us what you run, so the demo matches it.")
	}

	// The lead is the whole point of asking, so it is written before anything
	// slower can fail.
	if _, err := s.store.Unscoped().RecordDemoLead(r.Context(), database.RecordDemoLeadParams{
		FullName: name, Email: email, Phone: trim(body.Phone, 40),
		Organisation: trim(body.Organisation, 200), Role: trim(body.Role, 120),
		Learners: trim(body.Learners, 40), Runs: kind.Slug, Note: trim(body.Note, 2000),
	}); err != nil {
		return err
	}

	tenant, err := s.demoSchool(r.Context(), kind)
	if err != nil {
		return err
	}

	session, err := s.identity.SignInUnverified(r.Context(), email, name, r.UserAgent(), clientIP(r))
	if err != nil {
		return err
	}

	// Admin, so every screen opens; the demo tenant refuses writes on its own.
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		_, err := q.CreateMembership(r.Context(), database.CreateMembershipParams{
			TenantID: tenant.ID, UserID: session.UserID, Role: database.MemberRoleAdmin,
		})
		if isUniqueViolation(err) {
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}

	return httpx.JSON(w, http.StatusOK, map[string]any{
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
		"tenant":     tenant.Slug,
		"name":       tenant.Name,
	})
}

// demoSchool returns the shared school for a kind, building and filling it the
// first time anybody asks for it.
func (s *Server) demoSchool(ctx context.Context, kind demo.Kind) (database.Tenant, error) {
	slug := "demo-" + kind.Slug
	tenant, err := s.identity.ResolveTenant(ctx, slug)
	if err == nil {
		return tenant, nil
	}
	if !errors.Is(err, identity.ErrNoMembership) {
		return database.Tenant{}, err
	}

	tenant, err = s.store.Unscoped().ProvisionTenant(ctx, database.ProvisionTenantParams{
		Slug: slug, Name: kind.Name, Kind: kind.Tenant, DefaultDir: kind.Dir,
		Locale: kind.Locale, Currency: kind.Currency,
	})
	if isUniqueViolation(err) {
		// Somebody else asked first; theirs is the one being filled.
		return s.identity.ResolveTenant(ctx, slug)
	}
	if err != nil {
		return database.Tenant{}, err
	}

	err = s.store.InTenant(ctx, tenant.ID, func(q *database.Queries) error {
		return q.MarkTenantDemo(ctx, database.MarkTenantDemoParams{
			ID: tenant.ID, Institution: kind.Institution,
		})
	})
	if err != nil {
		return database.Tenant{}, err
	}
	tenant.Demo, tenant.Institution = true, kind.Institution

	if err := demo.Seed(ctx, s.store, tenant, kind, newSerial); err != nil {
		return database.Tenant{}, err
	}
	return tenant, nil
}

// demoKinds is what the public form offers, so the page and the seeder cannot
// drift apart.
func (s *Server) demoKinds(w http.ResponseWriter, r *http.Request) error {
	out := make([]map[string]string, 0, len(demo.Kinds))
	for _, kind := range demo.Kinds {
		out = append(out, map[string]string{"slug": kind.Slug, "label": kind.Label, "name": kind.Name})
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"kinds": out})
}

func trim(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}
