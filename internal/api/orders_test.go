package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/payment"
)

func testPayments(t *testing.T) *payment.Registry {
	t.Helper()
	r, err := payment.NewRegistry("bank_transfer", payment.BankTransfer{
		AccountName: "Darul Uloom", AccountNumber: "1234567890", BankName: "Islami Bank",
	})
	if err != nil {
		t.Fatalf("build payment registry: %v", err)
	}
	return r
}

// paidCourse publishes a course with a price, ready to be bought.
func paidCourse(t *testing.T, h http.Handler, a actor, priceMinor int64) string {
	t.Helper()
	id := createdID(t, do(t, h, "POST", "/v1/courses", a.token, a.slug, map[string]any{
		"title": "Advanced Tajweed", "visibility": "public", "price_minor": priceMinor, "currency": "BDT",
	}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+id+"/modules", a.token, a.slug,
		map[string]any{"title": "Unit 1"}))
	lessonID := createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", a.token, a.slug,
		map[string]any{"title": "Lesson", "kind": "video"}))
	if rec := do(t, h, "PATCH", "/v1/lessons/"+lessonID, a.token, a.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish lesson: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/v1/courses/"+id+"/status", a.token, a.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish course: got %d: %s", rec.Code, rec.Body)
	}
	return id
}

type orderBody struct {
	ID          string              `json:"id"`
	Status      string              `json:"status"`
	Reference   string              `json:"reference"`
	AmountMinor int64               `json:"amount_minor"`
	Currency    string              `json:"currency"`
	Instruction payment.Instruction `json:"instruction"`
}

func TestManualPaymentFlow(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	courseID := paidCourse(t, h, owner, 150000)

	var order orderBody

	t.Run("buying returns bank details and a reference", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if order.Status != "pending" || order.AmountMinor != 150000 || order.Currency != "BDT" {
			t.Fatalf("got %+v", order)
		}
		if order.Instruction.Kind != payment.InstructManual || !order.Instruction.NeedsProof {
			t.Fatalf("got %+v", order.Instruction)
		}
		if order.Instruction.Fields["account_number"] != "1234567890" {
			t.Errorf("bank details missing: %+v", order.Instruction.Fields)
		}
		if order.Instruction.Reference != order.Reference {
			t.Errorf("slip reference %q does not match the order %q", order.Instruction.Reference, order.Reference)
		}
	})

	t.Run("ordering twice returns the order already in flight", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var again orderBody
		if err := json.Unmarshal(rec.Body.Bytes(), &again); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if again.ID != order.ID {
			t.Errorf("a second tap created order %s, want the existing %s", again.ID, order.ID)
		}
	})

	t.Run("paying for a free course is refused", func(t *testing.T) {
		freeID, _ := publishedCourse(t, h, owner, 1)
		rec := do(t, h, "POST", "/v1/courses/"+freeID+"/orders", student.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("an unpublished course cannot be bought", func(t *testing.T) {
		draftID := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
			map[string]any{"title": "Draft", "price_minor": 500}))
		rec := do(t, h, "POST", "/v1/courses/"+draftID+"/orders", student.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("proof needs a slip or a transaction id", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/proof", student.token, owner.slug, map[string]any{})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("submitting a slip moves it into the queue", func(t *testing.T) {
		slipID := createdID(t, do(t, h, "POST", "/v1/media", owner.token, owner.slug,
			map[string]any{"url": "https://youtu.be/dQw4w9WgXcQ", "kind": "link"}))

		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/proof", student.token, owner.slug, map[string]any{
			"media_id": slipID, "provider_ref": "TXN99881", "note": "bKash থেকে পাঠানো হয়েছে",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body)
		}
		var got orderBody
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Status != "awaiting_review" {
			t.Errorf("got %q, want awaiting_review", got.Status)
		}
	})

	t.Run("another learner cannot submit proof on this order", func(t *testing.T) {
		outsider := enrolIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/proof", outsider.token, owner.slug,
			map[string]any{"provider_ref": "TXN00000"})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("only staff see the review queue", func(t *testing.T) {
		if rec := do(t, h, "GET", "/v1/orders/review", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("student: got %d, want 403", rec.Code)
		}
		rec := do(t, h, "GET", "/v1/orders/review", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Orders []struct {
				FullName string `json:"full_name"`
				Title    string `json:"title"`
			} `json:"orders"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Orders) != 1 || got.Orders[0].Title != "Advanced Tajweed" {
			t.Errorf("got %+v", got.Orders)
		}
	})

	t.Run("a learner cannot approve their own payment", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/review", student.token, owner.slug,
			map[string]any{"decision": "approve"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("rejects an unknown decision", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/review", owner.token, owner.slug,
			map[string]any{"decision": "maybe"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("approval pays the order and enrols the learner", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/review", owner.token, owner.slug,
			map[string]any{"decision": "approve", "note": "slip verified"})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body)
		}
		var got struct {
			Status string  `json:"status"`
			PaidAt *string `json:"paid_at"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Status != "paid" || got.PaidAt == nil {
			t.Fatalf("got %+v", got)
		}

		rec = do(t, h, "GET", "/v1/courses/"+courseID+"/progress", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("learner was not enrolled after payment: got %d: %s", rec.Code, rec.Body)
		}
		var progress struct {
			Enrollment struct {
				Source string `json:"source"`
			} `json:"enrollment"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &progress); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if progress.Enrollment.Source != "purchase" {
			t.Errorf("enrolment source = %q, want purchase", progress.Enrollment.Source)
		}
	})

	t.Run("reviewing twice is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/review", owner.token, owner.slug,
			map[string]any{"decision": "reject"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a settled order cannot be cancelled or re-proofed", func(t *testing.T) {
		if rec := do(t, h, "DELETE", "/v1/orders/"+order.ID, student.token, owner.slug, nil); rec.Code != http.StatusConflict {
			t.Errorf("cancel: got %d, want 409", rec.Code)
		}
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/proof", student.token, owner.slug,
			map[string]any{"provider_ref": "TXN2"})
		if rec.Code != http.StatusConflict {
			t.Errorf("proof: got %d, want 409", rec.Code)
		}
	})

	t.Run("a rejected order lets the learner try again", func(t *testing.T) {
		buyer := enrolIn(t, h, ch, store, owner.slug, "student")
		first := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/orders", buyer.token, owner.slug, nil))
		if rec := do(t, h, "POST", "/v1/orders/"+first+"/proof", buyer.token, owner.slug,
			map[string]any{"provider_ref": "WRONG"}); rec.Code != http.StatusOK {
			t.Fatalf("proof: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "POST", "/v1/orders/"+first+"/review", owner.token, owner.slug,
			map[string]any{"decision": "reject", "note": "amount did not match"}); rec.Code != http.StatusOK {
			t.Fatalf("reject: got %d: %s", rec.Code, rec.Body)
		}
		second := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/orders", buyer.token, owner.slug, nil))
		if second == first {
			t.Error("a rejected order should not be reused")
		}
	})

	t.Run("my orders list what I bought", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/orders", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Orders []struct {
				Title string `json:"title"`
			} `json:"orders"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Orders) != 1 {
			t.Errorf("got %d orders, want 1", len(got.Orders))
		}
	})

	t.Run("an order in another tenant is not found", func(t *testing.T) {
		other := enrol(t, h, ch, store, "owner")
		rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/review", other.token, other.slug,
			map[string]any{"decision": "approve"})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})
}
