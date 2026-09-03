package api

import (
	"net/http"

	"github.com/ebnsina/fajr-lms/internal/ai"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/media"
	"github.com/ebnsina/fajr-lms/internal/notify"
	"github.com/ebnsina/fajr-lms/internal/payment"
)

// Server holds the dependencies every endpoint shares.
type Server struct {
	store     *database.Store
	identity  *identity.Service
	media     *media.Registry
	payments  *payment.Registry
	notifier  *notify.Service
	publicURL string
	dns       DNSLookup
	ai        ai.Drafter
}

// UseNotifier wires the notification service after construction, since the
// service records through the server itself.
func (s *Server) UseNotifier(n *notify.Service) { s.notifier = n }

func NewServer(store *database.Store, ident *identity.Service, registry *media.Registry,
	payments *payment.Registry, publicURL string) *Server {
	return &Server{store: store, identity: ident, media: registry, payments: payments,
		publicURL: publicURL, dns: netLookup{}, ai: ai.Off{}}
}

// UseAI replaces the model that drafts teaching material.
func (s *Server) UseAI(drafter ai.Drafter) { s.ai = drafter }

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
	mux.Handle("GET /v1/tenants", authed(httpx.Handler(s.listMyTenants)))
	mux.Handle("POST /v1/tenants", authed(httpx.Handler(s.createSchool)))

	inTenant := func(h http.Handler) http.Handler { return s.RequireAuth(s.RequireTenant(h)) }
	mux.Handle("GET /v1/tenant", inTenant(httpx.Handler(s.currentTenant)))
	mux.Handle("GET /v1/tenant/members", inTenant(
		RequireRole("owner", "admin", "instructor")(httpx.Handler(s.listMembers))))

	runs := func(h httpx.Handler) http.Handler {
		return inTenant(RequireRole("owner", "admin")(h))
	}
	mux.Handle("POST /v1/tenant/members", runs(s.invite))
	mux.Handle("PUT /v1/tenant/members/{id}/role", runs(s.setMemberRole))
	mux.Handle("DELETE /v1/tenant/members/{id}", runs(s.removeMember))

	// Reading the catalog is open to any member; authoring is not.
	mux.Handle("GET /v1/courses", inTenant(httpx.Handler(s.listCourses)))
	mux.Handle("GET /v1/courses/{slug}", inTenant(httpx.Handler(s.courseOutline)))

	teaches := func(h httpx.Handler) http.Handler {
		return inTenant(RequireRole("owner", "admin", "instructor")(h))
	}
	mux.Handle("POST /v1/courses", teaches(s.createCourse))
	mux.Handle("PATCH /v1/courses/{id}", teaches(s.updateCourse))
	mux.Handle("PUT /v1/courses/{id}/status", teaches(s.setCourseStatus))
	mux.Handle("POST /v1/courses/{id}/modules", teaches(s.createModule))
	mux.Handle("POST /v1/modules/{id}/lessons", teaches(s.createLesson))
	mux.Handle("PATCH /v1/modules/{id}", teaches(s.updateModule))
	mux.Handle("PUT /v1/modules/{id}/position", teaches(s.moveModule))
	mux.Handle("DELETE /v1/modules/{id}", teaches(s.deleteModule))
	mux.Handle("PATCH /v1/lessons/{id}", teaches(s.updateLesson))
	mux.Handle("PUT /v1/lessons/{id}/position", teaches(s.moveLesson))
	mux.Handle("DELETE /v1/lessons/{id}", teaches(s.deleteLesson))

	mux.Handle("GET /v1/media/providers", inTenant(httpx.Handler(s.mediaProviders)))
	mux.Handle("GET /v1/media/{id}/playback", inTenant(httpx.Handler(s.mediaPlayback)))
	mux.Handle("POST /v1/media", teaches(s.ingestMedia))
	mux.Handle("POST /v1/media/{id}/complete", teaches(s.completeUpload))
	mux.Handle("PUT /v1/lessons/{id}/media", teaches(s.attachMedia))
	mux.Handle("GET /v1/media/usage", inTenant(RequireRole("owner", "admin")(httpx.Handler(s.mediaUsage))))

	mux.Handle("POST /v1/courses/{id}/enrollments", inTenant(httpx.Handler(s.enroll)))
	mux.Handle("GET /v1/courses/{id}/roster", teaches(s.courseRoster))
	mux.Handle("GET /v1/courses/{id}/progress", inTenant(httpx.Handler(s.myCourseProgress)))
	mux.Handle("GET /v1/enrollments", inTenant(httpx.Handler(s.listMyEnrollments)))
	mux.Handle("DELETE /v1/enrollments/{id}", teaches(s.cancelEnrollment))
	mux.Handle("PUT /v1/lessons/{id}/progress", inTenant(httpx.Handler(s.recordProgress)))

	// Unauthenticated: gateways call this, not users. Everything is re-verified.
	mux.Handle("POST /v1/payment/{tenant}/{provider}/callback", httpx.Handler(s.paymentCallback))
	mux.Handle("GET /v1/payment/{tenant}/{provider}/callback", httpx.Handler(s.paymentCallback))

	mux.Handle("POST /v1/lessons/{id}/quiz", teaches(s.createQuiz))
	mux.Handle("POST /v1/quizzes/{id}/questions", teaches(s.addQuestion))
	mux.Handle("GET /v1/quizzes/{id}", teaches(s.quizSheet))
	mux.Handle("DELETE /v1/questions/{id}", teaches(s.deleteQuestion))
	mux.Handle("GET /v1/lessons/{id}/quiz", inTenant(httpx.Handler(s.quizForLearner)))
	mux.Handle("POST /v1/quizzes/{id}/attempts", inTenant(httpx.Handler(s.startAttempt)))
	mux.Handle("GET /v1/attempts/{id}", inTenant(httpx.Handler(s.myAttempt)))
	mux.Handle("PUT /v1/attempts/{id}/answers", inTenant(httpx.Handler(s.saveAnswer)))
	mux.Handle("POST /v1/attempts/{id}/submit", inTenant(httpx.Handler(s.submitAttempt)))
	mux.Handle("GET /v1/courses/{id}/gradebook", teaches(s.gradebook))
	mux.Handle("POST /v1/courses/{id}/grade-items", teaches(s.createGradeItem))
	mux.Handle("GET /v1/courses/{id}/grades", inTenant(httpx.Handler(s.myGrades)))
	mux.Handle("PUT /v1/grade-items/{id}/learners/{enrollmentId}", teaches(s.setGrade))
	mux.Handle("DELETE /v1/grade-items/{id}/learners/{enrollmentId}", teaches(s.clearGrade))
	mux.Handle("POST /v1/lessons/{id}/assignment", teaches(s.createAssignment))
	mux.Handle("GET /v1/lessons/{id}/assignment", inTenant(httpx.Handler(s.assignmentForLearner)))
	mux.Handle("PATCH /v1/assignments/{id}", teaches(s.updateAssignment))
	mux.Handle("PUT /v1/assignments/{id}/submission", inTenant(httpx.Handler(s.submitWork)))
	mux.Handle("GET /v1/submissions", teaches(s.submissionQueue))
	mux.Handle("GET /v1/submissions/{id}", teaches(s.submissionSheet))
	mux.Handle("POST /v1/submissions/{id}/grade", teaches(s.gradeWork))
	// Public: anyone holding a serial can check a certificate.
	mux.Handle("GET /verify/{serial}", httpx.Handler(s.verifyCertificate))

	mux.Handle("POST /v1/courses/{id}/certificates", inTenant(httpx.Handler(s.issueCertificate)))
	mux.Handle("GET /v1/courses/{id}/certificates", teaches(s.courseCertificates))
	mux.Handle("GET /v1/certificates", inTenant(httpx.Handler(s.listMyCertificates)))
	mux.Handle("POST /v1/certificates/{id}/revoke", teaches(s.revokeCertificate))

	mux.Handle("POST /v1/courses/{id}/sessions", teaches(s.createClassSession))
	mux.Handle("GET /v1/courses/{id}/sessions", inTenant(httpx.Handler(s.listClassSessions)))
	mux.Handle("GET /v1/sessions/{id}/roll", teaches(s.sessionRoll))
	mux.Handle("PUT /v1/sessions/{id}/roll", teaches(s.takeRoll))
	mux.Handle("PUT /v1/sessions/{id}/link", teaches(s.setSessionLink))
	mux.Handle("PUT /v1/sessions/{id}/recording", teaches(s.attachRecording))
	mux.Handle("GET /v1/sessions/{id}/join", inTenant(httpx.Handler(s.joinSession)))
	mux.Handle("GET /v1/courses/{id}/attendance", inTenant(httpx.Handler(s.myAttendance)))
	mux.Handle("POST /v1/guardians", inTenant(RequireRole("owner", "admin")(httpx.Handler(s.addGuardian))))
	mux.Handle("GET /v1/guardians", inTenant(RequireRole("owner", "admin")(httpx.Handler(s.listGuardians))))

	mux.Handle("GET /v1/notifications", inTenant(httpx.Handler(s.inbox)))
	mux.Handle("POST /v1/notifications/read", inTenant(httpx.Handler(s.markAllRead)))
	mux.Handle("POST /v1/notifications/{id}/read", inTenant(httpx.Handler(s.markNotificationRead)))
	mux.Handle("GET /v1/grading", teaches(s.markingQueue))
	mux.Handle("GET /v1/attempts/{id}/sheet", teaches(s.attemptSheet))
	mux.Handle("PUT /v1/attempts/{id}/questions/{questionId}/grade", teaches(s.markAnswer))
	mux.Handle("POST /v1/attempts/{id}/release", teaches(s.releaseAttempt))

	mux.Handle("GET /v1/payment/providers", inTenant(httpx.Handler(s.paymentProviders)))
	mux.Handle("POST /v1/courses/{id}/orders", inTenant(httpx.Handler(s.createOrder)))
	mux.Handle("GET /v1/orders", inTenant(httpx.Handler(s.listMyOrders)))
	mux.Handle("POST /v1/orders/{id}/proof", inTenant(httpx.Handler(s.submitProof)))
	mux.Handle("DELETE /v1/orders/{id}", inTenant(httpx.Handler(s.cancelOrder)))
	money := func(h httpx.Handler) http.Handler {
		return inTenant(RequireRole("owner", "admin")(h))
	}
	mux.Handle("GET /v1/coupons", money(s.listCoupons))
	mux.Handle("POST /v1/coupons", money(s.createCoupon))
	mux.Handle("PUT /v1/coupons/{id}/active", money(s.setCouponActive))
	mux.Handle("DELETE /v1/coupons/{id}", money(s.deleteCoupon))
	mux.Handle("POST /v1/lessons/{id}/quiz/draft", teaches(s.draftQuestions))
	mux.Handle("GET /v1/orders/paid", money(s.listPaidOrders))
	mux.Handle("POST /v1/orders/{id}/refund", money(s.refundOrder))
	mux.Handle("GET /v1/orders/review", inTenant(RequireRole("owner", "admin")(httpx.Handler(s.listReviewQueue))))
	mux.Handle("POST /v1/orders/{id}/review", inTenant(RequireRole("owner", "admin")(httpx.Handler(s.reviewOrder))))

	// The institution's own website: staff build it, anyone reads it.
	site := func(h httpx.Handler) http.Handler {
		return inTenant(RequireRole("owner", "admin")(h))
	}
	mux.Handle("GET /v1/site/pages", site(s.listSitePages))
	mux.Handle("POST /v1/site/pages", site(s.createSitePage))
	mux.Handle("GET /v1/site/pages/{id}", site(s.getSitePage))
	mux.Handle("PATCH /v1/site/pages/{id}", site(s.updateSitePage))
	mux.Handle("PUT /v1/site/pages/{id}/status", site(s.setSitePageStatus))
	mux.Handle("DELETE /v1/site/pages/{id}", site(s.deleteSitePage))
	mux.Handle("PUT /v1/site/theme", site(s.setSiteTheme))
	mux.Handle("GET /v1/site/domain", site(s.siteDomain))
	mux.Handle("PUT /v1/site/domain", site(s.setDomain))
	mux.Handle("POST /v1/site/domain/verify", site(s.verifyDomain))
	mux.Handle("DELETE /v1/site/domain", site(s.clearDomain))
	mux.Handle("GET /site/resolve", httpx.Handler(s.resolveHost))
	mux.Handle("GET /site/{tenant}", httpx.Handler(s.publicPage))
	mux.Handle("GET /site/{tenant}/{slug}", httpx.Handler(s.publicPage))

	return mux
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) error {
	if err := s.store.Health(r.Context()); err != nil {
		return httpx.Errorf(http.StatusServiceUnavailable, "database_unavailable", "Database is not reachable.")
	}
	return httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
