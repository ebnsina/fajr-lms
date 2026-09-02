package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ebnsina/fajr-lms/internal/api"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/notify"
)

// notifyHarness returns a server whose notifier is live, plus the channel the
// dispatcher will use, so a whole announcement can be followed end to end.
func notifyHarness(t *testing.T) (http.Handler, *api.Server, *captureChannel, *database.Store) {
	t.Helper()
	url := envOrSkip(t)
	store, err := database.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(store.Close)

	ch := &captureChannel{}
	server := api.NewServer(store, identity.New(store, ch), testRegistry(t), testPayments(t), "https://fajr.test")
	server.UseNotifier(notify.NewService(server, "capture"))
	return server.Routes(), server, ch, store
}

func envOrSkip(t *testing.T) string {
	t.Helper()
	url := osGetenv("FAJR_DATABASE_URL")
	if url == "" {
		t.Skip("FAJR_DATABASE_URL not set")
	}
	return url
}

func TestNotifications(t *testing.T) {
	h, server, ch, store := notifyHarness(t)
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	courseID := paidCourse(t, h, owner, 150000)

	// Buy, submit proof, approve: approval is the announcement worth sending.
	rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("order: got %d: %s", rec.Code, rec.Body)
	}
	var order orderBody
	if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/proof", student.token, owner.slug,
		map[string]any{"provider_ref": "TXN1"}); rec.Code != http.StatusOK {
		t.Fatalf("proof: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/v1/orders/"+order.ID+"/review", owner.token, owner.slug,
		map[string]any{"decision": "approve"}); rec.Code != http.StatusOK {
		t.Fatalf("approve: got %d: %s", rec.Code, rec.Body)
	}

	var notificationID string
	t.Run("the learner has an unread announcement", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/notifications", student.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Notifications []struct {
				ID     string  `json:"id"`
				Kind   string  `json:"kind"`
				Title  string  `json:"title"`
				ReadAt *string `json:"read_at"`
			} `json:"notifications"`
			Unread int64 `json:"unread"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Unread != 1 || len(got.Notifications) != 1 {
			t.Fatalf("got %+v", got)
		}
		if got.Notifications[0].Kind != "payment.approved" || got.Notifications[0].ReadAt != nil {
			t.Fatalf("got %+v", got.Notifications[0])
		}
		notificationID = got.Notifications[0].ID
	})

	t.Run("nobody else receives it", func(t *testing.T) {
		rec := do(t, h, "GET", "/v1/notifications", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var got struct {
			Unread int64 `json:"unread"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Unread != 0 {
			t.Errorf("the teacher has %d unread, want 0", got.Unread)
		}
	})

	t.Run("the dispatcher sends it to the channel", func(t *testing.T) {
		dispatcher := notify.NewDispatcher(server, ch)
		n, err := dispatcher.Drain(context.Background())
		if err != nil {
			t.Fatalf("Drain: %v", err)
		}
		if n == 0 {
			t.Fatal("nothing was queued for delivery")
		}
		if ch.last.To == "" || ch.last.Body == "" {
			t.Fatalf("the channel received %+v", ch.last)
		}

		// Draining again must not resend what already went out.
		before := ch.last
		if _, err := dispatcher.Drain(context.Background()); err != nil {
			t.Fatalf("second Drain: %v", err)
		}
		if ch.last != before {
			t.Errorf("a settled delivery was sent again: %+v", ch.last)
		}
	})

	t.Run("marking read is idempotent and scoped to the owner", func(t *testing.T) {
		if rec := do(t, h, "POST", "/v1/notifications/"+notificationID+"/read", owner.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("another user marking it read: got %d, want 404", rec.Code)
		}
		if rec := do(t, h, "POST", "/v1/notifications/"+notificationID+"/read", student.token, owner.slug, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("mark read: got %d", rec.Code)
		}
		if rec := do(t, h, "POST", "/v1/notifications/"+notificationID+"/read", student.token, owner.slug, nil); rec.Code != http.StatusNotFound {
			t.Errorf("marking twice: got %d, want 404", rec.Code)
		}

		rec := do(t, h, "GET", "/v1/notifications", student.token, owner.slug, nil)
		var got struct {
			Unread int64 `json:"unread"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Unread != 0 {
			t.Errorf("unread = %d, want 0", got.Unread)
		}
	})

	t.Run("marking a quiz announces the result", func(t *testing.T) {
		learner := enrollIn(t, h, ch, store, owner.slug, "student")
		attemptID, _ := essayAttempt(t, h, owner, learner)
		sheet := readSheet(t, h, owner, attemptID)
		var essayID string
		for _, q := range sheet.Questions {
			if q.Kind == "essay" {
				essayID = q.ID
			}
		}
		if rec := do(t, h, "PUT", "/v1/attempts/"+attemptID+"/questions/"+essayID+"/grade", owner.token, owner.slug,
			map[string]any{"points_awarded": 4}); rec.Code != http.StatusOK {
			t.Fatalf("mark: got %d: %s", rec.Code, rec.Body)
		}
		if rec := do(t, h, "POST", "/v1/attempts/"+attemptID+"/release", owner.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("release: got %d: %s", rec.Code, rec.Body)
		}

		rec := do(t, h, "GET", "/v1/notifications", learner.token, owner.slug, nil)
		var got struct {
			Notifications []struct {
				Kind string `json:"kind"`
			} `json:"notifications"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Notifications) == 0 || got.Notifications[0].Kind != "quiz.result" {
			t.Fatalf("got %+v, want a quiz result announcement", got.Notifications)
		}
	})

	t.Run("a failing channel is retried, not lost", func(t *testing.T) {
		queue := &countingQueue{server: server}
		dispatcher := notify.NewDispatcher(queue, brokenChannel{})
		if _, err := dispatcher.Drain(context.Background()); err != nil {
			t.Fatalf("Drain: %v", err)
		}
		if queue.requeued == 0 {
			t.Error("a failed send should be queued again rather than dropped")
		}
	})
}

// countingQueue watches how the dispatcher settles failures.
type countingQueue struct {
	server   *api.Server
	requeued int
}

func (q *countingQueue) Claim(ctx context.Context, limit int32) ([]notify.Delivery, error) {
	return q.server.Claim(ctx, limit)
}

func (q *countingQueue) Settle(ctx context.Context, id, state, failure string, wait time.Duration) error {
	if state == "queued" {
		q.requeued++
	}
	return q.server.Settle(ctx, id, state, failure, wait)
}

type brokenChannel struct{}

func (brokenChannel) Name() string { return "capture" }

func (brokenChannel) Send(context.Context, notify.Message) error {
	return errBroken
}
