package api

import (
	"crypto/rand"
	"html/template"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type issueRequest struct {
	UserID string `json:"user_id"`
}

// issueCertificate awards a certificate for a finished course. Staff may issue
// to anyone; a learner may claim their own once the course is complete.
func (s *Server) issueCertificate(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body issueRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	caller := Authenticated(r.Context()).UserID
	target := caller
	staff := staffRole(CurrentRole(r.Context()))
	if raw := strings.TrimSpace(body.UserID); raw != "" {
		if target, err = uuid.Parse(raw); err != nil {
			return invalid("user_id", "Name the learner to award.")
		}
	}
	if target != caller && !staff {
		return httpx.ErrForbidden
	}

	tenant := CurrentTenant(r.Context())
	var certificate database.Certificate

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		course, err := q.GetCourse(r.Context(), courseID)
		if err != nil {
			return err
		}
		enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
			CourseID: courseID, UserID: target,
		})
		if err != nil {
			return err
		}
		// A learner claiming their own must have finished; staff may decide.
		if !staff && enrollment.Status != database.EnrollmentStatusCompleted {
			return httpx.Errorf(http.StatusConflict, "course_incomplete",
				"Finish the course before claiming a certificate.")
		}

		if existing, err := q.CertificateForEnrollment(r.Context(), enrollment.ID); err == nil {
			certificate = existing
			return nil
		} else if !database.IsNotFound(err) {
			return err
		}

		recipient, err := q.GetUser(r.Context(), target)
		if err != nil {
			return err
		}
		grade, err := s.courseGrade(r, q, courseID, enrollment.ID)
		if err != nil {
			return err
		}

		certificate, err = q.IssueCertificate(r.Context(), database.IssueCertificateParams{
			TenantID: tenant.ID, CourseID: courseID, EnrollmentID: enrollment.ID, UserID: target,
			Serial: newSerial(), RecipientName: recipient.FullName, CourseTitle: course.Title,
			IssuerName: tenant.Name, GradePercent: grade,
			IssuedBy: uuid.NullUUID{UUID: caller, Valid: true},
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	s.notifyUser(r.Context(), tenant.ID, certificate.UserID, "certificate.issued",
		"Your certificate is ready",
		"Certificate "+certificate.Serial+" for "+certificate.CourseTitle+".",
		map[string]any{"serial": certificate.Serial, "verify_url": s.verifyURL(certificate.Serial)})

	return httpx.JSON(w, http.StatusCreated, s.withVerifyURL(certificate))
}

// courseGrade is the learner's weighted percentage, or nothing if untested.
func (s *Server) courseGrade(r *http.Request, q *database.Queries, courseID, enrollmentID uuid.UUID) (*int16, error) {
	_, reports, err := s.buildGradebook(r, q, courseID)
	if err != nil {
		return nil, err
	}
	for _, report := range reports {
		if report.EnrollmentID == enrollmentID && report.Graded > 0 {
			grade := int16(report.Percent)
			return &grade, nil
		}
	}
	return nil, nil
}

type certificateResponse struct {
	database.Certificate
	VerifyURL string `json:"verify_url"`
}

func (s *Server) withVerifyURL(c database.Certificate) certificateResponse {
	return certificateResponse{Certificate: c, VerifyURL: s.verifyURL(c.Serial)}
}

func (s *Server) verifyURL(serial string) string { return s.publicURL + "/verify/" + serial }

// courseCertificates is the staff view: who has been awarded one on this
// course, and whether it still stands.
func (s *Server) courseCertificates(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}

	var rows []database.ListCourseCertificatesRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListCourseCertificates(r.Context(), database.ListCourseCertificatesParams{
			CourseID: courseID, PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}

	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"certificate": row.Certificate, "full_name": row.FullName,
			"verify_url": s.verifyURL(row.Certificate.Serial),
		})
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"certificates": out})
}

func (s *Server) listMyCertificates(w http.ResponseWriter, r *http.Request) error {
	var rows []database.ListMyCertificatesRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListMyCertificates(r.Context(), Authenticated(r.Context()).UserID)
		return err
	})
	if err != nil {
		return err
	}

	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"certificate": row.Certificate, "course_slug": row.CourseSlug,
			"verify_url": s.verifyURL(row.Certificate.Serial),
		})
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"certificates": out})
}

type revokeRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) revokeCertificate(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body revokeRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if len(body.Reason) > 500 {
		return invalid("reason", "Keep the reason under 500 characters.")
	}

	var certificate database.Certificate
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		if _, err := q.GetCertificate(r.Context(), id); err != nil {
			return err
		}
		var err error
		certificate, err = q.RevokeCertificate(r.Context(), database.RevokeCertificateParams{
			ID: id, RevokedReason: strings.TrimSpace(body.Reason),
		})
		if database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "already_revoked", "This certificate is already revoked.")
		}
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, s.withVerifyURL(certificate))
}

// verifyCertificate is public: anyone holding a serial can check it. It answers
// JSON to a machine and a printable page to a browser, and the browser does the
// Arabic and Bengali shaping that a server-side PDF library would get wrong.
func (s *Server) verifyCertificate(w http.ResponseWriter, r *http.Request) error {
	serial := strings.ToUpper(strings.TrimSpace(r.PathValue("serial")))
	if len(serial) > 32 {
		return httpx.ErrNotFound
	}

	row, err := s.store.Unscoped().VerifyCertificate(r.Context(), serial)
	if database.IsNotFound(err) {
		if wantsHTML(r) {
			return renderCertificate(w, http.StatusNotFound, certificateView{Serial: serial})
		}
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	view := certificateView{
		Found: true, Serial: row.Serial, Recipient: row.RecipientName,
		Course: row.CourseTitle, Issuer: row.IssuerName,
		IssuedOn: row.IssuedAt.Time.Format("2 January 2006"),
		Revoked:  row.RevokedAt.Valid, Dir: string(row.TenantDir),
	}
	if row.GradePercent != nil {
		view.Grade, view.HasGrade = int(*row.GradePercent), true
	}

	if wantsHTML(r) {
		return renderCertificate(w, http.StatusOK, view)
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"serial": view.Serial, "recipient_name": view.Recipient, "course_title": view.Course,
		"issuer_name": view.Issuer, "issued_at": row.IssuedAt.Time, "revoked": view.Revoked,
		"grade_percent": row.GradePercent, "valid": !view.Revoked,
	})
}

type certificateView struct {
	Found     bool
	Serial    string
	Recipient string
	Course    string
	Issuer    string
	IssuedOn  string
	Grade     int
	HasGrade  bool
	Revoked   bool
	Dir       string
}

func wantsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") ||
		(accept == "" && r.Header.Get("Sec-Fetch-Mode") == "navigate")
}

func renderCertificate(w http.ResponseWriter, status int, view certificateView) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	return certificateTemplate.Execute(w, view)
}

// The page carries dir="auto" on every name, so an Arabic or Bengali name
// renders correctly without the server guessing its direction.
var certificateTemplate = template.Must(template.New("certificate").Parse(`<!doctype html>
<html lang="en"{{if eq .Dir "rtl"}} dir="rtl"{{end}}>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{if .Found}}Certificate {{.Serial}}{{else}}Certificate not found{{end}}</title>
<style>
  :root { color-scheme: light; }
  body { margin: 0; padding: 2rem 1rem; background: #f6f5f1; color: #1a1a1a;
         font: 16px/1.7 system-ui, "Segoe UI", "Noto Sans", "Noto Sans Arabic", "Noto Sans Bengali", sans-serif; }
  .sheet { max-width: 40rem; margin: 0 auto; background: #fff; border: 1px solid #ddd8cc;
           border-radius: 4px; padding: 3rem 2rem; text-align: center; }
  .name, .course { font-size: 1.6rem; font-weight: 600; margin: .5rem 0; line-height: 1.4; }
  .label { text-transform: uppercase; letter-spacing: .1em; font-size: .75rem; color: #6b6b6b; }
  .serial { font-family: ui-monospace, monospace; letter-spacing: .05em; }
  .void { background: #fdeaea; border-color: #e0b4b4; }
  .badge { display: inline-block; padding: .25rem .75rem; border-radius: 999px;
           font-size: .8rem; font-weight: 600; }
  .ok { background: #e7f4ec; color: #1b5e34; }
  .bad { background: #fbe3e3; color: #8c1c1c; }
  @media print { body { background: #fff; padding: 0; } .sheet { border: 0; } }
</style>
{{if .Found}}
<div class="sheet{{if .Revoked}} void{{end}}">
  <p class="label">{{if .Revoked}}Revoked certificate{{else}}Certificate of completion{{end}}</p>
  <p class="name" dir="auto">{{.Recipient}}</p>
  <p class="label">has completed</p>
  <p class="course" dir="auto">{{.Course}}</p>
  {{if .HasGrade}}<p>Final grade: {{.Grade}}%</p>{{end}}
  <p>Issued by <span dir="auto">{{.Issuer}}</span> on {{.IssuedOn}}</p>
  <p class="serial">{{.Serial}}</p>
  <p><span class="badge {{if .Revoked}}bad{{else}}ok{{end}}">
    {{if .Revoked}}This certificate has been revoked{{else}}Verified as genuine{{end}}
  </span></p>
</div>
{{else}}
<div class="sheet void">
  <p class="label">Not found</p>
  <p class="name">No certificate matches that number</p>
  <p class="serial">{{.Serial}}</p>
</div>
{{end}}
`))

// newSerial is quotable over the phone: no lookalike characters.
func newSerial() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "FJR-0000-0000-0000"
	}
	out := make([]byte, 0, 18)
	out = append(out, "FJR-"...)
	for i, b := range raw {
		if i > 0 && i%4 == 0 {
			out = append(out, '-')
		}
		out = append(out, alphabet[int(b)%len(alphabet)])
	}
	return string(out)
}
