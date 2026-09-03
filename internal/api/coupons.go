package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/payment"
)

type couponRequest struct {
	Code           string     `json:"code"`
	Kind           string     `json:"kind"`
	Value          int64      `json:"value"`
	CourseID       string     `json:"course_id"`
	MaxRedemptions *int32     `json:"max_redemptions"`
	StartsAt       *time.Time `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
}

func (s *Server) createCoupon(w http.ResponseWriter, r *http.Request) error {
	var body couponRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	code := strings.ToUpper(strings.TrimSpace(body.Code))
	if len(code) < 3 || len(code) > 32 {
		return invalid("code", "A code is between 3 and 32 letters, numbers or hyphens.")
	}
	kind := database.DiscountKind(strings.TrimSpace(body.Kind))
	switch kind {
	case database.DiscountKindPercent:
		if body.Value < 1 || body.Value > 100 {
			return invalid("value", "A percentage is between 1 and 100.")
		}
	case database.DiscountKindAmount:
		if body.Value < 1 {
			return invalid("value", "Take off at least one unit.")
		}
	default:
		return invalid("kind", "Choose a percentage or an amount.")
	}
	if body.MaxRedemptions != nil && *body.MaxRedemptions < 1 {
		return invalid("max_redemptions", "Allow at least one use, or leave it open.")
	}
	if body.StartsAt != nil && body.EndsAt != nil && !body.EndsAt.After(*body.StartsAt) {
		return invalid("ends_at", "It has to end after it starts.")
	}

	params := database.CreateCouponParams{
		TenantID: CurrentTenant(r.Context()).ID, Code: code, Kind: kind, Value: body.Value,
		MaxRedemptions: body.MaxRedemptions,
		CreatedBy:      uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
	}
	if raw := strings.TrimSpace(body.CourseID); raw != "" {
		courseID, err := uuid.Parse(raw)
		if err != nil {
			return invalid("course_id", "Name the course this code is for, or leave it out.")
		}
		params.CourseID = uuid.NullUUID{UUID: courseID, Valid: true}
	}
	if body.StartsAt != nil {
		params.StartsAt = pgtype.Timestamptz{Time: *body.StartsAt, Valid: true}
	}
	if body.EndsAt != nil {
		params.EndsAt = pgtype.Timestamptz{Time: *body.EndsAt, Valid: true}
	}

	var coupon database.Coupon
	err := s.store.InTenant(r.Context(), params.TenantID, func(q *database.Queries) error {
		var err error
		coupon, err = q.CreateCoupon(r.Context(), params)
		return err
	})
	if isUniqueViolation(err) {
		return &httpx.Error{Status: http.StatusConflict, Code: "code_taken",
			Message: "This school already has a code with that name.", Field: "code"}
	}
	if isForeignKeyViolation(err) {
		return invalid("course_id", "That course is not in this school.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, coupon)
}

func (s *Server) listCoupons(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var coupons []database.Coupon
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		coupons, err = q.ListCoupons(r.Context(), database.ListCouponsParams{
			PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"coupons": coupons})
}

func (s *Server) setCouponActive(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body struct {
		Active bool `json:"active"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	var coupon database.Coupon
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		coupon, err = q.SetCouponActive(r.Context(), database.SetCouponActiveParams{
			ID: id, Active: body.Active,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, coupon)
}

func (s *Server) deleteCoupon(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.DeleteCoupon(r.Context(), id)
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// discountFor prices a code against a course, or says why it does not apply.
// The wording is the learner's, not the accountant's.
func discountFor(coupon database.Coupon, courseID uuid.UUID, price int64) (charge, off int64, err error) {
	d := payment.Discount{
		Kind: string(coupon.Kind), Value: coupon.Value, MaxRedemptions: coupon.MaxRedemptions,
		Redeemed: coupon.Redeemed, Active: coupon.Active,
	}
	if coupon.CourseID.Valid {
		d.CourseID = coupon.CourseID.UUID.String()
	}
	if coupon.StartsAt.Valid {
		d.StartsAt = &coupon.StartsAt.Time
	}
	if coupon.EndsAt.Valid {
		d.EndsAt = &coupon.EndsAt.Time
	}

	charge, off, err = payment.Apply(d, courseID.String(), price, time.Now())
	if err != nil {
		return 0, 0, &httpx.Error{Status: http.StatusConflict, Code: "coupon_not_usable",
			Message: capitalise(strings.TrimPrefix(err.Error(), "payment: ")) + ".", Field: "coupon"}
	}
	return charge, off, nil
}
