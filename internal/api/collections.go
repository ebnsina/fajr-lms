package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// Topics are what a course is about. Collections are several courses together:
// a path is worked through in order, a bundle is bought in one go. They share
// a table because they share a shape.

type topicRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Server) createTopic(w http.ResponseWriter, r *http.Request) error {
	var body topicRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 60)
	if err != nil {
		return err
	}
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		slug = Slugify(name)
	}
	if !slugRe.MatchString(slug) {
		return invalid("slug", "An address is lowercase letters, numbers and hyphens.")
	}

	tenant := CurrentTenant(r.Context())
	var topic database.Topic
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		topic, err = q.CreateTopic(r.Context(), database.CreateTopicParams{
			TenantID: tenant.ID, Name: name, Slug: slug,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "topic_exists", "That topic is already on the list.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, topic)
}

func (s *Server) listTopics(w http.ResponseWriter, r *http.Request) error {
	var rows []database.ListTopicsRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListTopics(r.Context())
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"topics": rows})
}

func (s *Server) deleteTopic(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteTopic(r.Context(), id)
	})
}

// courseTopics is what one course is filed under.
func (s *Server) courseTopics(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var topics []database.Topic
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		topics, err = q.TopicsOfCourse(r.Context(), courseID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"topics": topics})
}

type courseTopicsRequest struct {
	TopicIDs []string `json:"topic_ids"`
}

// setCourseTopics replaces what a course is filed under, in one call, so the
// page can send the whole set rather than a diff.
func (s *Server) setCourseTopics(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body courseTopicsRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	wanted, err := parseUUIDs(body.TopicIDs)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var topics []database.Topic
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		existing, err := q.TopicsOfCourse(r.Context(), courseID)
		if err != nil {
			return err
		}
		keep := make(map[uuid.UUID]bool, len(wanted))
		for _, id := range wanted {
			keep[id] = true
		}
		for _, topic := range existing {
			if !keep[topic.ID] {
				if _, err := q.UntagCourse(r.Context(), database.UntagCourseParams{
					CourseID: courseID, TopicID: topic.ID,
				}); err != nil {
					return err
				}
			}
		}
		for _, id := range wanted {
			if err := q.TagCourse(r.Context(), database.TagCourseParams{
				CourseID: courseID, TopicID: id, TenantID: tenant.ID,
			}); err != nil {
				return err
			}
		}
		topics, err = q.TopicsOfCourse(r.Context(), courseID)
		return err
	})
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"topics": topics})
}

type collectionRequest struct {
	Kind       string `json:"kind"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Dir        string `json:"dir"`
	PriceMinor int64  `json:"price_minor"`
}

func (s *Server) createCollection(w http.ResponseWriter, r *http.Request) error {
	var body collectionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	kind := database.CollectionKindPath
	switch strings.ToLower(strings.TrimSpace(body.Kind)) {
	case "", "path":
	case "bundle":
		kind = database.CollectionKindBundle
	default:
		return invalid("kind", "A collection is a path or a bundle.")
	}
	if kind == database.CollectionKindPath && body.PriceMinor != 0 {
		return invalid("price_minor", "A path is worked through, not sold. Make it a bundle to charge for it.")
	}
	if body.PriceMinor < 0 {
		return invalid("price_minor", "A price cannot be negative.")
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		slug = Slugify(title)
	}
	if !slugRe.MatchString(slug) {
		return invalid("slug", "An address is lowercase letters, numbers and hyphens.")
	}

	tenant := CurrentTenant(r.Context())
	var collection database.Collection
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		collection, err = q.CreateCollection(r.Context(), database.CreateCollectionParams{
			TenantID: tenant.ID, Kind: kind, Slug: slug, Title: title,
			Summary: strings.TrimSpace(body.Summary), Dir: dir,
			PriceMinor: body.PriceMinor, Currency: tenant.Currency,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "slug_taken", "Something already lives at that address.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, collection)
}

func (s *Server) listCollections(w http.ResponseWriter, r *http.Request) error {
	// No kind asked for means both.
	var kind *database.CollectionKind
	switch strings.ToLower(r.URL.Query().Get("kind")) {
	case "path":
		wanted := database.CollectionKindPath
		kind = &wanted
	case "bundle":
		wanted := database.CollectionKindBundle
		kind = &wanted
	}

	var rows []database.ListCollectionsRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListCollections(r.Context(), kind)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"collections": rows})
}

// readCollection is the path or bundle with its courses in order, and how far
// the person reading it has got through them.
func (s *Server) readCollection(w http.ResponseWriter, r *http.Request) error {
	slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
	userID := Authenticated(r.Context()).UserID

	var collection database.Collection
	var courses []database.CollectionCoursesRow
	var done int
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if collection, err = q.GetCollection(r.Context(), slug); err != nil {
			return err
		}
		if courses, err = q.CollectionCourses(r.Context(), collection.ID); err != nil {
			return err
		}
		// A course counts as finished when the person's enrolment says so.
		for _, row := range courses {
			enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
				CourseID: row.Course.ID, UserID: userID,
			})
			if database.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if enrollment.Status == database.EnrollmentStatusCompleted {
				done++
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
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"collection": collection, "courses": courses, "courses_done": done,
	})
}

type collectionCourseRequest struct {
	CourseID string `json:"course_id"`
}

func (s *Server) addToCollection(w http.ResponseWriter, r *http.Request) error {
	collectionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body collectionCourseRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	courseID, err := uuid.Parse(strings.TrimSpace(body.CourseID))
	if err != nil {
		return invalid("course_id", "Name the course to add.")
	}

	tenant := CurrentTenant(r.Context())
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		return q.AddCourseToCollection(r.Context(), database.AddCourseToCollectionParams{
			CollectionID: collectionID, CourseID: courseID, TenantID: tenant.ID,
		})
	})
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) removeFromCollection(w http.ResponseWriter, r *http.Request) error {
	collectionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	courseID, err := pathUUID(r, "courseId")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.RemoveCourseFromCollection(r.Context(), database.RemoveCourseFromCollectionParams{
			CollectionID: collectionID, CourseID: courseID,
		})
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

type updateCollectionRequest struct {
	Title      *string `json:"title"`
	Summary    *string `json:"summary"`
	Status     *string `json:"status"`
	PriceMinor *int64  `json:"price_minor"`
}

func (s *Server) updateCollection(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body updateCollectionRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	params := database.UpdateCollectionParams{ID: id, Summary: body.Summary}
	if body.Title != nil {
		title, err := requireText("title", *body.Title, 200)
		if err != nil {
			return err
		}
		params.Title = &title
	}
	if body.Status != nil {
		status, err := parseStatus(*body.Status)
		if err != nil {
			return err
		}
		params.Status = &status
	}
	if body.PriceMinor != nil {
		if *body.PriceMinor < 0 {
			return invalid("price_minor", "A price cannot be negative.")
		}
		params.PriceMinor = body.PriceMinor
	}

	var collection database.Collection
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		collection, err = q.UpdateCollection(r.Context(), params)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, collection)
}

func (s *Server) deleteCollection(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteCollection(r.Context(), id)
	})
}
