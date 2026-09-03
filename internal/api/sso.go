package api

import (
	"net/http"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/sso"
)

const ssoLoginTTL = 600

// ssoOffered says whether a school has sign-in with their own accounts, so the
// login page can offer it. It is public: the answer is a label, nothing more.
func (s *Server) ssoOffered(w http.ResponseWriter, r *http.Request) error {
	provider, err := s.store.Unscoped().SSOProviderFor(r.Context(), strings.ToLower(r.PathValue("slug")))
	if database.IsNotFound(err) {
		return httpx.JSON(w, http.StatusOK, map[string]any{"available": false})
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"available": true, "label": provider.Label})
}

type ssoStartRequest struct {
	RedirectURI string `json:"redirect_uri"`
}

// startSSO hands back where to send the browser, and remembers what the answer
// will have to match.
func (s *Server) startSSO(w http.ResponseWriter, r *http.Request) error {
	var body ssoStartRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	redirectURI := strings.TrimSpace(body.RedirectURI)
	if !strings.HasPrefix(redirectURI, s.publicURL+"/") {
		return invalid("redirect_uri", "That is not one of this installation's addresses.")
	}

	q := s.store.Unscoped()
	provider, err := q.SSOProviderFor(r.Context(), strings.ToLower(r.PathValue("slug")))
	if database.IsNotFound(err) {
		return httpx.Errorf(http.StatusNotFound, "no_sso",
			"This school does not offer sign-in with a school account.")
	}
	if err != nil {
		return err
	}

	login, err := s.sso.Start(r.Context(), sso.Provider{
		Issuer: provider.Issuer, ClientID: provider.ClientID, ClientSecret: provider.ClientSecret,
	}, redirectURI)
	if err != nil {
		return httpx.Errorf(http.StatusBadGateway, "sso_unreachable", err.Error())
	}

	if _, err := q.StartSSOLogin(r.Context(), database.StartSSOLoginParams{
		State: login.State, ProviderID: provider.ID, Nonce: login.Nonce,
		Verifier: login.Verifier, RedirectUri: redirectURI, TtlS: ssoLoginTTL,
	}); err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"url": login.URL, "state": login.State})
}

type ssoExchangeRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// finishSSO turns the provider's answer into a session here.
func (s *Server) finishSSO(w http.ResponseWriter, r *http.Request) error {
	var body ssoExchangeRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if strings.TrimSpace(body.Code) == "" || strings.TrimSpace(body.State) == "" {
		return invalid("code", "That sign-in did not come back complete. Start again.")
	}

	q := s.store.Unscoped()
	// Taking the state spends it, so a replayed callback finds nothing.
	login, err := q.TakeSSOLogin(r.Context(), body.State)
	if database.IsNotFound(err) {
		return httpx.Errorf(http.StatusUnauthorized, "sso_expired",
			"That sign-in has expired or was already used. Start again.")
	}
	if err != nil {
		return err
	}

	provider, err := q.SSOProviderByID(r.Context(), login.ProviderID)
	if database.IsNotFound(err) {
		return httpx.Errorf(http.StatusConflict, "no_sso",
			"This school has switched off sign-in with a school account.")
	}
	if err != nil {
		return err
	}

	person, err := s.sso.Exchange(r.Context(), sso.Provider{
		Issuer: provider.Issuer, ClientID: provider.ClientID, ClientSecret: provider.ClientSecret,
	}, body.Code, login.Verifier, login.Nonce, login.RedirectUri)
	if err != nil {
		return httpx.Errorf(http.StatusUnauthorized, "sso_refused", err.Error())
	}
	if !sso.AddressAllowed(person.Email, provider.AllowedDomains) {
		return httpx.Errorf(http.StatusForbidden, "address_not_allowed",
			"That address is not one this school signs in with.")
	}

	session, members, err := s.identity.SignInFederated(r.Context(), provider.ID,
		person.Subject, person.Email, person.Name, r.UserAgent(), clientIP(r))
	if err != nil {
		return err
	}

	// A first sign-in joins the school, if the school asked for that.
	joined := false
	for _, member := range members {
		if member.TenantID == provider.TenantID {
			joined = true
		}
	}
	if !joined {
		if !provider.AutoJoin {
			return httpx.Errorf(http.StatusForbidden, "not_a_member",
				"Your account signed in, but you are not a member of this school yet.")
		}
		if err := q.JoinBySSO(r.Context(), database.JoinBySSOParams{
			TenantID: provider.TenantID, UserID: session.UserID, JoinRole: provider.JoinRole,
		}); err != nil {
			return err
		}
		if members, err = s.identity.Memberships(r.Context(), session.UserID); err != nil {
			return err
		}
	}

	return httpx.JSON(w, http.StatusOK, sessionResponse{
		Token: session.Token, ExpiresAt: session.ExpiresAt,
		User:        userResponse{ID: session.UserID.String(), FullName: session.FullName},
		Memberships: members,
	})
}

type identityProviderRequest struct {
	Label          string   `json:"label"`
	Issuer         string   `json:"issuer"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	AllowedDomains []string `json:"allowed_domains"`
	JoinRole       string   `json:"join_role"`
	AutoJoin       *bool    `json:"auto_join"`
	Enabled        *bool    `json:"enabled"`
}

// setIdentityProvider is the school's own settings for signing in. The secret
// is write-only: an empty one on an update keeps what is stored.
func (s *Server) setIdentityProvider(w http.ResponseWriter, r *http.Request) error {
	var body identityProviderRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	issuer := strings.TrimSuffix(strings.TrimSpace(body.Issuer), "/")
	if !strings.HasPrefix(issuer, "https://") || len(issuer) > 300 {
		return invalid("issuer", "The issuer is an https address, as the provider gives it.")
	}
	clientID := strings.TrimSpace(body.ClientID)
	if clientID == "" || len(clientID) > 300 {
		return invalid("client_id", "Give the client id from the provider.")
	}
	label := strings.TrimSpace(body.Label)
	if label == "" {
		label = "Your school account"
	}
	if len(label) > 60 {
		return invalid("label", "Keep the button's words under 60 characters.")
	}

	domains := make([]string, 0, len(body.AllowedDomains))
	for _, domain := range body.AllowedDomains {
		clean := strings.ToLower(strings.TrimSpace(domain))
		if clean == "" {
			continue
		}
		if !strings.Contains(clean, ".") || strings.ContainsAny(clean, " @/") {
			return invalid("allowed_domains", "Each entry is a domain, like school.edu.bd.")
		}
		domains = append(domains, clean)
	}

	role := database.MemberRoleStudent
	if raw := strings.TrimSpace(body.JoinRole); raw != "" {
		parsed, err := parseRole(raw)
		if err != nil {
			return err
		}
		role = parsed
	}

	tenant := CurrentTenant(r.Context())
	var provider database.IdentityProvider
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		provider, err = q.SetIdentityProvider(r.Context(), database.SetIdentityProviderParams{
			TenantID: tenant.ID, Label: label, Issuer: issuer, ClientID: clientID,
			ClientSecret: strings.TrimSpace(body.ClientSecret), AllowedDomains: domains,
			JoinRole: role, AutoJoin: body.AutoJoin == nil || *body.AutoJoin,
			Enabled: body.Enabled == nil || *body.Enabled,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, hideSecret(provider))
}

func (s *Server) identityProvider(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	var provider database.IdentityProvider
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		provider, err = q.GetIdentityProvider(r.Context(), tenant.ID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.JSON(w, http.StatusOK, map[string]any{"configured": false})
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, hideSecret(provider))
}

func (s *Server) clearIdentityProvider(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	var rows int64
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		rows, err = q.DeleteIdentityProvider(r.Context(), tenant.ID)
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// hideSecret keeps the client secret write-only: it is shown as set or not.
func hideSecret(p database.IdentityProvider) map[string]any {
	return map[string]any{
		"configured": true, "id": p.ID, "label": p.Label, "issuer": p.Issuer,
		"client_id": p.ClientID, "has_secret": p.ClientSecret != "",
		"allowed_domains": p.AllowedDomains, "join_role": p.JoinRole,
		"auto_join": p.AutoJoin, "enabled": p.Enabled,
	}
}
