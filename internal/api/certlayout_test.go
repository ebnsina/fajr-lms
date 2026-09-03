package api_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/database"
)

// TestCertificateLayout covers a school laying out its own certificate, and
// what the public page then draws.
func TestCertificateLayout(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	t.Run("a school with no layout has an empty one, not an error", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/certificates/layout", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Fields        []map[string]any `json:"fields"`
			HasBackground bool             `json:"has_background"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Fields) != 0 || out.HasBackground {
			t.Fatalf("got %s", rec.Body)
		}
	})

	t.Run("a learner cannot lay out the school's certificate", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/certificates/layout", student.token, owner.slug,
			map[string]any{"fields": []map[string]any{}})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a field we cannot fill in is refused", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/certificates/layout", owner.token, owner.slug,
			map[string]any{"fields": []map[string]any{
				{"token": "salary", "x": 50, "y": 50, "size": 2},
			}})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a field off the page is refused", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/certificates/layout", owner.token, owner.slug,
			map[string]any{"fields": []map[string]any{
				{"token": "recipient", "x": 320, "y": 50, "size": 2},
			}})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a layout is kept and read back", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/certificates/layout", owner.token, owner.slug,
			map[string]any{"fields": []map[string]any{
				{"token": "text", "label": "Certificate of completion", "x": 50, "y": 20, "size": 1.2},
				{"token": "recipient", "x": 50, "y": 45, "size": 3, "bold": true, "color": "#047857"},
				{"token": "course", "x": 50, "y": 60, "size": 1.5},
				{"token": "serial", "x": 50, "y": 90, "size": 0.9, "align": "end"},
			}})
		if rec.Code != http.StatusOK {
			t.Fatalf("save: got %d: %s", rec.Code, rec.Body)
		}
		rec = do(t, h, "GET", "/v1/certificates/layout", owner.token, owner.slug, nil)
		var out struct {
			Fields []struct {
				Token string  `json:"token"`
				X     float64 `json:"x"`
				Bold  bool    `json:"bold"`
				Align string  `json:"align"`
			} `json:"fields"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Fields) != 4 || out.Fields[1].Token != "recipient" || !out.Fields[1].Bold {
			t.Fatalf("got %s", rec.Body)
		}
		if out.Fields[0].Align != "center" {
			t.Fatalf("a field with no alignment came back %q, want centered", out.Fields[0].Align)
		}
	})

	t.Run("the paper is stored and served on the public page", func(t *testing.T) {
		// A one-pixel PNG is enough to prove the round trip.
		// A real PNG signature, so the server's own sniffing accepts it.
		png := append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, []byte("test paper")...)
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, err := form.CreateFormFile("file", "paper.png")
		if err != nil {
			t.Fatalf("form: %v", err)
		}
		if _, err := part.Write(png); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := form.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}

		req := httptest.NewRequest("PUT", "/v1/certificates/layout/background", &body)
		req.Header.Set("content-type", form.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+owner.token)
		req.Header.Set("X-Fajr-Tenant", owner.slug)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("upload: got %d: %s", rec.Code, rec.Body)
		}

		serial := issueCertificate(t, h, ch, store, owner)
		got := do(t, h, "GET", "/verify/"+serial+"/background", "", "", nil)
		if got.Code != http.StatusOK {
			t.Fatalf("background: got %d: %s", got.Code, got.Body)
		}
		if !bytes.Equal(got.Body.Bytes(), png) {
			t.Fatal("the paper came back changed")
		}
		if kind := got.Header().Get("content-type"); kind != "image/png" {
			t.Fatalf("served as %q", kind)
		}

		// And the certificate page is now drawn with the school's own layout.
		page := httptest.NewRequest("GET", "/verify/"+serial, nil)
		page.Header.Set("Accept", "text/html")
		shown := httptest.NewRecorder()
		h.ServeHTTP(shown, page)
		if shown.Code != http.StatusOK {
			t.Fatalf("page: got %d", shown.Code)
		}
		if !bytes.Contains(shown.Body.Bytes(), []byte("class=\"laid\"")) {
			t.Fatal("the page did not use the school's layout")
		}
		if !bytes.Contains(shown.Body.Bytes(), []byte("Certificate of completion")) {
			t.Fatal("the school's own words are not on the page")
		}
	})

	t.Run("the school sees its own paper while drawing", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/certificates/layout/background", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if kind := rec.Header().Get("content-type"); kind != "image/png" {
			t.Fatalf("served as %q", kind)
		}
		if got := do(t, h, "GET", "/v1/certificates/layout/background", student.token, owner.slug,
			nil); got.Code != http.StatusForbidden {
			t.Fatalf("a learner got %d, want 403", got.Code)
		}
	})

	t.Run("the paper can be taken off again", func(t *testing.T) {
		if rec := do(t, h, "DELETE", "/v1/certificates/layout/background", owner.token, owner.slug,
			nil); rec.Code != http.StatusNoContent {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
	})
}

// issueCertificate walks a learner through a one-lesson course so there is a
// certificate to look at.
func issueCertificate(t *testing.T, h http.Handler, ch *captureChannel,
	store *database.Store, owner actor) string {
	t.Helper()
	learner := enrollIn(t, h, ch, store, owner.slug, "student")
	courseID, lessons := publishedCourse(t, h, owner, 1)
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", learner.token, owner.slug,
		nil); rec.Code != http.StatusCreated {
		t.Fatalf("enroll: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/v1/lessons/"+lessons[0]+"/progress", learner.token, owner.slug,
		map[string]any{"position_s": 10, "completed": true}); rec.Code != http.StatusOK {
		t.Fatalf("progress: got %d: %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "POST", "/v1/courses/"+courseID+"/certificates", learner.token, owner.slug, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("claim: got %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Serial string `json:"serial"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out.Serial
}
