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
			return renderCertificate(w, http.StatusNotFound,
				certificateView{Serial: serial, Assets: s.publicURL})
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
		Revoked:  row.RevokedAt.Valid, Dir: string(row.TenantDir), Assets: s.publicURL,
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
	// Where the product's own fonts are served from, so the printed page is
	// set in the same faces as the app that issued it.
	Assets string
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
  @font-face { font-family: 'Cabin'; src: url('{{.Assets}}/fonts/cabin-latin.woff2') format('woff2');
               font-weight: 400 700; font-display: swap; }
  @font-face { font-family: 'Geist Mono'; src: url('{{.Assets}}/fonts/geist-mono-latin.woff2') format('woff2');
               font-weight: 300 600; font-display: swap; }

  /* The product's own light tokens, so a printed certificate and the app that
     issued it are recognisably the same thing. */
  :root {
    color-scheme: light;
    --ink: #14171a; --ink-soft: #676f7b; --ink-faint: #9aa1ac;
    --brand: #047857; --brand-text: #046b4e; --brand-soft: #e7f6f0; --brand-line: #bfe6d6;
    --danger: #a03528; --danger-soft: #fdf0ee; --danger-line: #f0d3ce;
    --paper: #fffdf8; --ground: #f1efe8; --line: #e0e2e7;
    --radius-card: 1.5rem; --radius-control: 0.75rem;
    --sans: 'Cabin', 'Noto Sans Arabic', 'Noto Sans Bengali', system-ui, sans-serif;
    --mono: 'Geist Mono', ui-monospace, 'SF Mono', monospace;
  }

  * { box-sizing: border-box; }
  body { margin: 0; padding: 2.5rem 1rem; background: var(--ground); color: var(--ink);
         font: 16px/1.6 var(--sans); }
  :lang(bn), :lang(ar), :lang(ur) { line-height: 1.85; }

  .sheet { position: relative; max-inline-size: 52rem; margin-inline: auto;
           background: var(--paper); border: 1px solid var(--line);
           border-radius: var(--radius-card); padding: 3.5rem clamp(1.5rem, 6vw, 4.5rem);
           text-align: center; }
  /* The inner rule is the certificate's own border, drawn inside the sheet. */
  .sheet::before { content: ''; position: absolute; inset: 0.75rem;
                   border: 1px solid var(--brand-line); border-radius: 1rem; pointer-events: none; }
  .sheet > * { position: relative; }

  .crest { inline-size: 3rem; block-size: 3rem; margin-inline: auto; color: var(--brand-text); }
  .label { text-transform: uppercase; letter-spacing: 0.14em; font-size: 0.75rem;
           font-weight: 600; color: var(--ink-soft); margin: 0; }
  .issuer-name { font-size: 1.05rem; font-weight: 600; margin: 0.35rem 0 0; }
  .name { font-size: clamp(2rem, 6vw, 3rem); font-weight: 700; letter-spacing: -0.02em;
          line-height: 1.25; margin: 0.75rem 0 1rem; }
  .rule { inline-size: 8rem; block-size: 1px; margin: 0 auto 1.5rem; background: var(--brand-line);
          border: 0; }
  .course { font-size: clamp(1.15rem, 3vw, 1.5rem); font-weight: 600; margin: 0.4rem 0 0; }
  .grade { display: inline-block; margin-top: 1rem; padding: 0.25rem 0.8rem;
           border-radius: var(--radius-control); background: var(--brand-soft);
           border: 1px solid var(--brand-line); color: var(--brand-text);
           font-family: var(--mono); font-size: 0.85rem; }

  .meta { display: flex; flex-wrap: wrap; justify-content: center; gap: 1rem 3rem;
          margin-top: 2.5rem; padding-top: 1.75rem; border-top: 1px solid var(--line);
          text-align: center; }
  .meta div { min-inline-size: 8rem; }
  .meta dt { text-transform: uppercase; letter-spacing: 0.12em; font-size: 0.65rem;
             font-weight: 600; color: var(--ink-faint); margin: 0 0 0.25rem; }
  .meta dd { margin: 0; font-size: 0.95rem; }
  .serial { font-family: var(--mono); letter-spacing: 0.06em; }

  .badge { display: inline-flex; align-items: center; gap: 0.4rem; margin-top: 2rem;
           padding: 0.35rem 0.9rem; border-radius: 999px; font-size: 0.8rem; font-weight: 600;
           background: var(--brand-soft); border: 1px solid var(--brand-line); color: var(--brand-text); }
  .badge.bad { background: var(--danger-soft); border-color: var(--danger-line); color: var(--danger); }

  /* Revoked has to be unmistakable at a glance, printed or on screen. */
  .void .sheet::before { border-color: var(--danger-line); }
  .void .name, .void .course { color: var(--ink-soft); }
  .stamp { position: absolute; inset-block-start: 50%; inset-inline-start: 50%;
           transform: translate(-50%, -50%) rotate(-18deg); font-size: clamp(2.5rem, 10vw, 5rem);
           font-weight: 700; letter-spacing: 0.15em; color: var(--danger); opacity: 0.14;
           pointer-events: none; white-space: nowrap; }

  .checked { max-inline-size: 52rem; margin: 1.25rem auto 0; text-align: center;
             font-size: 0.8rem; color: var(--ink-faint); }

  @media print {
    body { background: #fff; padding: 0; print-color-adjust: exact; -webkit-print-color-adjust: exact; }
    .sheet { border: 0; border-radius: 0; max-inline-size: none; }
    .checked { display: none; }
  }
</style>
{{if .Found}}
<div{{if .Revoked}} class="void"{{end}}>
  <div class="sheet">
    {{if .Revoked}}<span class="stamp" aria-hidden="true">REVOKED</span>{{end}}
    <svg class="crest" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"
         stroke-linecap="round" stroke-linejoin="round" role="img" aria-label="Seal">
      <circle cx="12" cy="8" r="6"/><path d="M8.2 13.4 7 22l5-3 5 3-1.2-8.6"/>
    </svg>
    <p class="label" style="margin-top:1rem">{{if .Revoked}}Revoked certificate{{else}}Certificate of completion{{end}}</p>
    <p class="issuer-name" dir="auto">{{.Issuer}}</p>

    <p class="label" style="margin-top:2rem">This is to certify that</p>
    <p class="name" dir="auto">{{.Recipient}}</p>
    <hr class="rule">
    <p class="label">has completed the course</p>
    <p class="course" dir="auto">{{.Course}}</p>
    {{if .HasGrade}}<p class="grade">Final grade {{.Grade}}%</p>{{end}}

    <dl class="meta">
      <div><dt>Issued by</dt><dd dir="auto">{{.Issuer}}</dd></div>
      <div><dt>Issued on</dt><dd>{{.IssuedOn}}</dd></div>
      <div><dt>Serial</dt><dd class="serial">{{.Serial}}</dd></div>
    </dl>

    <p><span class="badge{{if .Revoked}} bad{{end}}">
      {{if .Revoked}}This certificate has been revoked{{else}}Verified as genuine{{end}}
    </span></p>
  </div>
</div>
<p class="checked">Checked against {{.Issuer}}'s records on Fajr LMS.</p>
{{else}}
<div class="void">
  <div class="sheet">
    <p class="label">Not found</p>
    <p class="course" style="margin-top:0.75rem">No certificate matches that number</p>
    <hr class="rule">
    <p class="serial">{{.Serial}}</p>
    <p><span class="badge bad">Nothing was issued under this serial</span></p>
  </div>
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
