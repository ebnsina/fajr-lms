package api_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
)

// TestPoints covers a school switching the leaderboard on, what earns points,
// and the badge that follows.
func TestPoints(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	courseID, lessons := publishedCourse(t, h, owner, 1)
	lessonID := lessons[0]
	if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", student.token, owner.slug,
		nil); rec.Code != http.StatusCreated {
		t.Fatalf("enroll: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("with points off, nothing is counted and there is no board", func(t *testing.T) {
		if rec := do(t, h, "PUT", "/v1/lessons/"+lessonID+"/progress", student.token, owner.slug,
			map[string]any{"completed": true, "position_s": 0}); rec.Code != http.StatusOK {
			t.Fatalf("progress: got %d: %s", rec.Code, rec.Body)
		}
		if got := standing(t, h, student); got != 0 {
			t.Fatalf("a school with points off counted %d", got)
		}
		rec := do(t, h, "GET", "/v1/points/board", student.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("board: got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("only the office switches it on", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/points/setting", student.token, owner.slug,
			map[string]any{"on": true})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	if rec := do(t, h, "PUT", "/v1/points/setting", owner.token, owner.slug,
		map[string]any{"on": true}); rec.Code != http.StatusOK {
		t.Fatalf("switch on: got %d: %s", rec.Code, rec.Body)
	}
	badgeName := fmt.Sprintf("First steps %d", rand.IntN(100000))
	if rec := do(t, h, "POST", "/v1/badges", owner.token, owner.slug, map[string]any{
		"name": badgeName, "emoji": "🌅", "threshold": 10,
		"description": "Finished a first lesson.",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("badge: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("finishing a lesson pays, and pays once", func(t *testing.T) {
		learner := enrollIn(t, h, ch, store, owner.slug, "student")
		if rec := do(t, h, "POST", "/v1/courses/"+courseID+"/enrollments", learner.token, owner.slug,
			nil); rec.Code != http.StatusCreated {
			t.Fatalf("enroll: got %d: %s", rec.Code, rec.Body)
		}
		for range 3 {
			if rec := do(t, h, "PUT", "/v1/lessons/"+lessonID+"/progress", learner.token, owner.slug,
				map[string]any{"completed": true, "position_s": 0}); rec.Code != http.StatusOK {
				t.Fatalf("progress: got %d: %s", rec.Code, rec.Body)
			}
		}
		// Ten for the lesson and a hundred for finishing the one-lesson course.
		if got := standing(t, h, learner); got != 110 {
			t.Fatalf("got %d points, want 110 paid once", got)
		}

		rec := do(t, h, "GET", "/v1/points/me", learner.token, owner.slug, nil)
		var out struct {
			Badges []struct {
				Name string `json:"name"`
			} `json:"badges"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		found := false
		for _, badge := range out.Badges {
			if badge.Name == badgeName {
				found = true
			}
		}
		if !found {
			t.Fatalf("the badge was not earned: %s", rec.Body)
		}
	})

	t.Run("the board ranks by points", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/points/board", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Standings []struct {
				FullName string `json:"full_name"`
				Points   int64  `json:"points"`
			} `json:"standings"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Standings) == 0 {
			t.Fatal("nobody is on the board")
		}
		for i := 1; i < len(out.Standings); i++ {
			if out.Standings[i-1].Points < out.Standings[i].Points {
				t.Fatalf("the board is out of order: %+v", out.Standings)
			}
		}
	})
}

func standing(t *testing.T, h http.Handler, a actor) int64 {
	t.Helper()
	rec := do(t, h, "GET", "/v1/points/me", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("points: got %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Points int64 `json:"points"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out.Points
}
