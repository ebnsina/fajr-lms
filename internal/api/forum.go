package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// The course forum. Anybody on the course may ask and answer; staff moderate.
// A removed post keeps its place in the thread, so what follows still reads.

type threadRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Dir   string `json:"dir"`
}

func (s *Server) startThread(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body threadRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	first, err := requireText("body", body.Body, 10000)
	if err != nil {
		return err
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	author := uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true}
	var thread database.ForumThread

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		thread, err = q.CreateThread(r.Context(), database.CreateThreadParams{
			TenantID: tenant.ID, CourseID: courseID, Title: title, Dir: dir, AuthorID: author,
		})
		if err != nil {
			return err
		}
		// The question is the thread's first post, so a thread is never empty.
		_, err = q.AddPost(r.Context(), database.AddPostParams{
			TenantID: tenant.ID, ThreadID: thread.ID, AuthorID: author, Body: first, Dir: dir,
		})
		return err
	})
	if isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, thread)
}

func (s *Server) listThreads(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var rows []database.ListThreadsRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListThreads(r.Context(), database.ListThreadsParams{
			CourseID: courseID, PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"threads": rows})
}

func (s *Server) readThread(w http.ResponseWriter, r *http.Request) error {
	threadID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var thread database.GetThreadRow
	var posts []database.ListPostsRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if thread, err = q.GetThread(r.Context(), threadID); err != nil {
			return err
		}
		posts, err = q.ListPosts(r.Context(), threadID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	// A removed post's words are not sent out again, only the fact of it.
	shown := make([]map[string]any, 0, len(posts))
	for _, row := range posts {
		post := map[string]any{
			"id": row.ForumPost.ID, "created_at": row.ForumPost.CreatedAt,
			"author_id": row.ForumPost.AuthorID, "author_name": row.AuthorName,
			"dir": row.ForumPost.Dir, "removed": row.ForumPost.RemovedAt.Valid,
			"body": row.ForumPost.Body,
		}
		if row.ForumPost.RemovedAt.Valid {
			post["body"] = ""
		}
		shown = append(shown, post)
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"thread": thread, "posts": shown})
}

type postRequest struct {
	Body string `json:"body"`
	Dir  string `json:"dir"`
}

func (s *Server) replyToThread(w http.ResponseWriter, r *http.Request) error {
	threadID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body postRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	text, err := requireText("body", body.Body, 10000)
	if err != nil {
		return err
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	author := uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true}
	var post database.ForumPost

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		thread, err := q.GetThread(r.Context(), threadID)
		if err != nil {
			return err
		}
		if thread.ForumThread.Locked {
			return httpx.Errorf(http.StatusConflict, "thread_locked",
				"This thread is closed to new replies.")
		}
		if post, err = q.AddPost(r.Context(), database.AddPostParams{
			TenantID: tenant.ID, ThreadID: threadID, AuthorID: author, Body: text, Dir: dir,
		}); err != nil {
			return err
		}
		return q.CountReply(r.Context(), threadID)
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, post)
}

// editPost lets somebody fix their own words. Staff moderate by removing, not
// by rewriting what another person said.
func (s *Server) editPost(w http.ResponseWriter, r *http.Request) error {
	postID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body postRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	text, err := requireText("body", body.Body, 10000)
	if err != nil {
		return err
	}

	caller := Authenticated(r.Context()).UserID
	var post database.ForumPost
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		existing, err := q.GetPost(r.Context(), postID)
		if err != nil {
			return err
		}
		if !existing.AuthorID.Valid || existing.AuthorID.UUID != caller {
			return httpx.ErrForbidden
		}
		post, err = q.EditPost(r.Context(), database.EditPostParams{ID: postID, Body: text})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, post)
}

// removePost is moderation, or somebody withdrawing their own words.
func (s *Server) removePost(w http.ResponseWriter, r *http.Request) error {
	postID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	caller := Authenticated(r.Context()).UserID
	staff := false
	switch CurrentRole(r.Context()) {
	case "owner", "admin", "instructor", "assistant":
		staff = true
	}

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		existing, err := q.GetPost(r.Context(), postID)
		if err != nil {
			return err
		}
		mine := existing.AuthorID.Valid && existing.AuthorID.UUID == caller
		if !staff && !mine {
			return httpx.ErrForbidden
		}
		_, err = q.RemovePost(r.Context(), database.RemovePostParams{
			ID: postID, RemovedBy: uuid.NullUUID{UUID: caller, Valid: true},
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type threadFlagsRequest struct {
	Pinned *bool `json:"pinned"`
	Locked *bool `json:"locked"`
}

func (s *Server) setThreadFlags(w http.ResponseWriter, r *http.Request) error {
	threadID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body threadFlagsRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.Pinned == nil && body.Locked == nil {
		return invalid("pinned", "Say what to change.")
	}

	var thread database.ForumThread
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		thread, err = q.SetThreadFlags(r.Context(), database.SetThreadFlagsParams{
			ID: threadID, Pinned: body.Pinned, Locked: body.Locked,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, thread)
}

func (s *Server) deleteThread(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteThread(r.Context(), id)
	})
}
