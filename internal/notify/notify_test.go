package notify_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ebnsina/fajr-lms/internal/notify"
)

// memQueue is an in-memory Queue, so the dispatcher is testable on its own.
type memQueue struct {
	mu       sync.Mutex
	pending  []notify.Delivery
	outcomes map[string]string
	failures map[string]string
}

func newQueue(deliveries ...notify.Delivery) *memQueue {
	return &memQueue{
		pending: deliveries, outcomes: map[string]string{}, failures: map[string]string{},
	}
}

func (q *memQueue) Claim(context.Context, int32) ([]notify.Delivery, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := q.pending
	q.pending = nil
	return out, nil
}

func (q *memQueue) Settle(_ context.Context, id, state, failure string, _ time.Duration) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.outcomes[id], q.failures[id] = state, failure
	return nil
}

type stubChannel struct {
	name string
	err  error
	sent []notify.Message
}

func (c *stubChannel) Name() string { return c.name }

func (c *stubChannel) Send(_ context.Context, m notify.Message) error {
	c.sent = append(c.sent, m)
	return c.err
}

func TestDispatcherOutcomes(t *testing.T) {
	sms := &stubChannel{name: "sms"}
	broken := &stubChannel{name: "whatsapp", err: errors.New("gateway down")}

	queue := newQueue(
		notify.Delivery{ID: "1", Channel: "sms", Destination: "+8801700000000", Body: "hello"},
		notify.Delivery{ID: "2", Channel: "whatsapp", Destination: "+8801700000001", Body: "hi", Attempts: 1},
		notify.Delivery{ID: "3", Channel: "whatsapp", Destination: "+8801700000002", Body: "hi", Attempts: 9},
		notify.Delivery{ID: "4", Channel: "carrier_pigeon", Destination: "+8801700000003", Body: "hi"},
	)

	d := notify.NewDispatcher(queue, sms, broken)
	n, err := d.Drain(context.Background())
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if n != 4 {
		t.Fatalf("handled %d, want 4", n)
	}

	want := map[string]string{"1": "sent", "2": "queued", "3": "failed", "4": "skipped"}
	for id, state := range want {
		if queue.outcomes[id] != state {
			t.Errorf("delivery %s = %q, want %q", id, queue.outcomes[id], state)
		}
	}
	if len(sms.sent) != 1 || sms.sent[0].To != "+8801700000000" {
		t.Errorf("sms channel got %+v", sms.sent)
	}
	if queue.failures["4"] == "" {
		t.Error("an unconfigured channel should record why it was skipped")
	}
}

func TestHTTPChannelForm(t *testing.T) {
	var got struct {
		method string
		form   map[string]string
		auth   string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got.method, got.auth = r.Method, r.Header.Get("Authorization")
		got.form = map[string]string{}
		for key := range r.PostForm {
			got.form[key] = r.PostForm.Get(key)
		}
		_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
	}))
	defer srv.Close()

	channel := notify.HTTPChannel{
		ChannelName: "sms", Method: "POST", URL: srv.URL, Encoding: "form",
		Body:            "msisdn={to}&sms={message}&sender={sender}",
		Sender:          "FAJR",
		Headers:         map[string]string{"Authorization": "Bearer k"},
		SuccessContains: "SUCCESS",
	}

	msg := notify.Message{To: "+8801700000000", Body: "আপনার ফলাফল প্রকাশিত হয়েছে"}
	if err := channel.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got.method != "POST" || got.auth != "Bearer k" {
		t.Errorf("got %+v", got)
	}
	if got.form["msisdn"] != msg.To || got.form["sms"] != msg.Body || got.form["sender"] != "FAJR" {
		t.Errorf("form = %+v", got.form)
	}
}

func TestHTTPChannelJSONAndFailures(t *testing.T) {
	var body string
	status := http.StatusOK
	reply := `{"ok":true}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body = string(buf[:n])
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	defer srv.Close()

	channel := notify.HTTPChannel{
		ChannelName: "whatsapp", URL: srv.URL, Encoding: "json",
		Body: `{"to":"{to}","text":"{message}"}`, SuccessContains: `"ok":true`,
	}

	// A quote in the message must not break the JSON template.
	msg := notify.Message{To: "+8801700000000", Body: `he said "pass"` + "\n" + `now`}
	if err := channel.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(body, `\"pass\"`) || !strings.Contains(body, `\n`) {
		t.Errorf("json body was not escaped: %s", body)
	}

	t.Run("a non-2xx response fails", func(t *testing.T) {
		status = http.StatusBadGateway
		if err := channel.Send(context.Background(), msg); err == nil {
			t.Error("a 502 should fail the send")
		}
	})

	t.Run("a 200 with a failure body fails", func(t *testing.T) {
		status, reply = http.StatusOK, `{"ok":false,"reason":"no balance"}`
		if err := channel.Send(context.Background(), msg); err == nil {
			t.Error("a gateway rejecting in the body should fail the send")
		}
	})

	t.Run("a message with no destination fails", func(t *testing.T) {
		if err := channel.Send(context.Background(), notify.Message{Body: "x"}); err == nil {
			t.Error("an empty destination should fail")
		}
	})
}
