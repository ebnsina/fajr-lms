package payment

import (
	"errors"
	"time"
)

// Discount describes a coupon well enough to price an order, without the
// database types coming along.
type Discount struct {
	Kind           string // "percent" or "amount"
	Value          int64
	CourseID       string // empty means every course
	MaxRedemptions *int32
	Redeemed       int32
	StartsAt       *time.Time
	EndsAt         *time.Time
	Active         bool
}

var (
	ErrCouponNotActive   = errors.New("payment: this code is not in use")
	ErrCouponNotStarted  = errors.New("payment: this code is not in use yet")
	ErrCouponExpired     = errors.New("payment: this code has expired")
	ErrCouponUsedUp      = errors.New("payment: this code has been used as many times as it allows")
	ErrCouponWrongCourse = errors.New("payment: this code is for a different course")
)

// Apply returns what to charge and what came off. A discount never takes the
// price below zero, and never below one unit either: a paid course that becomes
// free would leave an order nobody can pay.
func Apply(d Discount, courseID string, priceMinor int64, now time.Time) (charge, off int64, err error) {
	switch {
	case !d.Active:
		return 0, 0, ErrCouponNotActive
	case d.StartsAt != nil && now.Before(*d.StartsAt):
		return 0, 0, ErrCouponNotStarted
	case d.EndsAt != nil && !now.Before(*d.EndsAt):
		return 0, 0, ErrCouponExpired
	case d.MaxRedemptions != nil && d.Redeemed >= *d.MaxRedemptions:
		return 0, 0, ErrCouponUsedUp
	case d.CourseID != "" && d.CourseID != courseID:
		return 0, 0, ErrCouponWrongCourse
	}

	switch d.Kind {
	case "percent":
		off = priceMinor * d.Value / 100
	case "amount":
		off = d.Value
	default:
		return 0, 0, ErrCouponNotActive
	}

	if off >= priceMinor {
		off = priceMinor - 1
	}
	if off < 0 {
		off = 0
	}
	return priceMinor - off, off, nil
}
