package api_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testManifest = `<?xml version="1.0"?>
<manifest identifier="M1">
  <metadata><schemaversion>1.2</schemaversion></metadata>
  <organizations default="ORG1">
    <organization identifier="ORG1">
      <title>Laboratory Safety</title>
      <item identifier="I1" identifierref="R1"><title>Part one</title><masteryscore>60</masteryscore></item>
    </organization>
  </organizations>
  <resources>
    <resource identifier="R1" type="webcontent" href="start.html"/>
  </resources>
</manifest>`

func packageZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"imsmanifest.xml": testManifest,
		"start.html":      "<html><body>Lesson one</body></html>",
	} {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes()
}

// upload posts a package the way a browser would.
func upload(t *testing.T, h http.Handler, a actor, lessonID string, zipped []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", "safety.zip")
	if err != nil {
		t.Fatalf("form: %v", err)
	}
	if _, err := part.Write(zipped); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/lessons/"+lessonID+"/scorm", &body)
	req.Header.Set("content-type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("X-Fajr-Tenant", a.slug)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCoursePackages(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	courseID := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
		map[string]any{"title": "Laboratory work", "visibility": "public"}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/modules", owner.token, owner.slug,
		map[string]any{"title": "Unit"}))
	lessonID := createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", owner.token, owner.slug,
		map[string]any{"title": "Safety package", "kind": "text"}))
	do(t, h, "PATCH", "/v1/lessons/"+lessonID, owner.token, owner.slug, map[string]any{"status": "published"})
	do(t, h, "PUT", "/v1/courses/"+courseID+"/status", owner.token, owner.slug, map[string]any{"status": "published"})
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug, nil); rec.Code != http.StatusCreated {
		t.Fatalf("enroll: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("a learner cannot upload a package", func(t *testing.T) {
		if rec := upload(t, h, student, lessonID, packageZip(t)); rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("something that is not a package is refused", func(t *testing.T) {
		rec := upload(t, h, owner, lessonID, []byte("not a zip at all"))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("the manifest decides where the course starts", func(t *testing.T) {
		rec := upload(t, h, owner, lessonID, packageZip(t))
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var pkg struct {
			Title     string `json:"title"`
			EntryHref string `json:"entry_href"`
			FileCount int32  `json:"file_count"`
			Mastery   *int16 `json:"mastery"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &pkg); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if pkg.Title != "Laboratory Safety" || pkg.EntryHref != "start.html" || pkg.FileCount != 2 {
			t.Fatalf("got %+v", pkg)
		}
		if pkg.Mastery == nil || *pkg.Mastery != 60 {
			t.Fatalf("mastery is %v", pkg.Mastery)
		}
	})

	t.Run("the learner is served the package's own files", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/lessons/"+lessonID+"/scorm/files/start.html",
			student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("Lesson one")) {
			t.Fatalf("got %s", rec.Body)
		}
	})

	t.Run("a file that is not in the package is not found", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/lessons/"+lessonID+"/scorm/files/secrets.html",
			student.token, owner.slug, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("what the package reports is kept, and the lesson is finished", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/lessons/"+lessonID+"/scorm/state", student.token, owner.slug,
			map[string]any{
				"lesson_status": "passed", "score_raw": 82,
				"suspend_data": "page=4", "location": "page-4", "total_time_s": 640,
				"cmi": map[string]string{"cmi.core.lesson_location": "page-4"},
			})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}

		rec = do(t, h, "GET", "/v1/courses/"+courseID+"/progress", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("progress: got %d: %s", rec.Code, rec.Body)
		}
		var progress struct {
			LessonsDone int `json:"lessons_done"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &progress); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if progress.LessonsDone != 1 {
			t.Fatalf("the lesson was not counted as finished: %s", rec.Body)
		}
	})

	t.Run("a status SCORM does not define is refused", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/lessons/"+lessonID+"/scorm/state", student.token, owner.slug,
			map[string]any{"lesson_status": "brilliant"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a score outside 0 to 100 is refused", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/lessons/"+lessonID+"/scorm/state", student.token, owner.slug,
			map[string]any{"lesson_status": "passed", "score_raw": 5000})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("the teacher sees who has been through it", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/lessons/"+lessonID+"/scorm/progress", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Learners []struct {
				ScormState struct {
					LessonStatus string `json:"lesson_status"`
				} `json:"scorm_state"`
				FullName string `json:"full_name"`
			} `json:"learners"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Learners) != 1 || out.Learners[0].ScormState.LessonStatus != "passed" {
			t.Fatalf("got %s", rec.Body)
		}
	})
}
