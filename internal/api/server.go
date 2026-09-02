package api

import (
	"net/http"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
)

// Server holds the dependencies every endpoint shares.
type Server struct {
	store    *database.Store
	identity *identity.Service
}

func NewServer(store *database.Store, ident *identity.Service) *Server {
	return &Server{store: store, identity: ident}
}

// Routes returns the full API surface, health probes included.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /healthz", httpx.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	mux.Handle("GET /readyz", httpx.Handler(s.ready))

	mux.Handle("POST /v1/auth/otp", httpx.Handler(s.requestOTP))
	mux.Handle("POST /v1/auth/otp/verify", httpx.Handler(s.verifyOTP))

	authed := func(h http.Handler) http.Handler { return s.RequireAuth(h) }
	mux.Handle("GET /v1/me", authed(httpx.Handler(s.me)))
	mux.Handle("POST /v1/auth/logout", authed(httpx.Handler(s.logout)))

	inTenant := func(h http.Handler) http.Handler { return s.RequireAuth(s.RequireTenant(h)) }
	mux.Handle("GET /v1/tenant", inTenant(httpx.Handler(s.currentTenant)))
	mux.Handle("GET /v1/tenant/members", inTenant(
		RequireRole("owner", "admin", "instructor")(httpx.Handler(s.listMembers))))

	return mux
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) error {
	if err := s.store.Health(r.Context()); err != nil {
		return httpx.Errorf(http.StatusServiceUnavailable, "database_unavailable", "Database is not reachable.")
	}
	return httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
