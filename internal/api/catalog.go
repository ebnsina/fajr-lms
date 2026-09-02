package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type createCourseRequest struct {
	Title      string `json:"title"`
	Slug       string `json:"slug"`
	Summary    string `json:"summary"`
	Dir        string `json:"dir"`
	Visibility string `json:"visibility"`
	PriceMinor int64  `json:"price_minor"`
	Currency   string `json:"currency"`
}

func (s *Server) createCourse(w http.ResponseWriter, r *http.Request) error {
	var body createCourseRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	visibility, err := parseVisibility(body.Visibility)
	if err != nil {
		return err
	}
	if body.PriceMinor < 0 {
		return invalid("price_minor", "Price cannot be negative.")
	}

	tenant := CurrentTenant(r.Context())
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		slug = Slugify(title)
	}
	if !slugRe.MatchString(slug) {
		return invalid("slug", "Use lowercase letters, numbers and hyphens.")
	}
	currency := strings.ToUpper(strings.TrimSpace(body.Currency))
	if currency == "" {
		currency = tenant.Currency
	}

	var course database.Course
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		course, err = q.CreateCourse(r.Context(), database.CreateCourseParams{
			TenantID: tenant.ID, Slug: slug, Title: title,
			Summary: strings.TrimSpace(body.Summary), Dir: dir, Visibility: visibility,
			PriceMinor: body.PriceMinor, Currency: currency,
			CreatedBy: uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
		})
		return err
	})
	if isUniqueViolation(err) {
		return &httpx.Error{Status: http.StatusConflict, Code: "slug_taken",
			Message: "A course with this address already exists.", Field: "slug"}
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, course)
}

func (s *Server) listCourses(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var status *database.PublishStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		parsed, err := parseStatus(raw)
		if err != nil {
			return err
		}
		status = &parsed
	}

	var (
		courses []database.Course
		total   int64
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if courses, err = q.ListCourses(r.Context(), database.ListCoursesParams{
			StatusFilter: status, PageLimit: limit, PageOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountCourses(r.Context(), status)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"courses": courses, "total": total})
}

// outline is a course with its modules and their lessons, in display order.
type outline struct {
	Course  database.Course `json:"course"`
	Modules []outlineModule `json:"modules"`
}

type outlineModule struct {
	database.Module
	Lessons []database.Lesson `json:"lessons"`
}

func (s *Server) courseOutline(w http.ResponseWriter, r *http.Request) error {
	slug := r.PathValue("slug")
	if !slugRe.MatchString(slug) {
		return httpx.ErrNotFound
	}

	var out outline
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		course, err := q.GetCourseBySlug(r.Context(), slug)
		if err != nil {
			return err
		}
		modules, err := q.ListModules(r.Context(), course.ID)
		if err != nil {
			return err
		}
		lessons, err := q.ListLessonsForCourse(r.Context(), course.ID)
		if err != nil {
			return err
		}

		byModule := make(map[uuid.UUID][]database.Lesson, len(modules))
		for _, l := range lessons {
			byModule[l.ModuleID] = append(byModule[l.ModuleID], l)
		}
		out.Course = course
		out.Modules = make([]outlineModule, 0, len(modules))
		for _, m := range modules {
			out.Modules = append(out.Modules, outlineModule{Module: m, Lessons: emptyIfNil(byModule[m.ID])})
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, out)
}

type publishRequest struct {
	Status string `json:"status"`
}

func (s *Server) setCourseStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body publishRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	status, err := parseStatus(body.Status)
	if err != nil {
		return err
	}

	var course database.Course
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		course, err = q.SetCourseStatus(r.Context(), database.SetCourseStatusParams{ID: id, Status: status})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, course)
}

type createModuleRequest struct {
	Title string `json:"title"`
}

func (s *Server) createModule(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body createModuleRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var module database.Module
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		if _, err := q.GetCourse(r.Context(), courseID); err != nil {
			return err
		}
		var err error
		module, err = q.CreateModule(r.Context(), database.CreateModuleParams{
			TenantID: tenant.ID, CourseID: courseID, Title: title,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, module)
}

type createLessonRequest struct {
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	Body      string `json:"body"`
	Dir       string `json:"dir"`
	DurationS int32  `json:"duration_s"`
	IsPreview bool   `json:"is_preview"`
}

func (s *Server) createLesson(w http.ResponseWriter, r *http.Request) error {
	moduleID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body createLessonRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	kind, err := parseLessonKind(body.Kind)
	if err != nil {
		return err
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	if body.DurationS < 0 {
		return invalid("duration_s", "Duration cannot be negative.")
	}

	tenant := CurrentTenant(r.Context())
	var lesson database.Lesson
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		lesson, err = q.CreateLesson(r.Context(), database.CreateLessonParams{
			TenantID: tenant.ID, ModuleID: moduleID, Title: title, Kind: kind,
			Body: body.Body, Dir: dir, DurationS: body.DurationS, IsPreview: body.IsPreview,
		})
		return err
	})
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, lesson)
}

type moveLessonRequest struct {
	ModuleID string  `json:"module_id"`
	Position float64 `json:"position"`
}

// moveLesson takes the position the client computed, so a drag writes one row.
func (s *Server) moveLesson(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body moveLessonRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	moduleID, err := uuid.Parse(strings.TrimSpace(body.ModuleID))
	if err != nil {
		return invalid("module_id", "Provide the id of the module to move into.")
	}
	if body.Position <= 0 || body.Position > 1e15 {
		return invalid("position", "Position must be a positive number.")
	}

	var lesson database.Lesson
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		lesson, err = q.MoveLesson(r.Context(), database.MoveLessonParams{
			ID: id, ModuleID: moduleID, Position: body.Position,
		})
		return err
	})
	if database.IsNotFound(err) || isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, lesson)
}

func (s *Server) deleteLesson(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.DeleteLesson(r.Context(), id)
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	return httpx.NoContent(w)
}

func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, httpx.ErrNotFound
	}
	return id, nil
}

func parseDir(v string) (database.TextDir, error) {
	switch database.TextDir(strings.TrimSpace(v)) {
	case "", database.TextDirAuto:
		return database.TextDirAuto, nil
	case database.TextDirLtr:
		return database.TextDirLtr, nil
	case database.TextDirRtl:
		return database.TextDirRtl, nil
	default:
		return "", invalid("dir", "Direction must be auto, ltr or rtl.")
	}
}

func parseVisibility(v string) (database.CourseVisibility, error) {
	switch database.CourseVisibility(strings.TrimSpace(v)) {
	case "", database.CourseVisibilityPrivate:
		return database.CourseVisibilityPrivate, nil
	case database.CourseVisibilityUnlisted:
		return database.CourseVisibilityUnlisted, nil
	case database.CourseVisibilityPublic:
		return database.CourseVisibilityPublic, nil
	default:
		return "", invalid("visibility", "Visibility must be private, unlisted or public.")
	}
}

func parseStatus(v string) (database.PublishStatus, error) {
	switch database.PublishStatus(strings.TrimSpace(v)) {
	case database.PublishStatusDraft:
		return database.PublishStatusDraft, nil
	case database.PublishStatusPublished:
		return database.PublishStatusPublished, nil
	case database.PublishStatusArchived:
		return database.PublishStatusArchived, nil
	default:
		return "", invalid("status", "Status must be draft, published or archived.")
	}
}

func parseLessonKind(v string) (database.LessonKind, error) {
	k := database.LessonKind(strings.TrimSpace(v))
	switch k {
	case "":
		return database.LessonKindText, nil
	case database.LessonKindVideo, database.LessonKindAudio, database.LessonKindText,
		database.LessonKindPdf, database.LessonKindLink, database.LessonKindLive,
		database.LessonKindQuiz, database.LessonKindAssignment:
		return k, nil
	default:
		return "", invalid("kind", "Unknown lesson kind.")
	}
}

func emptyIfNil(l []database.Lesson) []database.Lesson {
	if l == nil {
		return []database.Lesson{}
	}
	return l
}

func isUniqueViolation(err error) bool { return pgErrCode(err) == "23505" }

func isForeignKeyViolation(err error) bool { return pgErrCode(err) == "23503" }

func pgErrCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
