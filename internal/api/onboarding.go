package api

import (
	"net/http"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type newSchoolRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Kind     string `json:"kind"`
	Dir      string `json:"dir"`
	Locale   string `json:"locale"`
	Currency string `json:"currency"`
}

// createSchool is self service: anyone signed in may open a school and becomes
// its owner. Provisioning runs as the owner role, the membership as the tenant.
func (s *Server) createSchool(w http.ResponseWriter, r *http.Request) error {
	var body newSchoolRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 200)
	if err != nil {
		return err
	}
	slug := strings.ToLower(strings.TrimSpace(body.Slug))
	if slug == "" {
		slug = Slugify(name)
	}
	if !slugRe.MatchString(slug) {
		return invalid("slug", "Use lowercase letters, numbers and hyphens.")
	}

	kind := database.TenantKind(strings.TrimSpace(body.Kind))
	switch kind {
	case database.TenantKindInstitution, database.TenantKindCreator, database.TenantKindCorporate:
	case "":
		kind = database.TenantKindInstitution
	default:
		return invalid("kind", "Choose an institution, a creator or a company.")
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	locale := strings.TrimSpace(body.Locale)
	if locale == "" {
		locale = "en"
	}
	currency := strings.ToUpper(strings.TrimSpace(body.Currency))
	if currency == "" {
		currency = "BDT"
	}
	if len(currency) != 3 {
		return invalid("currency", "Use a three letter currency code.")
	}

	tenant, err := s.store.Unscoped().ProvisionTenant(r.Context(), database.ProvisionTenantParams{
		Slug: slug, Name: name, Kind: kind, DefaultDir: dir, Locale: locale, Currency: currency,
	})
	if isUniqueViolation(err) {
		return &httpx.Error{Status: http.StatusConflict, Code: "slug_taken",
			Message: "That address is taken. Try another.", Field: "slug"}
	}
	if err != nil {
		return err
	}

	userID := Authenticated(r.Context()).UserID
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		_, err := q.CreateMembership(r.Context(), database.CreateMembershipParams{
			TenantID: tenant.ID, UserID: userID, Role: database.MemberRoleOwner,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, tenant)
}
