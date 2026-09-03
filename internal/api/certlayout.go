package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// A school lays out its own certificate: which fields, where on the page, how
// large. Positions are percentages of the paper, so one layout prints the same
// at any size and on any screen.

const maxBackgroundBytes = 3 << 20

type certField struct {
	Token string  `json:"token"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Size  float64 `json:"size"`
	Align string  `json:"align"`
	Bold  bool    `json:"bold"`
	Color string  `json:"color"`
	Label string  `json:"label"`
}

// The tokens a layout may place. Anything else is a field we cannot fill in.
var certTokens = map[string]bool{
	"recipient": true, "course": true, "issuer": true,
	"date": true, "serial": true, "grade": true, "text": true,
}

func validateFields(raw []certField) ([]certField, error) {
	if len(raw) > 24 {
		return nil, invalid("fields", "That is more than a certificate can hold.")
	}
	out := make([]certField, 0, len(raw))
	for _, field := range raw {
		token := strings.ToLower(strings.TrimSpace(field.Token))
		if !certTokens[token] {
			return nil, invalid("fields", "One of those fields is not something we can fill in.")
		}
		if field.X < 0 || field.X > 100 || field.Y < 0 || field.Y > 100 {
			return nil, invalid("fields", "A field sits somewhere on the page, between 0 and 100.")
		}
		if field.Size < 0.5 || field.Size > 12 {
			return nil, invalid("fields", "A field's size is between 0.5 and 12.")
		}
		align := strings.ToLower(strings.TrimSpace(field.Align))
		if align != "start" && align != "center" && align != "end" {
			align = "center"
		}
		color := strings.TrimSpace(field.Color)
		if color != "" && !isHexColor(color) {
			return nil, invalid("fields", "A colour is written as #112233.")
		}
		label := strings.TrimSpace(field.Label)
		if len(label) > 120 {
			return nil, invalid("fields", "Keep a line of text under 120 characters.")
		}
		if token == "text" && label == "" {
			return nil, invalid("fields", "A line of text needs something to say.")
		}
		out = append(out, certField{
			Token: token, X: field.X, Y: field.Y, Size: field.Size,
			Align: align, Bold: field.Bold, Color: color, Label: label,
		})
	}
	return out, nil
}

func isHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, c := range value[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

func (s *Server) certificateLayout(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	var row database.GetCertificateLayoutRow
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		row, err = q.GetCertificateLayout(r.Context(), tenant.ID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.JSON(w, http.StatusOK, map[string]any{
			"fields": []certField{}, "has_background": false,
		})
	}
	if err != nil {
		return err
	}
	var fields []certField
	if err := json.Unmarshal(row.Fields, &fields); err != nil {
		fields = []certField{}
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"fields": fields, "has_background": row.HasBackground,
	})
}

type layoutRequest struct {
	Fields []certField `json:"fields"`
}

func (s *Server) saveCertificateLayout(w http.ResponseWriter, r *http.Request) error {
	var body layoutRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	fields, err := validateFields(body.Fields)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		_, err := q.SaveCertificateFields(r.Context(), database.SaveCertificateFieldsParams{
			TenantID: tenant.ID, Fields: encoded,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"fields": fields})
}

// setBackground takes the paper a certificate is printed on. It is stored
// beside the layout because it is one small image per school, and it is served
// publicly: it is on every certificate the school ever issues.
func (s *Server) setBackground(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBackgroundBytes+(1<<20))
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		return invalid("file", "Send the image as a file upload.")
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		return invalid("file", "Choose an image.")
	}
	defer file.Close()

	body, err := io.ReadAll(io.LimitReader(file, maxBackgroundBytes+1))
	if err != nil {
		return invalid("file", "That file could not be read.")
	}
	if len(body) > maxBackgroundBytes {
		return invalid("file", "Keep the background under 3 MB.")
	}

	// What it is, not what it says it is: this file is served back to whoever
	// opens a certificate.
	kind := imageKind(body, header.Header.Get("Content-Type"))
	if kind == "" {
		return invalid("file", "A background is a PNG, JPEG, WEBP or SVG.")
	}

	tenant := CurrentTenant(r.Context())
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		return q.SaveCertificateBackground(r.Context(), database.SaveCertificateBackgroundParams{
			TenantID: tenant.ID, Background: body, BackgroundType: kind,
		})
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"has_background": true})
}

// imageKind reads the first bytes rather than trusting the browser, and takes
// the declared type only for SVG, which is text and has no signature.
func imageKind(body []byte, declared string) string {
	switch http.DetectContentType(body) {
	case "image/png":
		return "image/png"
	case "image/jpeg":
		return "image/jpeg"
	case "image/webp":
		return "image/webp"
	}
	if strings.HasPrefix(declared, "image/svg+xml") && bytes.Contains(body[:min(len(body), 1024)], []byte("<svg")) {
		return "image/svg+xml"
	}
	return ""
}

// sealImage keeps an uploaded background from being anything but a picture:
// an SVG is a document that can carry script, and this one is served from our
// own origin.
func sealImage(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'")
	w.Header().Set("Content-Disposition", "inline")
}

func (s *Server) clearBackground(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		return q.ClearCertificateBackground(r.Context(), tenant.ID)
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// certificateBackground is public, like the certificate it is printed on.
func (s *Server) certificateBackground(w http.ResponseWriter, r *http.Request) error {
	serial := strings.ToUpper(strings.TrimSpace(r.PathValue("serial")))
	if len(serial) > 32 {
		return httpx.ErrNotFound
	}

	q := s.store.Unscoped()
	tenantID, err := q.CertificateTenant(r.Context(), serial)
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !tenantID.Valid {
		return httpx.ErrNotFound
	}
	row, err := q.CertificateBackground(r.Context(), tenantID.UUID)
	if database.IsNotFound(err) || len(row.Background) == 0 {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	w.Header().Set("content-type", row.BackgroundType)
	w.Header().Set("cache-control", "public, max-age=3600")
	sealImage(w)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(row.Background)
	return err
}

// layoutFor reads the school's layout for a certificate being verified, and
// says nothing if there is none: the shipped design is then used.
func (s *Server) layoutFor(r *http.Request, serial string) ([]certField, bool) {
	q := s.store.Unscoped()
	tenantID, err := q.CertificateTenant(r.Context(), serial)
	if err != nil {
		return nil, false
	}
	if !tenantID.Valid {
		return nil, false
	}
	row, err := q.PublicCertificateLayout(r.Context(), tenantID.UUID)
	if err != nil {
		return nil, false
	}
	var fields []certField
	if err := json.Unmarshal(row.Fields, &fields); err != nil || len(fields) == 0 {
		return nil, false
	}
	return fields, row.HasBackground
}

// ownBackground is the school looking at the paper it chose, while it draws
// the layout: the public route needs a serial, and a draft has none yet.
func (s *Server) ownBackground(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	var row database.OwnCertificateBackgroundRow
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		row, err = q.OwnCertificateBackground(r.Context(), tenant.ID)
		return err
	})
	if database.IsNotFound(err) || len(row.Background) == 0 {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	w.Header().Set("content-type", row.BackgroundType)
	w.Header().Set("cache-control", "private, no-store")
	sealImage(w)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(row.Background)
	return err
}
