package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestNotices covers telling a section something, and a guardian finding their
// own children.
func TestNotices(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	guardian := enrollIn(t, h, ch, store, owner.slug, "student")
	outsider := enrollIn(t, h, ch, store, owner.slug, "student")

	createdID(t, do(t, h, "POST", "/v1/academics/years", owner.token, owner.slug,
		map[string]any{"name": "notice-year", "starts_on": "2026-01-01", "ends_on": "2026-12-31"}))
	yearID := currentYearID(t, h, owner)
	if rec := do(t, h, "PUT", "/v1/academics/years/"+yearID+"/current", owner.token, owner.slug,
		nil); rec.Code != http.StatusOK {
		t.Fatalf("make current: got %d: %s", rec.Code, rec.Body)
	}
	classID := createdID(t, do(t, h, "POST", "/v1/academics/classes", owner.token, owner.slug,
		map[string]any{"name": "Notice class"}))
	sectionID := createdID(t, do(t, h, "POST", "/v1/academics/classes/"+classID+"/sections",
		owner.token, owner.slug, map[string]any{"name": "A"}))
	if rec := do(t, h, "POST", "/v1/academics/sections/"+sectionID+"/roll", owner.token, owner.slug,
		map[string]any{"user_id": student.userID}); rec.Code != http.StatusCreated {
		t.Fatalf("place: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/v1/guardians", owner.token, owner.slug, map[string]any{
		"guardian_id": guardian.userID, "student_id": student.userID, "relation": "mother",
	}); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("link: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("a learner cannot send a notice", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/notices", student.token, owner.slug, map[string]any{
			"audience": "school", "title": "No lessons", "body": "Please stay home.",
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a notice to a section reaches its guardians", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/notices", owner.token, owner.slug, map[string]any{
			"audience": "section", "target_id": sectionID, "to": "guardians",
			"title": "Closed on Thursday", "body": "The madrasah is closed for the holiday.",
		})
		if rec.Code != http.StatusAccepted {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			SentTo int `json:"sent_to"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.SentTo != 1 {
			t.Fatalf("sent to %d people, want the one guardian", out.SentTo)
		}

		// It lands in the guardian's inbox, and in nobody else's.
		if got := inboxCount(t, h, guardian, "Closed on Thursday"); got != 1 {
			t.Fatalf("the guardian has %d copies", got)
		}
		if got := inboxCount(t, h, outsider, "Closed on Thursday"); got != 0 {
			t.Fatalf("somebody outside the section got %d copies", got)
		}
	})

	t.Run("a section nobody is in is refused rather than sent to nobody", func(t *testing.T) {
		empty := createdID(t, do(t, h, "POST", "/v1/academics/classes/"+classID+"/sections",
			owner.token, owner.slug, map[string]any{"name": "Empty"}))
		rec := do(t, h, "POST", "/v1/notices", owner.token, owner.slug, map[string]any{
			"audience": "section", "target_id": empty, "title": "Hello", "body": "Anybody?",
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a guardian finds their own children and where they sit", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/children", guardian.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Children []struct {
				StudentID   string  `json:"student_id"`
				Relation    string  `json:"relation"`
				FullName    string  `json:"full_name"`
				SectionName *string `json:"section_name"`
				ClassName   *string `json:"class_name"`
			} `json:"children"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Children) != 1 || out.Children[0].Relation != "mother" {
			t.Fatalf("got %s", rec.Body)
		}
		if out.Children[0].SectionName == nil || *out.Children[0].SectionName != "A" {
			t.Fatalf("the child's section came back as %v", out.Children[0].SectionName)
		}
	})

	t.Run("somebody with no children gets an empty list, not an error", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/children", outsider.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
	})
}

func currentYearID(t *testing.T, h http.Handler, a actor) string {
	t.Helper()
	rec := do(t, h, "GET", "/v1/academics/years", a.token, a.slug, nil)
	var out struct {
		Years []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"years"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, year := range out.Years {
		if year.Name == "notice-year" {
			return year.ID
		}
	}
	t.Fatal("the year just created is not in the list")
	return ""
}

func inboxCount(t *testing.T, h http.Handler, a actor, title string) int {
	t.Helper()
	rec := do(t, h, "GET", "/v1/notifications", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("inbox: got %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Notifications []struct {
			Title string `json:"title"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	seen := 0
	for _, row := range out.Notifications {
		if row.Title == title {
			seen++
		}
	}
	return seen
}
