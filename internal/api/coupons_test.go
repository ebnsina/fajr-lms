package api_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"
)

func couponCode() string { return fmt.Sprintf("EID%06d", rand.IntN(1_000_000)) }

func TestCoupons(t *testing.T) {
	h, ch, store := newHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")

	courseID := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
		map[string]any{"title": "Tajweed", "visibility": "public", "price_minor": 200000}))
	if rec := do(t, h, "PUT", "/v1/courses/"+courseID+"/status", owner.token, owner.slug,
		map[string]any{"status": "published"}); rec.Code != http.StatusOK {
		t.Fatalf("publish: got %d: %s", rec.Code, rec.Body)
	}

	code := couponCode()
	rec := do(t, h, "POST", "/v1/coupons", owner.token, owner.slug, map[string]any{
		"code": code, "kind": "percent", "value": 25,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create coupon: got %d: %s", rec.Code, rec.Body)
	}

	t.Run("a code takes the price down on the order", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug,
			map[string]any{"coupon": code})
		if rec.Code != http.StatusCreated {
			t.Fatalf("order: got %d: %s", rec.Code, rec.Body)
		}
		var order struct {
			AmountMinor   int64 `json:"amount_minor"`
			DiscountMinor int64 `json:"discount_minor"`
			ListAmount    int64 `json:"list_amount_minor"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if order.AmountMinor != 150000 || order.DiscountMinor != 50000 || order.ListAmount != 200000 {
			t.Fatalf("the order was not discounted: %+v", order)
		}
	})

	t.Run("a code nobody made is refused", func(t *testing.T) {
		buyer := enrollIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", buyer.token, owner.slug,
			map[string]any{"coupon": "NOSUCHCODE"})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a code that has ended is refused", func(t *testing.T) {
		past := couponCode()
		if rec := do(t, h, "POST", "/v1/coupons", owner.token, owner.slug, map[string]any{
			"code": past, "kind": "amount", "value": 5000,
			"starts_at": time.Now().Add(-48 * time.Hour), "ends_at": time.Now().Add(-time.Hour),
		}); rec.Code != http.StatusCreated {
			t.Fatalf("create: got %d: %s", rec.Code, rec.Body)
		}
		buyer := enrollIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", buyer.token, owner.slug,
			map[string]any{"coupon": past})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a code switched off stops working", func(t *testing.T) {
		var coupons struct {
			Coupons []struct {
				ID   string `json:"id"`
				Code string `json:"code"`
			} `json:"coupons"`
		}
		rec := do(t, h, "GET", "/v1/coupons", owner.token, owner.slug, nil)
		if err := json.Unmarshal(rec.Body.Bytes(), &coupons); err != nil {
			t.Fatalf("decode: %v", err)
		}
		var id string
		for _, row := range coupons.Coupons {
			if row.Code == code {
				id = row.ID
			}
		}
		if id == "" {
			t.Fatal("the code is not in the list")
		}
		if rec := do(t, h, "PUT", "/v1/coupons/"+id+"/active", owner.token, owner.slug,
			map[string]any{"active": false}); rec.Code != http.StatusOK {
			t.Fatalf("switch off: got %d: %s", rec.Code, rec.Body)
		}

		buyer := enrollIn(t, h, ch, store, owner.slug, "student")
		rec = do(t, h, "POST", "/v1/courses/"+courseID+"/orders", buyer.token, owner.slug,
			map[string]any{"coupon": code})
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("only staff who handle money touch codes", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/coupons", student.token, owner.slug, map[string]any{
			"code": couponCode(), "kind": "percent", "value": 90,
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "GET", "/v1/coupons", student.token, owner.slug, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("list: got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a percentage outside one to a hundred is refused", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/coupons", owner.token, owner.slug, map[string]any{
			"code": couponCode(), "kind": "percent", "value": 150,
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})
}
