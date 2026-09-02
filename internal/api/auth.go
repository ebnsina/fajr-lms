// Package api wires HTTP endpoints onto the domain services.
package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
)

type ctxKey int

const (
	sessionKey ctxKey = iota
	tenantKey
	roleKey
)

// Authenticated pulls the caller's session, panicking only on a routing mistake.
func Authenticated(ctx context.Context) identity.Session {
	s, ok := ctx.Value(sessionKey).(identity.Session)
	if !ok {
		panic("api: handler requires RequireAuth middleware")
	}
	return s
}

// CurrentTenant returns the tenant this request acts in.
func CurrentTenant(ctx context.Context) database.Tenant {
	t, ok := ctx.Value(tenantKey).(database.Tenant)
	if !ok {
		panic("api: handler requires RequireTenant middleware")
	}
	return t
}

func CurrentRole(ctx context.Context) string {
	r, _ := ctx.Value(roleKey).(string)
	return r
}

// RequireAuth rejects any request without a live bearer session.
func (s *Server) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			httpx.WriteError(w, r, httpx.ErrUnauthorized)
			return
		}

		session, err := s.identity.Authenticate(r.Context(), token)
		if errors.Is(err, identity.ErrInvalidSession) {
			httpx.WriteError(w, r, httpx.ErrUnauthorized)
			return
		}
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey, session)))
	})
}

// RequireTenant resolves the X-Fajr-Tenant slug and checks membership.
func (s *Server) RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimSpace(r.Header.Get("X-Fajr-Tenant"))
		if slug == "" {
			httpx.WriteError(w, r, httpx.Errorf(http.StatusBadRequest, "tenant_required", "Set the X-Fajr-Tenant header."))
			return
		}

		session := Authenticated(r.Context())
		tenant, member, err := s.identity.MembershipIn(r.Context(), session.UserID, slug)
		if errors.Is(err, identity.ErrNoMembership) {
			httpx.WriteError(w, r, httpx.ErrForbidden)
			return
		}
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}

		ctx := context.WithValue(r.Context(), tenantKey, tenant)
		ctx = context.WithValue(ctx, roleKey, member.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole allows only the listed roles through.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[CurrentRole(r.Context())] {
				httpx.WriteError(w, r, httpx.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if len(header) < 8 || !strings.EqualFold(header[:7], "bearer ") {
		return "", false
	}
	token := strings.TrimSpace(header[7:])
	return token, token != ""
}

func clientIP(r *http.Request) *netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}
	return &addr
}
