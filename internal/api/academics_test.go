package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestAcademicSpine walks a school setting itself up: a year, its terms, the
// classes it teaches, and who sits where.
func TestAcademicSpine(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	yearID := createdID(t, do(t, h, "POST", "/v1/academics/years", owner.token, owner.slug,
		map[string]any{"name": "2026", "starts_on": "2026-01-01", "ends_on": "2026-12-31"}))

	t.Run("a year that ends before it starts is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/academics/years", owner.token, owner.slug,
			map[string]any{"name": "backwards", "starts_on": "2026-12-31", "ends_on": "2026-01-01"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("only the office changes the spine", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/academics/classes", student.token, owner.slug,
			map[string]any{"name": "Class Six"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("one year is current, and choosing another puts the first back", func(t *testing.T) {
		second := createdID(t, do(t, h, "POST", "/v1/academics/years", owner.token, owner.slug,
			map[string]any{"name": "2027", "starts_on": "2027-01-01", "ends_on": "2027-12-31"}))
		for _, id := range []string{yearID, second, yearID} {
			if rec := do(t, h, "PUT", "/v1/academics/years/"+id+"/current", owner.token, owner.slug,
				nil); rec.Code != http.StatusOK {
				t.Fatalf("make current: got %d: %s", rec.Code, rec.Body)
			}
		}
		rec := do(t, h, "GET", "/v1/academics/years", owner.token, owner.slug, nil)
		var out struct {
			Years []struct {
				ID        string `json:"id"`
				IsCurrent bool   `json:"is_current"`
			} `json:"years"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		current := 0
		for _, year := range out.Years {
			if year.IsCurrent {
				current++
				if year.ID != yearID {
					t.Fatalf("the wrong year is current: %s", year.ID)
				}
			}
		}
		if current != 1 {
			t.Fatalf("%d years are current, want 1", current)
		}
	})

	t.Run("terms belong to a year", func(t *testing.T) {
		termID := createdID(t, do(t, h, "POST", "/v1/academics/years/"+yearID+"/terms",
			owner.token, owner.slug,
			map[string]any{"name": "First term", "starts_on": "2026-01-01", "ends_on": "2026-04-30"}))
		if rec := do(t, h, "PUT", "/v1/academics/terms/"+termID+"/current", owner.token, owner.slug,
			nil); rec.Code != http.StatusOK {
			t.Fatalf("make current: got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "GET", "/v1/academics/years", owner.token, owner.slug, nil)
		var out struct {
			Terms []struct {
				Name      string `json:"name"`
				IsCurrent bool   `json:"is_current"`
			} `json:"terms"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Terms) != 1 || !out.Terms[0].IsCurrent {
			t.Fatalf("got %s", rec.Body)
		}
	})

	classID := createdID(t, do(t, h, "POST", "/v1/academics/classes", owner.token, owner.slug,
		map[string]any{"name": "Class Six", "rank": 6}))
	sectionID := createdID(t, do(t, h, "POST", "/v1/academics/classes/"+classID+"/sections",
		owner.token, owner.slug, map[string]any{"name": "A", "capacity": 40}))

	t.Run("the same class name twice is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/academics/classes", owner.token, owner.slug,
			map[string]any{"name": "Class Six"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a student sits in one section, and moving them does not seat them twice", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/academics/sections/"+sectionID+"/roll",
			owner.token, owner.slug,
			map[string]any{"user_id": student.userID, "roll_no": 1}); rec.Code != http.StatusCreated {
			t.Fatalf("place: got %d: %s", rec.Code, rec.Body)
		}
		other := createdID(t, do(t, h, "POST", "/v1/academics/classes/"+classID+"/sections",
			owner.token, owner.slug, map[string]any{"name": "B"}))
		if rec := do(t, h, "POST", "/v1/academics/sections/"+other+"/roll", owner.token, owner.slug,
			map[string]any{"user_id": student.userID, "roll_no": 3}); rec.Code != http.StatusCreated {
			t.Fatalf("move: got %d: %s", rec.Code, rec.Body)
		}

		first := roll(t, h, owner, sectionID)
		second := roll(t, h, owner, other)
		if len(first) != 0 || len(second) != 1 {
			t.Fatalf("the student is seated in %d and %d sections", len(first), len(second))
		}
		if second[0].RollNo == nil || *second[0].RollNo != 3 {
			t.Fatalf("roll number is %v", second[0].RollNo)
		}
	})

	t.Run("two students cannot share a roll number", func(t *testing.T) {
		classmate := enrollIn(t, h, ch, store, owner.slug, "student")
		if rec := do(t, h, "POST", "/v1/academics/sections/"+sectionID+"/roll", owner.token, owner.slug,
			map[string]any{"user_id": classmate.userID, "roll_no": 7}); rec.Code != http.StatusCreated {
			t.Fatalf("place: got %d: %s", rec.Code, rec.Body)
		}
		another := enrollIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/academics/sections/"+sectionID+"/roll", owner.token, owner.slug,
			map[string]any{"user_id": another.userID, "roll_no": 7})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("subjects hang off a class or the whole school", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/academics/subjects", owner.token, owner.slug,
			map[string]any{"name": "Fiqh", "class_id": classID}); rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "POST", "/v1/academics/subjects", owner.token, owner.slug,
			map[string]any{"name": "Assembly"}); rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "POST", "/v1/academics/subjects", owner.token, owner.slug,
			map[string]any{"name": "Assembly"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("the same school-wide subject twice: got %d, want 409: %s", rec.Code, rec.Body)
		}
	})
}

type rollRow struct {
	Placement struct {
		ID     string `json:"id"`
		RollNo *int32 `json:"roll_no"`
	} `json:"placement"`
	FullName string `json:"full_name"`
	RollNo   *int32 `json:"-"`
}

func roll(t *testing.T, h http.Handler, a actor, sectionID string) []rollRow {
	t.Helper()
	rec := do(t, h, "GET", "/v1/academics/sections/"+sectionID+"/roll", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("roll: got %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Students []rollRow `json:"students"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i := range out.Students {
		out.Students[i].RollNo = out.Students[i].Placement.RollNo
	}
	return out.Students
}
