package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type certificateBody struct {
	ID        string `json:"id"`
	Serial    string `json:"serial"`
	Recipient string `json:"recipient_name"`
	Course    string `json:"course_title"`
	Issuer    string `json:"issuer_name"`
	VerifyURL string `json:"verify_url"`
	Grade     *int16 `json:"grade_percent"`
}

func getPublic(t *testing.T, h http.Handler, path, accept string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCertificates(t *testing.T) {
	h, _, ch, store := notifyHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	courseID, lessons := publishedCourse(t, h, owner, 1)
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
		t.Fatalf("enroll: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("a learner cannot claim before finishing", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/certificates", student.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a learner cannot award somebody else", func(t *testing.T) {
		other := enrollIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/certificates", student.token, owner.slug,
			map[string]any{"user_id": other.userID.String()})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	// Finish the course, then claim.
	if rec := do(t, h, "PUT", "/v1/lessons/"+lessons[0]+"/progress", student.token, owner.slug,
		map[string]any{"position_s": 10, "completed": true}); rec.Code != http.StatusOK {
		t.Fatalf("progress: got %d: %s", rec.Code, rec.Body)
	}

	var cert certificateBody
	t.Run("finishing the course allows a claim", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/certificates", student.token, owner.slug, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &cert); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !strings.HasPrefix(cert.Serial, "FJR-") || len(cert.Serial) != 18 {
			t.Fatalf("serial = %q", cert.Serial)
		}
		if cert.Issuer == "" || cert.Course == "" || cert.Recipient == "" {
			t.Fatalf("got %+v", cert)
		}
		if cert.VerifyURL != "https://fajr.test/verify/"+cert.Serial {
			t.Errorf("verify_url = %q", cert.VerifyURL)
		}
	})

	t.Run("claiming twice returns the same certificate", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/certificates", student.token, owner.slug, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var again certificateBody
		if err := json.Unmarshal(rec.Body.Bytes(), &again); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if again.Serial != cert.Serial {
			t.Errorf("a second claim issued %s, want the existing %s", again.Serial, cert.Serial)
		}
	})

	t.Run("the learner is told, and the certificate is listed", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/notifications", student.token, owner.slug, nil)
		var inbox struct {
			Notifications []struct {
				Kind string `json:"kind"`
			} `json:"notifications"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &inbox); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(inbox.Notifications) == 0 || inbox.Notifications[0].Kind != "certificate.issued" {
			t.Errorf("got %+v", inbox.Notifications)
		}

		rec = do(t, h, "GET", "/v1/certificates", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list: got %d: %s", rec.Code, rec.Body)
		}
		var listed struct {
			Certificates []struct {
				VerifyURL string `json:"verify_url"`
			} `json:"certificates"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(listed.Certificates) != 1 || listed.Certificates[0].VerifyURL == "" {
			t.Errorf("got %+v", listed.Certificates)
		}
	})

	t.Run("anyone can verify it without signing in", func(t *testing.T) {
		rec := getPublic(t, h, "/verify/"+cert.Serial, "application/json")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Serial  string `json:"serial"`
			Valid   bool   `json:"valid"`
			Revoked bool   `json:"revoked"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Serial != cert.Serial || !got.Valid || got.Revoked {
			t.Fatalf("got %+v", got)
		}

		// A lowercase serial typed off a printout still resolves.
		if rec := getPublic(t, h, "/verify/"+strings.ToLower(cert.Serial), "application/json"); rec.Code != http.StatusOK {
			t.Errorf("lowercase serial: got %d", rec.Code)
		}
	})

	t.Run("a browser gets a printable page with the name direction preserved", func(t *testing.T) {
		rec := getPublic(t, h, "/verify/"+cert.Serial, "text/html")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("content type = %q", ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `dir="auto"`) {
			t.Error("names must carry dir=auto so Arabic and Bengali render correctly")
		}
		if !strings.Contains(body, cert.Serial) || !strings.Contains(body, "Verified as genuine") {
			t.Errorf("page did not confirm the certificate: %s", body)
		}
	})

	t.Run("an unknown serial is reported, not invented", func(t *testing.T) {
		if rec := getPublic(t, h, "/verify/FJR-ZZZZ-ZZZZ-ZZZZ", "application/json"); rec.Code != http.StatusNotFound {
			t.Errorf("json: got %d, want 404", rec.Code)
		}
		rec := getPublic(t, h, "/verify/FJR-ZZZZ-ZZZZ-ZZZZ", "text/html")
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "No certificate matches") {
			t.Errorf("html: got %d: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a revoked certificate says so publicly", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/certificates/"+cert.ID+"/revoke", student.token, owner.slug,
			map[string]any{"reason": "mine"}); rec.Code != http.StatusForbidden {
			t.Errorf("a learner revoking: got %d, want 403", rec.Code)
		}
		if rec := do(t, h, "POST", "/v1/certificates/"+cert.ID+"/revoke", owner.token, owner.slug,
			map[string]any{"reason": "issued in error"}); rec.Code != http.StatusOK {
			t.Fatalf("revoke: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "POST", "/v1/certificates/"+cert.ID+"/revoke", owner.token, owner.slug, nil); rec.Code != http.StatusConflict {
			t.Errorf("revoking twice: got %d, want 409", rec.Code)
		}

		rec := getPublic(t, h, "/verify/"+cert.Serial, "application/json")
		var got struct {
			Valid   bool `json:"valid"`
			Revoked bool `json:"revoked"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Valid || !got.Revoked {
			t.Fatalf("a revoked certificate still verifies: %+v", got)
		}
		page := getPublic(t, h, "/verify/"+cert.Serial, "text/html").Body.String()
		if !strings.Contains(page, "has been revoked") {
			t.Errorf("the page does not say it is revoked: %s", page)
		}
	})

	t.Run("a name is kept as it was on the day", func(t *testing.T) {
		// The snapshot is what the certificate shows, not a live join.
		rec := getPublic(t, h, "/verify/"+cert.Serial, "application/json")
		var got struct {
			Recipient string `json:"recipient_name"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Recipient != cert.Recipient {
			t.Errorf("recipient = %q, want the name recorded at issue %q", got.Recipient, cert.Recipient)
		}
	})
}
