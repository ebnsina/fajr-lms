package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/notify"
)

// Record writes the inbox entry and its deliveries in one transaction, so a
// learner never sees an announcement that was never queued, or the reverse.
func (s *Server) Record(ctx context.Context, e notify.Event, targets []notify.Target) error {
	tenantID, err := uuid.Parse(e.TenantID)
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(e.Recipient.UserID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}

	body := e.Body
	if len(body) > 2000 {
		body = body[:2000]
	}

	return s.store.InTenant(ctx, tenantID, func(q *database.Queries) error {
		notification, err := q.CreateNotification(ctx, database.CreateNotificationParams{
			TenantID: tenantID, UserID: userID, Kind: e.Kind,
			Title: e.Title, Body: body, Data: data,
		})
		if err != nil {
			return err
		}
		for _, target := range targets {
			if _, err := q.QueueDelivery(ctx, database.QueueDeliveryParams{
				NotificationID: notification.ID, TenantID: tenantID,
				Channel: target.Channel, Destination: target.Destination,
				Body: e.Title + "\n" + body,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// Claim and Settle let the dispatcher drain the queue across every tenant.
func (s *Server) Claim(ctx context.Context, limit int32) ([]notify.Delivery, error) {
	rows, err := s.store.Unscoped().ClaimDeliveries(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]notify.Delivery, 0, len(rows))
	for _, row := range rows {
		out = append(out, notify.Delivery{
			ID: row.ID.String(), Channel: row.Channel, Destination: row.Destination,
			Body: row.Body, Attempts: row.Attempts,
		})
	}
	return out, nil
}

func (s *Server) Settle(ctx context.Context, id, state, failure string, wait time.Duration) error {
	deliveryID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return s.store.Unscoped().SettleDelivery(ctx, database.SettleDeliveryParams{
		DeliveryID: deliveryID, NewState: database.DeliveryState(state),
		Failure: failure, Backoff: interval(wait),
	})
}

func interval(d time.Duration) pgtype.Interval {
	return pgtype.Interval{Microseconds: d.Microseconds(), Valid: true}
}

func (s *Server) inbox(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	userID := Authenticated(r.Context()).UserID

	var (
		items  []database.Notification
		unread int64
	)
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		if items, err = q.ListInbox(r.Context(), database.ListInboxParams{
			UserID: userID, PageLimit: limit, PageOffset: offset,
		}); err != nil {
			return err
		}
		unread, err = q.CountUnread(r.Context(), userID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"notifications": items, "unread": unread})
}

func (s *Server) markNotificationRead(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	userID := Authenticated(r.Context()).UserID

	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.MarkRead(r.Context(), database.MarkReadParams{ID: id, UserID: userID})
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	return httpx.NoContent(w)
}

func (s *Server) markAllRead(w http.ResponseWriter, r *http.Request) error {
	var rows int64
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.MarkAllRead(r.Context(), Authenticated(r.Context()).UserID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"marked": rows})
}

// notifyUser looks up how to reach someone, then hands the event to the service.
func (s *Server) notifyUser(ctx context.Context, tenantID, userID uuid.UUID, kind, title, body string, data map[string]any) {
	if s.notifier == nil {
		return
	}
	user, err := s.store.Unscoped().GetUserContact(ctx, userID)
	if err != nil {
		return
	}
	s.notifier.Notify(ctx, notify.Event{
		TenantID: tenantID.String(), Kind: kind, Title: title, Body: body, Data: data,
		Recipient: notify.Recipient{
			UserID: userID.String(), Phone: deref(user.Phone), Email: deref(user.Email),
		},
	})
}
