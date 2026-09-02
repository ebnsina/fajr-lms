package api

import (
	"net/http"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/media"
)

// Server holds the dependencies every endpoint shares.
type Server struct {
	store    *database.Store
	identity *identity.Service
	media    *media.Registry
}

func NewServer(store *database.Store, ident *identity.Service, registry *media.Registry) *Server {
	return &Server{store: store, identity: ident, media: registry}
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

	// Reading the catalog is open to any member; authoring is not.
	mux.Handle("GET /v1/courses", inTenant(httpx.Handler(s.listCourses)))
	mux.Handle("GET /v1/courses/{slug}", inTenant(httpx.Handler(s.courseOutline)))

	teaches := func(h httpx.Handler) http.Handler {
		return inTenant(RequireRole("owner", "admin", "instructor")(h))
	}
	mux.Handle("POST /v1/courses", teaches(s.createCourse))
	mux.Handle("PUT /v1/courses/{id}/status", teaches(s.setCourseStatus))
	mux.Handle("POST /v1/courses/{id}/modules", teaches(s.createModule))
	mux.Handle("POST /v1/modules/{id}/lessons", teaches(s.createLesson))
	mux.Handle("PUT /v1/lessons/{id}/position", teaches(s.moveLesson))
	mux.Handle("DELETE /v1/lessons/{id}", teaches(s.deleteLesson))

	mux.Handle("GET /v1/media/providers", inTenant(httpx.Handler(s.mediaProviders)))
	mux.Handle("GET /v1/media/{id}/playback", inTenant(httpx.Handler(s.mediaPlayback)))
	mux.Handle("POST /v1/media", teaches(s.ingestMedia))
	mux.Handle("PUT /v1/lessons/{id}/media", teaches(s.attachMedia))
	mux.Handle("GET /v1/media/usage", inTenant(RequireRole("owner", "admin")(httpx.Handler(s.mediaUsage))))

	return mux
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) error {
	if err := s.store.Health(r.Context()); err != nil {
		return httpx.Errorf(http.StatusServiceUnavailable, "database_unavailable", "Database is not reachable.")
	}
	return httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
