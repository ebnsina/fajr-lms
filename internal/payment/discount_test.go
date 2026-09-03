package payment

import (
	"testing"
	"time"
)

func TestApply(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	before := now.Add(-24 * time.Hour)
	after := now.Add(24 * time.Hour)
	live := Discount{Kind: "percent", Value: 25, Active: true}

	t.Run("a percentage comes off", func(t *testing.T) {
		charge, off, err := Apply(live, "course", 200000, now)
		if err != nil || charge != 150000 || off != 50000 {
			t.Fatalf("charge %d off %d err %v", charge, off, err)
		}
	})

	t.Run("a fixed amount comes off", func(t *testing.T) {
		charge, off, err := Apply(Discount{Kind: "amount", Value: 30000, Active: true}, "c", 200000, now)
		if err != nil || charge != 170000 || off != 30000 {
			t.Fatalf("charge %d off %d err %v", charge, off, err)
		}
	})

	t.Run("a discount never makes a paid course free", func(t *testing.T) {
		charge, off, err := Apply(Discount{Kind: "amount", Value: 999999, Active: true}, "c", 5000, now)
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if charge != 1 || off != 4999 {
			t.Fatalf("charge %d off %d, want one unit left to pay", charge, off)
		}
	})

	t.Run("a window is honoured", func(t *testing.T) {
		early := live
		early.StartsAt = &after
		if _, _, err := Apply(early, "c", 1000, now); err != ErrCouponNotStarted {
			t.Errorf("before the window: %v", err)
		}
		done := live
		done.EndsAt = &before
		if _, _, err := Apply(done, "c", 1000, now); err != ErrCouponExpired {
			t.Errorf("after the window: %v", err)
		}
	})

	t.Run("a cap is honoured", func(t *testing.T) {
		limit := int32(2)
		used := live
		used.MaxRedemptions, used.Redeemed = &limit, 2
		if _, _, err := Apply(used, "c", 1000, now); err != ErrCouponUsedUp {
			t.Errorf("used up: %v", err)
		}
	})

	t.Run("a code for one course does not work on another", func(t *testing.T) {
		tied := live
		tied.CourseID = "maths"
		if _, _, err := Apply(tied, "arabic", 1000, now); err != ErrCouponWrongCourse {
			t.Errorf("wrong course: %v", err)
		}
		if _, _, err := Apply(tied, "maths", 1000, now); err != nil {
			t.Errorf("right course: %v", err)
		}
	})

	t.Run("a code switched off does nothing", func(t *testing.T) {
		off := live
		off.Active = false
		if _, _, err := Apply(off, "c", 1000, now); err != ErrCouponNotActive {
			t.Errorf("inactive: %v", err)
		}
	})
}
