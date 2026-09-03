package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestForum covers a course discussion: asking, answering, editing, and what
// moderation does to a thread.
func TestForum(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	classmate := enrollIn(t, h, ch, store, owner.slug, "student")

	courseID, _ := publishedCourse(t, h, owner, 1)

	threadID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/threads",
		student.token, owner.slug,
		map[string]any{"title": "Why does the verb come first?", "body": "I keep getting this wrong."}))

	t.Run("a thread starts with the question in it", func(t *testing.T) {
		thread, posts := readThread(t, h, student, threadID)
		if thread.Title != "Why does the verb come first?" {
			t.Fatalf("title is %q", thread.Title)
		}
		if len(posts) != 1 || !strings.Contains(posts[0].Body, "keep getting this wrong") {
			t.Fatalf("got %+v", posts)
		}
	})

	t.Run("a thread with no words is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/threads", student.token, owner.slug,
			map[string]any{"title": "Hello", "body": "   "})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	var replyID string
	t.Run("somebody answers, and the thread counts it", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/threads/"+threadID+"/posts", owner.token, owner.slug,
			map[string]any{"body": "It does not always. Look at the nominal sentence."})
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		replyID = createdID(t, rec)

		thread, posts := readThread(t, h, student, threadID)
		if thread.ReplyCount != 1 || len(posts) != 2 {
			t.Fatalf("thread counts %d replies over %d posts", thread.ReplyCount, len(posts))
		}
	})

	t.Run("a person may fix their own words and nobody else's", func(t *testing.T) {
		if rec := do(t, h, "PATCH", "/v1/posts/"+replyID, owner.token, owner.slug,
			map[string]any{"body": "It does not always — see the nominal sentence."}); rec.Code != http.StatusOK {
			t.Fatalf("own edit: got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "PATCH", "/v1/posts/"+replyID, classmate.token, owner.slug,
			map[string]any{"body": "I am rewriting your answer."})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("somebody else's edit: got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a locked thread takes no more replies", func(t *testing.T) {
		if rec := do(t, h, "PUT", "/v1/threads/"+threadID+"/flags", owner.token, owner.slug,
			map[string]any{"locked": true, "pinned": true}); rec.Code != http.StatusOK {
			t.Fatalf("lock: got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "POST", "/v1/threads/"+threadID+"/posts", classmate.token, owner.slug,
			map[string]any{"body": "One more thing"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "PUT", "/v1/threads/"+threadID+"/flags", owner.token, owner.slug,
			map[string]any{"locked": false}); rec.Code != http.StatusOK {
			t.Fatalf("unlock: got %d: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a learner cannot moderate", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/threads/"+threadID+"/flags", student.token, owner.slug,
			map[string]any{"pinned": true})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a removed post keeps its place but not its words", func(t *testing.T) {
		if rec := do(t, h, "DELETE", "/v1/posts/"+replyID, owner.token, owner.slug,
			nil); rec.Code != http.StatusNoContent {
			t.Fatalf("remove: got %d: %s", rec.Code, rec.Body)
		}
		_, posts := readThread(t, h, student, threadID)
		if len(posts) != 2 {
			t.Fatalf("the thread lost a post: %d remain", len(posts))
		}
		gone := posts[1]
		if !gone.Removed || gone.Body != "" {
			t.Fatalf("the removed post still reads: %+v", gone)
		}
	})
}

type threadView struct {
	Title      string `json:"title"`
	ReplyCount int32  `json:"reply_count"`
	Pinned     bool   `json:"pinned"`
	Locked     bool   `json:"locked"`
}

type postView struct {
	ID      string `json:"id"`
	Body    string `json:"body"`
	Removed bool   `json:"removed"`
}

func readThread(t *testing.T, h http.Handler, a actor, threadID string) (threadView, []postView) {
	t.Helper()
	rec := do(t, h, "GET", "/v1/threads/"+threadID, a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("read thread: got %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Thread struct {
			ForumThread threadView `json:"forum_thread"`
		} `json:"thread"`
		Posts []postView `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out.Thread.ForumThread, out.Posts
}
