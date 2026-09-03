package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/payment"
)

// paymentCallback receives a gateway notification. It is unauthenticated by
// necessity, so nothing in the request body is trusted: the order is found by
// reference and the provider re-checks the payment with the gateway itself.
func (s *Server) paymentCallback(w http.ResponseWriter, r *http.Request) error {
	tenantSlug := r.PathValue("tenant")
	providerName := r.PathValue("provider")

	provider, err := s.payments.Get(providerName)
	if errors.Is(err, payment.ErrUnknownProvider) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	raw, err := callbackFields(r)
	if err != nil {
		return err
	}
	reference := strings.TrimSpace(firstString(raw, "tran_id", "reference", "merchantInvoiceNumber"))
	if reference == "" {
		return invalid("tran_id", "The callback carries no order reference.")
	}

	tenant, err := s.identity.ResolveTenant(r.Context(), tenantSlug)
	if errors.Is(err, identity.ErrNoMembership) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	eventID := firstString(raw, "val_id", "event_id", "paymentID", "bank_tran_id")
	if eventID == "" {
		eventID = reference + ":" + firstString(raw, "status")
	}
	// The kind column requires a value; not every gateway sends a status field.
	kind := firstString(raw, "status", "kind")
	if kind == "" {
		kind = "callback"
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	var settled bool
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		order, err := q.GetOrderByReference(r.Context(), reference)
		if err != nil {
			return err
		}

		// A duplicate event id means the gateway is retrying; record once and stop.
		_, err = q.RecordPaymentEvent(r.Context(), database.RecordPaymentEventParams{
			TenantID: tenant.ID, OrderID: uuid.NullUUID{UUID: order.ID, Valid: true},
			Provider: providerName, EventID: eventID,
			Kind: kind, Payload: payload,
		})
		if database.IsNotFound(err) {
			slog.InfoContext(r.Context(), "duplicate payment callback ignored",
				"provider", providerName, "event_id", eventID, "order", order.ID)
			return nil
		}
		if err != nil {
			return err
		}

		if order.Status == database.OrderStatusPaid {
			return nil
		}

		result, err := provider.Verify(r.Context(), payment.Order{
			ID: order.ID.String(), TenantID: tenant.ID.String(), TenantSlug: tenant.Slug,
			Reference:   order.Reference,
			AmountMinor: order.AmountMinor, Currency: order.Currency,
		}, payment.Callback{EventID: eventID, Kind: kind, Raw: raw})

		if errors.Is(err, payment.ErrEventMismatch) {
			slog.ErrorContext(r.Context(), "payment callback did not match the order",
				"order", order.ID, "reference", reference, "error", err)
			return nil
		}
		if errors.Is(err, payment.ErrNotVerifiable) {
			return httpx.Errorf(http.StatusConflict, "manual_provider",
				"This payment method is settled by staff review, not by callback.")
		}
		if err != nil {
			return err
		}

		status := database.OrderStatus(result.Status)
		if status != database.OrderStatusPaid && status != database.OrderStatusRejected &&
			status != database.OrderStatusCancelled {
			return nil
		}

		if _, err := q.SettleOrder(r.Context(), database.SettleOrderParams{
			ID: order.ID, Status: status,
		}); err != nil && !database.IsNotFound(err) {
			return err
		}
		if status != database.OrderStatusPaid {
			return nil
		}

		settled = true
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

	// A GET is the payer's browser coming back from the gateway, so send them
	// somewhere readable; a POST is server-to-server and wants an acknowledgement.
	if r.Method == http.MethodGet {
		http.Redirect(w, r, s.returnURL(tenantSlug, reference, settled), http.StatusSeeOther)
		return nil
	}
	// Gateways retry anything that is not 2xx, so acknowledge once handled.
	return httpx.JSON(w, http.StatusOK, map[string]any{"received": true, "settled": settled})
}

// returnURL is where the payer lands after the gateway sends them back.
func (s *Server) returnURL(tenant, reference string, settled bool) string {
	outcome := "pending"
	if settled {
		outcome = "paid"
	}
	return s.publicURL + "/pay/" + outcome + "?tenant=" + url.QueryEscape(tenant) +
		"&reference=" + url.QueryEscape(reference)
}

// callbackFields reads a callback sent as a form, as JSON, or as query
// parameters on a browser redirect.
func callbackFields(r *http.Request) (map[string]any, error) {
	if r.Method == http.MethodGet {
		raw := make(map[string]any, len(r.URL.Query()))
		for key := range r.URL.Query() {
			raw[key] = r.URL.Query().Get(key)
		}
		return raw, nil
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var raw map[string]any
		if err := httpx.DecodeJSONLoose(r.Body, &raw); err != nil {
			return nil, invalid("body", "The callback body is not valid JSON.")
		}
		return raw, nil
	}

	if err := r.ParseForm(); err != nil {
		return nil, invalid("body", "The callback body could not be read.")
	}
	raw := make(map[string]any, len(r.Form))
	for key := range r.Form {
		raw[key] = r.Form.Get(key)
	}
	return raw, nil
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := raw[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
