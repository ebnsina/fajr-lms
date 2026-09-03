package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/payment"
)

// redeem counts a use of the code once the money is actually in. The cap is
// checked in the same statement, so two payments landing together cannot push
// a code past its limit.
func redeem(ctx context.Context, q *database.Queries, coupon uuid.NullUUID) error {
	if !coupon.Valid {
		return nil
	}
	_, err := q.RedeemCoupon(ctx, coupon.UUID)
	return err
}

func (s *Server) paymentProviders(w http.ResponseWriter, r *http.Request) error {
	return httpx.JSON(w, http.StatusOK, map[string]any{"providers": s.payments.Capabilities()})
}

type createOrderRequest struct {
	Provider string `json:"provider"`
	Coupon   string `json:"coupon"`
}

type orderResponse struct {
	database.Order
	Instruction payment.Instruction `json:"instruction"`
}

// createOrder starts a purchase, or returns the order already in flight so a
// double tap on a flaky connection does not create two.
func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) error {
	courseID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body createOrderRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	provider, err := s.payments.Get(body.Provider)
	if errors.Is(err, payment.ErrUnknownProvider) {
		return invalid("provider", "No payment method by that name is available.")
	}
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	session := Authenticated(r.Context())

	var order database.Order
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		course, err := q.GetCourse(r.Context(), courseID)
		if err != nil {
			return err
		}
		if course.Status != database.PublishStatusPublished {
			return httpx.Errorf(http.StatusConflict, "course_not_published", "This course is not on sale.")
		}
		if course.PriceMinor <= 0 {
			return httpx.Errorf(http.StatusConflict, "course_is_free", "This course is free; enroll directly.")
		}

		// A code is priced here rather than at checkout, so what the learner is
		// told to pay is what the record says.
		charge, off := course.PriceMinor, int64(0)
		var coupon uuid.NullUUID
		if code := strings.ToUpper(strings.TrimSpace(body.Coupon)); code != "" {
			found, err := q.CouponByCode(r.Context(), code)
			if database.IsNotFound(err) {
				return &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_coupon",
					Message: "No code by that name.", Field: "coupon"}
			}
			if err != nil {
				return err
			}
			if charge, off, err = discountFor(found, courseID, course.PriceMinor); err != nil {
				return err
			}
			coupon = uuid.NullUUID{UUID: found.ID, Valid: true}
		}

		existing, err := q.OpenOrderForCourse(r.Context(), database.OpenOrderForCourseParams{
			CourseID: courseID, UserID: session.UserID,
		})
		if err == nil {
			order = existing
			return nil
		}
		if !database.IsNotFound(err) {
			return err
		}

		order, err = q.CreateOrder(r.Context(), database.CreateOrderParams{
			TenantID: tenant.ID, UserID: session.UserID, CourseID: courseID,
			Provider: provider.Caps().Name, AmountMinor: charge,
			Currency: course.Currency, Reference: payment.NewReference(),
			CouponID: coupon, ListAmountMinor: &course.PriceMinor, DiscountMinor: off,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	instruction, err := provider.Start(r.Context(), payment.Order{
		ID: order.ID.String(), TenantID: tenant.ID.String(), TenantSlug: tenant.Slug,
		Reference: order.Reference, AmountMinor: order.AmountMinor, Currency: order.Currency,
		PayerName: session.FullName,
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, orderResponse{Order: order, Instruction: instruction})
}

type proofRequest struct {
	MediaID     string `json:"media_id"`
	ProviderRef string `json:"provider_ref"`
	Note        string `json:"note"`
}

// submitProof attaches a deposit slip and moves the order into the review queue.
func (s *Server) submitProof(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body proofRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if len(body.Note) > 2000 {
		return invalid("note", "Keep the note under 2000 characters.")
	}
	if len(body.ProviderRef) > 255 {
		return invalid("provider_ref", "That transaction id is too long.")
	}

	mediaID := uuid.NullUUID{}
	if raw := strings.TrimSpace(body.MediaID); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return invalid("media_id", "Upload the slip first, then send its media id.")
		}
		mediaID = uuid.NullUUID{UUID: parsed, Valid: true}
	}
	if !mediaID.Valid && strings.TrimSpace(body.ProviderRef) == "" {
		return invalid("media_id", "Attach a slip or give the transaction id.")
	}

	session := Authenticated(r.Context())
	var order database.Order
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		current, err := q.GetOrder(r.Context(), id)
		if err != nil {
			return err
		}
		if current.UserID != session.UserID {
			return httpx.ErrNotFound
		}
		order, err = q.SubmitPaymentProof(r.Context(), database.SubmitPaymentProofParams{
			ID: id, ProofMediaID: mediaID,
			ProviderRef: strings.TrimSpace(body.ProviderRef), Note: strings.TrimSpace(body.Note),
		})
		if database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "order_not_open",
				"This order has already been reviewed.")
		}
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if isForeignKeyViolation(err) {
		return invalid("media_id", "That upload does not exist.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, order)
}

type reviewRequest struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

// reviewOrder is the approval queue: approving enrolls the learner in the same
// transaction, so payment and access can never disagree.
func (s *Server) reviewOrder(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body reviewRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	var status database.OrderStatus
	switch strings.TrimSpace(body.Decision) {
	case "approve":
		status = database.OrderStatusPaid
	case "reject":
		status = database.OrderStatusRejected
	default:
		return invalid("decision", "Decision must be approve or reject.")
	}

	tenant := CurrentTenant(r.Context())
	reviewer := Authenticated(r.Context()).UserID
	var order database.Order

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		// Look first, so an order in another tenant reads as missing rather
		// than as one that has already been reviewed.
		if _, err := q.GetOrder(r.Context(), id); err != nil {
			return err
		}

		order, err = q.SettleOrder(r.Context(), database.SettleOrderParams{
			ID: id, Status: status, ReviewedBy: uuid.NullUUID{UUID: reviewer, Valid: true},
		})
		if database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "order_not_open", "This order has already been reviewed.")
		}
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{
			"decision": body.Decision, "note": body.Note, "reviewer": reviewer.String(),
		})
		if err != nil {
			return err
		}
		if _, err := q.RecordPaymentEvent(r.Context(), database.RecordPaymentEventParams{
			TenantID: tenant.ID, OrderID: uuid.NullUUID{UUID: order.ID, Valid: true},
			Provider: order.Provider, EventID: "review:" + order.ID.String(),
			Kind: "manual_review", Payload: payload,
		}); err != nil && !database.IsNotFound(err) {
			return err
		}

		if status != database.OrderStatusPaid {
			return nil
		}
		if err := redeem(r.Context(), q, order.CouponID); err != nil {
			return err
		}
		_, err = q.EnrollUser(r.Context(), database.EnrollUserParams{
			TenantID: tenant.ID, CourseID: order.CourseID, UserID: order.UserID,
			Source: database.EnrollmentSourcePurchase,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	if status == database.OrderStatusPaid {
		s.notifyUser(r.Context(), tenant.ID, order.UserID, "payment.approved",
			"Payment confirmed", "Your payment was approved and you are now enrolled.",
			map[string]any{"order_id": order.ID, "reference": order.Reference})
	} else {
		s.notifyUser(r.Context(), tenant.ID, order.UserID, "payment.rejected",
			"Payment not accepted", "Your payment could not be verified. Please contact the office.",
			map[string]any{"order_id": order.ID, "reference": order.Reference})
	}
	return httpx.JSON(w, http.StatusOK, order)
}

func (s *Server) listReviewQueue(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var rows []database.ListOrdersForReviewRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListOrdersForReview(r.Context(), database.ListOrdersForReviewParams{
			PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"orders": rows})
}

// listPaidOrders is the record of money taken, which is where a refund starts.
func (s *Server) listPaidOrders(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var rows []database.ListPaidOrdersRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListPaidOrders(r.Context(), database.ListPaidOrdersParams{
			PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"orders": rows})
}

func (s *Server) listMyOrders(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	var rows []database.ListMyOrdersRow
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListMyOrders(r.Context(), database.ListMyOrdersParams{
			UserID: Authenticated(r.Context()).UserID, PageLimit: limit, PageOffset: offset,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"orders": rows})
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	session := Authenticated(r.Context())

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		current, err := q.GetOrder(r.Context(), id)
		if err != nil {
			return err
		}
		if current.UserID != session.UserID && !staffRole(CurrentRole(r.Context())) {
			return httpx.ErrForbidden
		}
		if _, err := q.CancelOrder(r.Context(), id); database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "order_not_open", "This order is already settled.")
		} else if err != nil {
			return err
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.NoContent(w)
}

type refundRequest struct {
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason"`
	KeepAccess  bool   `json:"keep_access"`
}

// refundOrder hands money back on the books. The gateway or the bank moves the
// actual money; this records what was returned and closes the access it bought.
func (s *Server) refundOrder(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body refundRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.AmountMinor < 0 {
		return invalid("amount_minor", "A refund cannot be negative.")
	}
	reason := strings.TrimSpace(body.Reason)
	if len(reason) > 500 {
		return invalid("reason", "Keep the reason under 500 characters.")
	}

	tenant := CurrentTenant(r.Context())
	staff := Authenticated(r.Context()).UserID
	var order database.Order

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		existing, err := q.GetOrder(r.Context(), id)
		if err != nil {
			return err
		}
		if existing.Status != database.OrderStatusPaid && existing.Status != database.OrderStatusRefunded {
			return httpx.Errorf(http.StatusConflict, "order_not_paid",
				"Only a paid order can be refunded.")
		}

		// Left empty, a refund is for whatever is still outstanding.
		amount := body.AmountMinor
		if amount == 0 {
			amount = existing.AmountMinor - existing.RefundedMinor
		}
		if amount <= 0 {
			return httpx.Errorf(http.StatusConflict, "already_refunded",
				"This order has already been refunded in full.")
		}

		order, err = q.RefundOrder(r.Context(), database.RefundOrderParams{
			ID: id, AmountMinor: amount, Reason: reason,
			RefundedBy: uuid.NullUUID{UUID: staff, Valid: true},
		})
		if database.IsNotFound(err) {
			return httpx.Errorf(http.StatusConflict, "refund_too_large",
				"That is more than is left on this order.")
		}
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{
			"amount_minor": amount, "reason": reason, "staff": staff.String(),
		})
		if err != nil {
			return err
		}
		if _, err := q.RecordPaymentEvent(r.Context(), database.RecordPaymentEventParams{
			TenantID: tenant.ID, OrderID: uuid.NullUUID{UUID: order.ID, Valid: true},
			Provider: order.Provider,
			EventID:  fmt.Sprintf("refund:%s:%d", order.ID, order.RefundedMinor),
			Kind:     "refund", Payload: payload,
		}); err != nil && !database.IsNotFound(err) {
			return err
		}

		if order.Status != database.OrderStatusRefunded || body.KeepAccess {
			return nil
		}
		enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
			CourseID: order.CourseID, UserID: order.UserID,
		})
		if database.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = q.CancelEnrollment(r.Context(), enrollment.ID)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	s.notifyUser(r.Context(), tenant.ID, order.UserID, "payment.refunded",
		"Refund recorded", "A refund has been recorded against your payment.",
		map[string]any{"order_id": order.ID, "reference": order.Reference})
	return httpx.JSON(w, http.StatusOK, order)
}
