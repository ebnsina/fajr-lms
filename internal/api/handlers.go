package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
)

type otpRequest struct {
	Destination string `json:"destination"`
}

func (s *Server) requestOTP(w http.ResponseWriter, r *http.Request) error {
	var body otpRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	dest, err := identity.ParseDestination(body.Destination)
	if errors.Is(err, identity.ErrInvalidDestination) {
		return &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_destination",
			Message: "Enter a phone number in +8801… form, or an email address.", Field: "destination"}
	}

	switch err := s.identity.RequestOTP(r.Context(), dest); {
	case errors.Is(err, identity.ErrRateLimited):
		return httpx.Errorf(http.StatusTooManyRequests, "rate_limited", "Too many codes requested. Try again in an hour.")
	case err != nil:
		return err
	}
	// Always 202, so the response cannot be used to enumerate accounts.
	return httpx.JSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

type verifyRequest struct {
	Destination string `json:"destination"`
	Code        string `json:"code"`
	FullName    string `json:"full_name"`
}

type sessionResponse struct {
	Token       string                `json:"token"`
	ExpiresAt   time.Time             `json:"expires_at"`
	User        userResponse          `json:"user"`
	Memberships []identity.Membership `json:"memberships"`
}

type userResponse struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
}

func (s *Server) verifyOTP(w http.ResponseWriter, r *http.Request) error {
	var body verifyRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	dest, err := identity.ParseDestination(body.Destination)
	if errors.Is(err, identity.ErrInvalidDestination) {
		return &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_destination",
			Message: "Enter a phone number in +8801… form, or an email address.", Field: "destination"}
	}
	if len(body.Code) != 6 {
		return &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_code",
			Message: "The code is six digits.", Field: "code"}
	}

	session, members, err := s.identity.VerifyOTP(r.Context(), dest, body.Code, body.FullName, r.UserAgent(), clientIP(r))
	switch {
	case errors.Is(err, identity.ErrInvalidCode):
		return httpx.Errorf(http.StatusUnauthorized, "invalid_code", "That code is wrong or has expired.")
	case err != nil:
		return err
	}

	return httpx.JSON(w, http.StatusOK, sessionResponse{
		Token: session.Token, ExpiresAt: session.ExpiresAt,
		User:        userResponse{ID: session.UserID.String(), FullName: session.FullName},
		Memberships: members,
	})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	session := Authenticated(r.Context())
	members, err := s.identity.Memberships(r.Context(), session.UserID)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"user":        userResponse{ID: session.UserID.String(), FullName: session.FullName},
		"memberships": members,
	})
}

// listMyTenants answers before a tenant is chosen, so it takes no tenant header.
func (s *Server) listMyTenants(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.store.Unscoped().ListUserTenants(r.Context(), Authenticated(r.Context()).UserID)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"tenants": rows})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	if err := s.identity.Logout(r.Context(), Authenticated(r.Context()).ID); err != nil {
		return err
	}
	return httpx.NoContent(w)
}

func (s *Server) currentTenant(w http.ResponseWriter, r *http.Request) error {
	t := CurrentTenant(r.Context())
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"id": t.ID, "slug": t.Slug, "name": t.Name, "kind": t.Kind,
		"default_dir": t.DefaultDir, "locale": t.Locale, "currency": t.Currency,
		"role": CurrentRole(r.Context()),
	})
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}

	var (
		rows  []database.ListTenantMembersRow
		total int64
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if rows, err = q.ListTenantMembers(r.Context(), database.ListTenantMembersParams{
			PageLimit: limit, PageOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountTenantMembers(r.Context())
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"members": rows, "total": total})
}

// pagination reads limit and offset, rejecting values a client should not send.
func pagination(r *http.Request) (int32, int32, error) {
	limit, offset := int32(50), int32(0)

	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > 200 {
			return 0, 0, &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_limit",
				Message: "limit must be between 1 and 200.", Field: "limit"}
		}
		limit = int32(v)
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return 0, 0, &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_offset",
				Message: "offset must be zero or greater.", Field: "offset"}
		}
		offset = int32(v)
	}
	return limit, offset, nil
}
