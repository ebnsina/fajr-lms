package api_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
)

type memberRow struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
}

func members(t *testing.T, h http.Handler, a actor) []memberRow {
	t.Helper()
	rec := do(t, h, "GET", "/v1/tenant/members", a.token, a.slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list members: got %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Members []memberRow `json:"members"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out.Members
}

func TestInviteMembers(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	phone := fmt.Sprintf("+8801%09d", rand.IntN(1_000_000_000))
	rec := do(t, h, "POST", "/v1/tenant/members", owner.token, owner.slug, map[string]any{
		"full_name": "Fatima Rahman", "destination": phone, "role": "instructor",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("the invited person is in the school", func(t *testing.T) {
		for _, row := range members(t, h, owner) {
			if row.FullName == "Fatima Rahman" && row.Role == "instructor" {
				return
			}
		}
		t.Fatal("the invited teacher is not in the list")
	})

	t.Run("somebody already here cannot be invited twice", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/tenant/members", owner.token, owner.slug, map[string]any{
			"full_name": "Fatima Again", "destination": phone, "role": "assistant",
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a student cannot invite anybody", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/tenant/members", student.token, owner.slug, map[string]any{
			"full_name": "A Friend", "destination": "+8801700000000", "role": "owner",
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a destination that is neither phone nor email is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/tenant/members", owner.token, owner.slug, map[string]any{
			"full_name": "Nobody", "destination": "not-a-contact", "role": "student",
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a role can be changed and the person removed", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/tenant/members/"+student.userID.String()+"/role",
			owner.token, owner.slug, map[string]any{"role": "assistant"})
		if rec.Code != http.StatusOK {
			t.Fatalf("set role: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "DELETE", "/v1/tenant/members/"+student.userID.String(),
			owner.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("remove: got %d: %s", rec.Code, rec.Body)
		}
		for _, row := range members(t, h, owner) {
			if row.UserID == student.userID.String() {
				t.Fatal("the removed member is still listed")
			}
		}
	})

	t.Run("the last owner keeps the school", func(t *testing.T) {
		rec := do(t, h, "PUT", "/v1/tenant/members/"+owner.userID.String()+"/role",
			owner.token, owner.slug, map[string]any{"role": "admin"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
		rec = do(t, h, "DELETE", "/v1/tenant/members/"+owner.userID.String(),
			owner.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("removing yourself: got %d, want 409: %s", rec.Code, rec.Body)
		}
	})
}
