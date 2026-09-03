package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestHifz covers the daily record a madrasah keeps: what was heard, from whom,
// and who is allowed to read it back.
func TestHifz(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	classmate := enrollIn(t, h, ch, store, owner.slug, "student")
	guardian := enrollIn(t, h, ch, store, owner.slug, "student")

	t.Run("a teacher records a sitting", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/hifz", owner.token, owner.slug, map[string]any{
			"student_id": student.userID, "kind": "sabaq",
			"from_surah": 78, "from_ayah": 1, "to_surah": 78, "to_ayah": 20,
			"quality": "excellent", "mistakes": 1, "note": "steady",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
	})

	t.Run("an ayah that does not exist is refused", func(t *testing.T) {
		// An-Naba has 40 ayahs.
		rec := do(t, h, "POST", "/v1/hifz", owner.token, owner.slug, map[string]any{
			"student_id": student.userID, "kind": "sabaq",
			"from_surah": 78, "from_ayah": 1, "to_surah": 78, "to_ayah": 99,
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a range that runs backwards is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/hifz", owner.token, owner.slug, map[string]any{
			"student_id": student.userID, "kind": "sabqi",
			"from_surah": 80, "from_ayah": 10, "to_surah": 79, "to_ayah": 5,
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a learner cannot write their own record", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/hifz", student.token, owner.slug, map[string]any{
			"student_id": student.userID, "kind": "sabaq",
			"from_surah": 1, "from_ayah": 1, "to_surah": 1, "to_ayah": 7,
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("what is memorised is counted in ayahs", func(t *testing.T) {
		// Surah 114 has 6 ayahs and 113 has 5, so the two together are 11.
		if rec := do(t, h, "POST", "/v1/hifz", owner.token, owner.slug, map[string]any{
			"student_id": student.userID, "kind": "sabaq",
			"from_surah": 113, "from_ayah": 1, "to_surah": 114, "to_ayah": 6,
		}); rec.Code != http.StatusCreated {
			t.Fatalf("record: got %d: %s", rec.Code, rec.Body)
		}

		rec := do(t, h, "GET", "/v1/hifz/students/"+student.userID.String(),
			student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("read: got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Ayahs      int64 `json:"ayahs_memorised"`
			Lessons    int64 `json:"lessons"`
			TotalAyahs int64 `json:"total_ayahs"`
			Surahs     []struct {
				Number    int16 `json:"number"`
				AyahCount int16 `json:"ayah_count"`
			} `json:"surahs"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// Twenty ayahs of An-Naba, then eleven across the last two surahs.
		if out.Ayahs != 31 || out.Lessons != 2 {
			t.Fatalf("got %d ayahs over %d lessons, want 31 over 2", out.Ayahs, out.Lessons)
		}
		if out.TotalAyahs != 6236 || len(out.Surahs) != 114 {
			t.Fatalf("the Qur'an came back as %d ayahs over %d surahs", out.TotalAyahs, len(out.Surahs))
		}
	})

	t.Run("a classmate cannot read somebody else's record", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/hifz/students/"+student.userID.String(),
			classmate.token, owner.slug, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a guardian reads their own child's record and no other", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/guardians", owner.token, owner.slug, map[string]any{
			"guardian_id": guardian.userID, "student_id": student.userID, "relation": "father",
		}); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("link guardian: got %d: %s", rec.Code, rec.Body)
		}

		if rec := do(t, h, "GET", "/v1/hifz/students/"+student.userID.String(),
			guardian.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("guardian read: got %d: %s", rec.Code, rec.Body)
		}
		rec := do(t, h, "GET", "/v1/hifz/students/"+classmate.userID.String(),
			guardian.token, owner.slug, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("another child: got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("the day's sittings come back together", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/hifz", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Entries []struct {
				StudentName string `json:"student_name"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Entries) < 2 {
			t.Fatalf("got %d sittings today", len(out.Entries))
		}
	})
}
