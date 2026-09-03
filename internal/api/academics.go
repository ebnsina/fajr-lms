package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// The academic spine: the year a school teaches in, the terms inside it, and
// the classes, sections and subjects everything else is arranged by. A school
// names its own ladder, because no two agree on one.

func parseDay(field, raw string) (pgtype.Date, error) {
	day, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return pgtype.Date{}, invalid(field, "Write the date as 2026-01-31.")
	}
	return pgtype.Date{Time: day, Valid: true}, nil
}

type yearRequest struct {
	Name     string `json:"name"`
	StartsOn string `json:"starts_on"`
	EndsOn   string `json:"ends_on"`
}

func (s *Server) createYear(w http.ResponseWriter, r *http.Request) error {
	var body yearRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 60)
	if err != nil {
		return err
	}
	starts, err := parseDay("starts_on", body.StartsOn)
	if err != nil {
		return err
	}
	ends, err := parseDay("ends_on", body.EndsOn)
	if err != nil {
		return err
	}
	if !ends.Time.After(starts.Time) {
		return invalid("ends_on", "A year ends after it starts.")
	}

	tenant := CurrentTenant(r.Context())
	var year database.AcademicYear
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		year, err = q.CreateAcademicYear(r.Context(), database.CreateAcademicYearParams{
			TenantID: tenant.ID, Name: name, StartsOn: starts, EndsOn: ends,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "year_exists", "This school already has a year by that name.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, year)
}

func (s *Server) listYears(w http.ResponseWriter, r *http.Request) error {
	var years []database.AcademicYear
	var terms []database.Term
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if years, err = q.ListAcademicYears(r.Context()); err != nil {
			return err
		}
		// The terms of the current year, which is the only one being taught in.
		current, err := q.CurrentYear(r.Context())
		if database.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		terms, err = q.ListTerms(r.Context(), current.ID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"years": years, "terms": orEmptyTerms(terms),
	})
}

func orEmptyTerms(terms []database.Term) []database.Term {
	if terms == nil {
		return []database.Term{}
	}
	return terms
}

func (s *Server) makeYearCurrent(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var year database.AcademicYear
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		if err := q.ClearCurrentYear(r.Context()); err != nil {
			return err
		}
		var err error
		year, err = q.MakeYearCurrent(r.Context(), id)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, year)
}

func (s *Server) deleteYear(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteAcademicYear(r.Context(), id)
	})
}

type termRequest struct {
	Name     string `json:"name"`
	StartsOn string `json:"starts_on"`
	EndsOn   string `json:"ends_on"`
}

func (s *Server) createTerm(w http.ResponseWriter, r *http.Request) error {
	yearID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body termRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 60)
	if err != nil {
		return err
	}
	starts, err := parseDay("starts_on", body.StartsOn)
	if err != nil {
		return err
	}
	ends, err := parseDay("ends_on", body.EndsOn)
	if err != nil {
		return err
	}
	if !ends.Time.After(starts.Time) {
		return invalid("ends_on", "A term ends after it starts.")
	}

	tenant := CurrentTenant(r.Context())
	var term database.Term
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		term, err = q.CreateTerm(r.Context(), database.CreateTermParams{
			TenantID: tenant.ID, YearID: yearID, Name: name, StartsOn: starts, EndsOn: ends,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "term_exists", "That year already has a term by that name.")
	}
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, term)
}

func (s *Server) makeTermCurrent(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var term database.Term
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		if err := q.ClearCurrentTerm(r.Context()); err != nil {
			return err
		}
		var err error
		term, err = q.MakeTermCurrent(r.Context(), id)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, term)
}

func (s *Server) deleteTerm(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteTerm(r.Context(), id)
	})
}

type classRequest struct {
	Name string `json:"name"`
	Rank *int32 `json:"rank"`
}

func (s *Server) createClass(w http.ResponseWriter, r *http.Request) error {
	var body classRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 100)
	if err != nil {
		return err
	}
	rank := int32(0)
	if body.Rank != nil {
		rank = *body.Rank
	}

	tenant := CurrentTenant(r.Context())
	var class database.Class
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		class, err = q.CreateClass(r.Context(), database.CreateClassParams{
			TenantID: tenant.ID, Name: name, Rank: rank,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "class_exists", "This school already has a class by that name.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, class)
}

func (s *Server) listClasses(w http.ResponseWriter, r *http.Request) error {
	var classes []database.Class
	var sections []database.ListSectionsRow
	var subjects []database.ListSubjectsRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if classes, err = q.ListClasses(r.Context()); err != nil {
			return err
		}
		if sections, err = q.ListSections(r.Context()); err != nil {
			return err
		}
		subjects, err = q.ListSubjects(r.Context())
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"classes": classes, "sections": sections, "subjects": subjects,
	})
}

func (s *Server) updateClass(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body classRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	params := database.UpdateClassParams{ID: id, Rank: body.Rank}
	if strings.TrimSpace(body.Name) != "" {
		name, err := requireText("name", body.Name, 100)
		if err != nil {
			return err
		}
		params.Name = &name
	}

	var class database.Class
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		class, err = q.UpdateClass(r.Context(), params)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "class_exists", "This school already has a class by that name.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, class)
}

func (s *Server) deleteClass(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteClass(r.Context(), id)
	})
}

type sectionRequest struct {
	Name      string  `json:"name"`
	Capacity  *int32  `json:"capacity"`
	TeacherID *string `json:"teacher_id"`
}

func (s *Server) createSection(w http.ResponseWriter, r *http.Request) error {
	classID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body sectionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 60)
	if err != nil {
		return err
	}
	if body.Capacity != nil && *body.Capacity <= 0 {
		return invalid("capacity", "A section holds at least one student.")
	}
	teacher, err := optionalUUID("teacher_id", body.TeacherID)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var section database.Section
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		section, err = q.CreateSection(r.Context(), database.CreateSectionParams{
			TenantID: tenant.ID, ClassID: classID, Name: name,
			Capacity: body.Capacity, TeacherID: teacher,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "section_exists", "That class already has a section by that name.")
	}
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, section)
}

func (s *Server) updateSection(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body sectionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	params := database.UpdateSectionParams{ID: id}
	if strings.TrimSpace(body.Name) != "" {
		name, err := requireText("name", body.Name, 60)
		if err != nil {
			return err
		}
		params.Name = &name
	}
	if body.Capacity != nil {
		if *body.Capacity <= 0 {
			return invalid("capacity", "A section holds at least one student.")
		}
		params.SetCapacity, params.Capacity = true, body.Capacity
	}
	if body.TeacherID != nil {
		teacher, err := optionalUUID("teacher_id", body.TeacherID)
		if err != nil {
			return err
		}
		params.SetTeacher, params.TeacherID = true, teacher
	}

	var section database.Section
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		section, err = q.UpdateSection(r.Context(), params)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, section)
}

func (s *Server) deleteSection(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteSection(r.Context(), id)
	})
}

type subjectRequest struct {
	Name    string  `json:"name"`
	Code    string  `json:"code"`
	Dir     string  `json:"dir"`
	ClassID *string `json:"class_id"`
}

func (s *Server) createSubject(w http.ResponseWriter, r *http.Request) error {
	var body subjectRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 100)
	if err != nil {
		return err
	}
	if len(body.Code) > 30 {
		return invalid("code", "A subject code is 30 characters at most.")
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	class, err := optionalUUID("class_id", body.ClassID)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var subject database.Subject
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		subject, err = q.CreateSubject(r.Context(), database.CreateSubjectParams{
			TenantID: tenant.ID, ClassID: class, Name: name,
			Code: strings.TrimSpace(body.Code), Dir: dir,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "subject_exists", "That subject is already on the list.")
	}
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, subject)
}

func (s *Server) deleteSubject(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteSubject(r.Context(), id)
	})
}

type placementRequest struct {
	UserID string `json:"user_id"`
	RollNo *int32 `json:"roll_no"`
}

// placeStudent puts a student in a section for the year being taught in. A
// student sits in one section per year, so this moves them rather than adding
// a second seat.
func (s *Server) placeStudent(w http.ResponseWriter, r *http.Request) error {
	sectionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body placementRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	userID, err := uuid.Parse(strings.TrimSpace(body.UserID))
	if err != nil {
		return invalid("user_id", "Name the student to place.")
	}
	if body.RollNo != nil && *body.RollNo <= 0 {
		return invalid("roll_no", "A roll number counts from one.")
	}

	tenant := CurrentTenant(r.Context())
	var placement database.Placement
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		year, err := q.CurrentYear(r.Context())
		if database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "no_current_year",
				"Set the year this school is teaching in first.")
		}
		if err != nil {
			return err
		}
		placement, err = q.PlaceStudent(r.Context(), database.PlaceStudentParams{
			TenantID: tenant.ID, YearID: year.ID, SectionID: sectionID,
			UserID: userID, RollNo: body.RollNo,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "roll_taken", "That roll number is already used in this section.")
	}
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, placement)
}

func (s *Server) sectionRoll(w http.ResponseWriter, r *http.Request) error {
	sectionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows []database.ListSectionRollRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListSectionRoll(r.Context(), sectionID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"students": rows})
}

func (s *Server) removePlacement(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.RemovePlacement(r.Context(), id)
	})
}

// removeRow is the shape every delete here takes: gone is 204, missing is 404.
func (s *Server) removeRow(w http.ResponseWriter, r *http.Request,
	remove func(*database.Queries, uuid.UUID) (int64, error)) error {

	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = remove(q, id)
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

func optionalUUID(field string, raw *string) (uuid.NullUUID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return uuid.NullUUID{}, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(*raw))
	if err != nil {
		return uuid.NullUUID{}, invalid(field, "That is not something on this school's list.")
	}
	return uuid.NullUUID{UUID: parsed, Valid: true}, nil
}
