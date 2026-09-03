package api

import (
	"net/http"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/media"
	"github.com/ebnsina/fajr-lms/internal/notify"
	"github.com/ebnsina/fajr-lms/internal/payment"
	"github.com/ebnsina/fajr-lms/internal/sso"
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
	sso       *sso.Client
}

// UseNotifier wires the notification service after construction, since the
// service records through the server itself.
func (s *Server) UseNotifier(n *notify.Service) { s.notifier = n }

func NewServer(store *database.Store, ident *identity.Service, registry *media.Registry,
	payments *payment.Registry, publicURL string) *Server {
	return &Server{store: store, identity: ident, media: registry, payments: payments,
		publicURL: publicURL, dns: netLookup{}, sso: &sso.Client{}}
}

// UseSSO replaces the OpenID client, which a test needs to trust its own
// provider.
func (s *Server) UseSSO(client *sso.Client) { s.sso = client }

// Routes returns the full API surface, health probes included.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /healthz", httpx.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	mux.Handle("GET /readyz", httpx.Handler(s.ready))

	mux.Handle("POST /v1/auth/otp", httpx.Handler(s.requestOTP))
	mux.Handle("POST /v1/auth/otp/verify", httpx.Handler(s.verifyOTP))
	mux.Handle("GET /v1/auth/sso/{slug}", httpx.Handler(s.ssoOffered))
	mux.Handle("POST /v1/auth/sso/{slug}/start", httpx.Handler(s.startSSO))
	mux.Handle("POST /v1/auth/sso/finish", httpx.Handler(s.finishSSO))

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
	mux.Handle("GET /v1/lessons/{id}", teaches(s.oneLesson))
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
	mux.Handle("GET /v1/plans", inTenant(httpx.Handler(s.myPlans)))
	mux.Handle("POST /v1/orders/{id}/proof", inTenant(httpx.Handler(s.submitProof)))
	mux.Handle("DELETE /v1/orders/{id}", inTenant(httpx.Handler(s.cancelOrder)))
	money := func(h httpx.Handler) http.Handler {
		return inTenant(RequireRole("owner", "admin")(h))
	}
	mux.Handle("GET /v1/coupons", money(s.listCoupons))
	mux.Handle("POST /v1/coupons", money(s.createCoupon))
	mux.Handle("PUT /v1/coupons/{id}/active", money(s.setCouponActive))
	mux.Handle("DELETE /v1/coupons/{id}", money(s.deleteCoupon))
	mux.Handle("POST /v1/lessons/{id}/scorm", teaches(s.uploadPackage))
	mux.Handle("DELETE /v1/lessons/{id}/scorm", teaches(s.deletePackage))
	mux.Handle("GET /v1/lessons/{id}/scorm/progress", teaches(s.listPackageProgress))
	mux.Handle("GET /v1/lessons/{id}/scorm", inTenant(httpx.Handler(s.lessonPackage)))
	mux.Handle("PUT /v1/lessons/{id}/scorm/state", inTenant(httpx.Handler(s.saveScormState)))
	mux.Handle("GET /v1/lessons/{id}/scorm/files/{path...}", inTenant(httpx.Handler(s.packageFile)))
	mux.Handle("GET /v1/orders/paid", money(s.listPaidOrders))
	mux.Handle("GET /v1/plans/all", money(s.listPlans))
	mux.Handle("POST /v1/orders/{id}/refund", money(s.refundOrder))
	// The academic spine. Reading it is open to any member, because the whole
	// school is arranged by it; changing it belongs to the office.
	mux.Handle("GET /v1/academics/years", inTenant(httpx.Handler(s.listYears)))
	mux.Handle("POST /v1/academics/years", runs(s.createYear))
	mux.Handle("PUT /v1/academics/years/{id}/current", runs(s.makeYearCurrent))
	mux.Handle("DELETE /v1/academics/years/{id}", runs(s.deleteYear))
	mux.Handle("POST /v1/academics/years/{id}/terms", runs(s.createTerm))
	mux.Handle("PUT /v1/academics/terms/{id}/current", runs(s.makeTermCurrent))
	mux.Handle("DELETE /v1/academics/terms/{id}", runs(s.deleteTerm))

	mux.Handle("GET /v1/academics/classes", inTenant(httpx.Handler(s.listClasses)))
	mux.Handle("POST /v1/academics/classes", runs(s.createClass))
	mux.Handle("PATCH /v1/academics/classes/{id}", runs(s.updateClass))
	mux.Handle("DELETE /v1/academics/classes/{id}", runs(s.deleteClass))
	mux.Handle("POST /v1/academics/classes/{id}/sections", runs(s.createSection))
	mux.Handle("PATCH /v1/academics/sections/{id}", runs(s.updateSection))
	mux.Handle("DELETE /v1/academics/sections/{id}", runs(s.deleteSection))
	mux.Handle("GET /v1/academics/sections/{id}/roll", teaches(s.sectionRoll))
	mux.Handle("POST /v1/academics/sections/{id}/roll", runs(s.placeStudent))
	mux.Handle("DELETE /v1/academics/placements/{id}", runs(s.removePlacement))

	// Points and badges, if a school wants them.
	mux.Handle("GET /v1/points/me", inTenant(httpx.Handler(s.myStanding)))
	mux.Handle("GET /v1/points/board", inTenant(httpx.Handler(s.leaderboard)))
	mux.Handle("GET /v1/badges", inTenant(httpx.Handler(s.listBadges)))
	mux.Handle("POST /v1/badges", runs(s.createBadge))
	mux.Handle("DELETE /v1/badges/{id}", runs(s.deleteBadge))
	mux.Handle("PUT /v1/points/setting", runs(s.setPointsOn))

	// Topics file a course; collections put several together — a path to work
	// through, a bundle to buy.
	mux.Handle("GET /v1/topics", inTenant(httpx.Handler(s.listTopics)))
	mux.Handle("POST /v1/topics", teaches(s.createTopic))
	mux.Handle("DELETE /v1/topics/{id}", teaches(s.deleteTopic))
	mux.Handle("GET /v1/courses/{id}/topics", inTenant(httpx.Handler(s.courseTopics)))
	mux.Handle("PUT /v1/courses/{id}/topics", teaches(s.setCourseTopics))

	mux.Handle("GET /v1/collections", inTenant(httpx.Handler(s.listCollections)))
	mux.Handle("POST /v1/collections", teaches(s.createCollection))
	mux.Handle("GET /v1/collections/{slug}", inTenant(httpx.Handler(s.readCollection)))
	mux.Handle("PATCH /v1/collections/{id}", teaches(s.updateCollection))
	mux.Handle("DELETE /v1/collections/{id}", teaches(s.deleteCollection))
	mux.Handle("POST /v1/collections/{id}/courses", teaches(s.addToCollection))
	mux.Handle("DELETE /v1/collections/{id}/courses/{courseId}", teaches(s.removeFromCollection))

	// The course forum: anybody on the course may ask, staff moderate.
	mux.Handle("GET /v1/courses/{id}/threads", inTenant(httpx.Handler(s.listThreads)))
	mux.Handle("POST /v1/courses/{id}/threads", inTenant(httpx.Handler(s.startThread)))
	mux.Handle("GET /v1/threads/{id}", inTenant(httpx.Handler(s.readThread)))
	mux.Handle("POST /v1/threads/{id}/posts", inTenant(httpx.Handler(s.replyToThread)))
	mux.Handle("PATCH /v1/posts/{id}", inTenant(httpx.Handler(s.editPost)))
	mux.Handle("DELETE /v1/posts/{id}", inTenant(httpx.Handler(s.removePost)))
	mux.Handle("PUT /v1/threads/{id}/flags", teaches(s.setThreadFlags))
	mux.Handle("DELETE /v1/threads/{id}", teaches(s.deleteThread))

	mux.Handle("POST /v1/notices", teaches(s.sendNotice))
	mux.Handle("GET /v1/children", inTenant(httpx.Handler(s.myChildren)))

	// Hifz: a teacher writes the record, and a learner or their guardian reads
	// their own.
	mux.Handle("POST /v1/hifz", teaches(s.recordHifz))
	mux.Handle("GET /v1/hifz", teaches(s.hifzOnDate))
	mux.Handle("DELETE /v1/hifz/{id}", teaches(s.deleteHifz))
	mux.Handle("GET /v1/hifz/students/{id}", inTenant(httpx.Handler(s.studentHifz)))

	mux.Handle("POST /v1/academics/subjects", runs(s.createSubject))
	mux.Handle("DELETE /v1/academics/subjects/{id}", runs(s.deleteSubject))

	mux.Handle("GET /v1/sso", runs(s.identityProvider))
	mux.Handle("PUT /v1/sso", runs(s.setIdentityProvider))
	mux.Handle("DELETE /v1/sso", runs(s.clearIdentityProvider))
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
