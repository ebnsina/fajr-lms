package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/scorm"
)

// uploadPackage takes the zip a publisher gave the school, reads the manifest,
// and stores every file so the lesson can serve it back.
func (s *Server) uploadPackage(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, scorm.MaxBytes+(1<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return invalid("file", "Send the package as a file upload.")
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		return invalid("file", "Choose the package's zip file.")
	}
	defer file.Close()

	body, err := io.ReadAll(io.LimitReader(file, scorm.MaxBytes+1))
	if err != nil {
		return invalid("file", "That file could not be read all the way through.")
	}
	pkg, err := scorm.Read(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "bad_package",
			Message: strings.TrimPrefix(err.Error(), "scorm: "), Field: "file"}
	}
	if title := strings.TrimSpace(header.Filename); pkg.Title == "Course package" && title != "" {
		pkg.Title = strings.TrimSuffix(title, ".zip")
	}

	tenant := CurrentTenant(r.Context())
	var stored database.ScormPackage
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		lesson, err := q.GetLesson(r.Context(), lessonID)
		if err != nil {
			return err
		}
		courseID, err := q.LessonCourse(r.Context(), lesson.ID)
		if err != nil {
			return err
		}
		// Replacing a package replaces its files with it.
		if _, err := q.DeleteSCORMPackage(r.Context(), lessonID); err != nil {
			return err
		}

		// The package reports a score, so it needs somewhere for it to land.
		item, err := q.CreateGradeItem(r.Context(), database.CreateGradeItemParams{
			TenantID: tenant.ID, CourseID: courseID, Source: database.GradeSourceManual,
			Title: pkg.Title, Category: "package", PointsPossible: 100, Weight: 100,
		})
		if err != nil {
			return err
		}

		stored, err = q.CreateSCORMPackage(r.Context(), database.CreateSCORMPackageParams{
			TenantID: tenant.ID, LessonID: lessonID, Title: pkg.Title, EntryHref: pkg.Entry,
			Version: pkg.Version, Mastery: pkg.Mastery, FileCount: int32(len(pkg.Files)),
			Bytes: pkg.Bytes, GradeItemID: uuid.NullUUID{UUID: item.ID, Valid: true},
		})
		if err != nil {
			return err
		}
		for _, one := range pkg.Files {
			if err := q.AddSCORMFile(r.Context(), database.AddSCORMFileParams{
				PackageID: stored.ID, Path: one.Path, ContentType: one.ContentType, Body: one.Body,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, stored)
}

func (s *Server) lessonPackage(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var pkg database.ScormPackage
	var state database.ScormState
	userID := Authenticated(r.Context()).UserID

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if pkg, err = q.SCORMPackageForLesson(r.Context(), lessonID); err != nil {
			return err
		}
		state, err = q.SCORMState(r.Context(), database.SCORMStateParams{
			PackageID: pkg.ID, UserID: userID,
		})
		if database.IsNotFound(err) {
			state = database.ScormState{LessonStatus: "not attempted"}
			return nil
		}
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"package": pkg,
		"state": map[string]any{
			"lesson_status": state.LessonStatus, "suspend_data": state.SuspendData,
			"location": state.Location, "total_time_s": state.TotalTimeS,
			"score_raw": state.ScoreRaw,
		},
	})
}

// packageFile serves one file out of the package to the frame playing it.
func (s *Server) packageFile(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	wanted := strings.TrimPrefix(r.PathValue("path"), "/")
	if wanted == "" {
		return httpx.ErrNotFound
	}

	var row database.SCORMFileRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		pkg, err := q.SCORMPackageForLesson(r.Context(), lessonID)
		if err != nil {
			return err
		}
		row, err = q.SCORMFile(r.Context(), database.SCORMFileParams{PackageID: pkg.ID, Path: wanted})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	w.Header().Set("content-type", row.ContentType)
	w.Header().Set("content-length", strconv.Itoa(len(row.Body)))
	// A package never changes once uploaded; replacing it makes a new one.
	w.Header().Set("cache-control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(row.Body)
	return err
}

type scormStateRequest struct {
	CMI          map[string]string `json:"cmi"`
	LessonStatus string            `json:"lesson_status"`
	ScoreRaw     *float64          `json:"score_raw"`
	SuspendData  string            `json:"suspend_data"`
	Location     string            `json:"location"`
	TotalTimeS   int32             `json:"total_time_s"`
}

// saveScormState takes what the package reports. A package says whatever it
// likes, so everything is bounded here and a pass is written to the gradebook
// as an ordinary mark a teacher can change.
func (s *Server) saveScormState(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body scormStateRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if len(body.SuspendData) > 64000 {
		return invalid("suspend_data", "That is more than the package is allowed to keep.")
	}
	if len(body.Location) > 1000 {
		return invalid("location", "That bookmark is too long.")
	}
	if body.TotalTimeS < 0 || body.TotalTimeS > 1_000_000 {
		return invalid("total_time_s", "That is not a length of time.")
	}
	status := strings.ToLower(strings.TrimSpace(body.LessonStatus))
	if !knownStatus(status) {
		return invalid("lesson_status", "That is not a status SCORM defines.")
	}
	if body.ScoreRaw != nil && (*body.ScoreRaw < 0 || *body.ScoreRaw > 100) {
		return invalid("score_raw", "A score is between 0 and 100.")
	}

	cmi, err := json.Marshal(trimmedCMI(body.CMI))
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	userID := Authenticated(r.Context()).UserID
	var state database.ScormState

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		pkg, err := q.SCORMPackageForLesson(r.Context(), lessonID)
		if err != nil {
			return err
		}
		courseID, err := q.LessonCourse(r.Context(), lessonID)
		if err != nil {
			return err
		}
		enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
			CourseID: courseID, UserID: userID,
		})
		if err != nil {
			return err
		}
		if enrollment.Status == database.EnrollmentStatusCancelled {
			return httpx.ErrForbidden
		}

		state, err = q.SaveSCORMState(r.Context(), database.SaveSCORMStateParams{
			PackageID: pkg.ID, UserID: userID, TenantID: tenant.ID, Cmi: cmi,
			LessonStatus: status, ScoreRaw: numeric(body.ScoreRaw),
			SuspendData: body.SuspendData, Location: body.Location, TotalTimeS: body.TotalTimeS,
		})
		if err != nil {
			return err
		}

		done := status == "completed" || status == "passed"
		if _, err := q.RecordProgress(r.Context(), database.RecordProgressParams{
			TenantID: tenant.ID, EnrollmentID: enrollment.ID, LessonID: lessonID,
			Completed: done, PositionS: 0,
		}); err != nil {
			return err
		}

		if body.ScoreRaw == nil || !pkg.GradeItemID.Valid {
			return nil
		}
		_, err = q.SetGradeOverride(r.Context(), database.SetGradeOverrideParams{
			TenantID: tenant.ID, GradeItemID: pkg.GradeItemID.UUID, EnrollmentID: enrollment.ID,
			Points: int32(*body.ScoreRaw), Note: "reported by the package",
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"lesson_status": state.LessonStatus, "total_time_s": state.TotalTimeS,
	})
}

// listPackageProgress is the teacher's view of who has been through it.
func (s *Server) listPackageProgress(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows []database.ListSCORMStatesRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		pkg, err := q.SCORMPackageForLesson(r.Context(), lessonID)
		if err != nil {
			return err
		}
		rows, err = q.ListSCORMStates(r.Context(), pkg.ID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"learners": rows})
}

func (s *Server) deletePackage(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.DeleteSCORMPackage(r.Context(), lessonID)
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func knownStatus(status string) bool {
	switch status {
	case "passed", "failed", "completed", "incomplete", "browsed", "not attempted":
		return true
	}
	return false
}

// trimmedCMI keeps the package's own record small enough to store.
func trimmedCMI(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	kept := 0
	for name, value := range in {
		if kept >= 200 || len(name) > 200 {
			break
		}
		if len(value) > 4000 {
			value = value[:4000]
		}
		out[name] = value
		kept++
	}
	return out
}

func numeric(score *float64) pgtype.Numeric {
	if score == nil {
		return pgtype.Numeric{}
	}
	var out pgtype.Numeric
	if err := out.Scan(fmt.Sprintf("%.2f", *score)); err != nil {
		return pgtype.Numeric{}
	}
	return out
}
