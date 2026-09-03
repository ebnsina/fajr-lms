package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestPaymentPlans walks a course paid off in three parts.
func TestPaymentPlans(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	courseID := paidCourse(t, h, owner, 150000)

	if rec := do(t, h, "PATCH", "/v1/courses/"+courseID, owner.token, owner.slug,
		map[string]any{"installments": 3, "installment_gap_days": 30}); rec.Code != http.StatusOK {
		t.Fatalf("set the plan: got %d: %s", rec.Code, rec.Body)
	}

	pay := func(t *testing.T, wantAmount int64) {
		t.Helper()
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug,
			map[string]any{"in_parts": true})
		if rec.Code != http.StatusCreated {
			t.Fatalf("order: got %d: %s", rec.Code, rec.Body)
		}
		var order struct {
			ID          string `json:"id"`
			AmountMinor int64  `json:"amount_minor"`
			PartNo      *int16 `json:"part_no"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if order.AmountMinor != wantAmount {
			t.Fatalf("this part costs %d, want %d", order.AmountMinor, wantAmount)
		}
		if order.PartNo == nil {
			t.Fatal("the order is not tied to a part of the plan")
		}
		if rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/review", owner.token, owner.slug,
			map[string]any{"decision": "approve"}); rec.Code != http.StatusOK {
			t.Fatalf("approve: got %d: %s", rec.Code, rec.Body)
		}
	}

	t.Run("the first part enrolls the learner", func(t *testing.T) {
		pay(t, 50000)
		rec := do(t, h, "GET", "/v1/enrollments", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("enrollments: got %d: %s", rec.Code, rec.Body)
		}
	})

	t.Run("the plan says what is left", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/plans", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Plans []struct {
				PaymentPlan struct {
					Parts      int16 `json:"parts"`
					PaidParts  int16 `json:"paid_parts"`
					TotalMinor int64 `json:"total_minor"`
				} `json:"payment_plan"`
			} `json:"plans"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Plans) != 1 || out.Plans[0].PaymentPlan.PaidParts != 1 ||
			out.Plans[0].PaymentPlan.TotalMinor != 150000 {
			t.Fatalf("got %+v, want one plan with one part paid of 150000", out.Plans)
		}
	})

	t.Run("the rest is paid off and the plan closes", func(t *testing.T) {
		pay(t, 50000)
		pay(t, 50000)

		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug,
			map[string]any{"in_parts": true})
		if rec.Code != http.StatusConflict {
			t.Fatalf("paying a finished plan: got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a course sold in full has no plan", func(t *testing.T) {
		other := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug, map[string]any{
			"title": "Recitation in full", "visibility": "public", "price_minor": 90000,
		}))
		if rec := do(t, h, "PUT", "/v1/courses/"+other+"/status", owner.token, owner.slug,
			map[string]any{"status": "published"}); rec.Code != http.StatusOK {
			t.Fatalf("publish: got %d: %s", rec.Code, rec.Body)
		}
		buyer := enrollIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/courses/"+other+"/orders", buyer.token, owner.slug,
			map[string]any{"in_parts": true})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("only staff who handle money see everybody's plans", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/plans/all", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "GET", "/v1/plans/all", owner.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
	})
}
