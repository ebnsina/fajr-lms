package payment_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/payment"
)

// fakeBKash counts calls so token caching and one-shot execution are observable.
type fakeBKash struct {
	mu          sync.Mutex
	tokenCalls  int
	createCalls int
	execCalls   int
	create      map[string]any
	execute     map[string]any
	lastCreate  map[string]any
	lastAuth    string
}

func (f *fakeBKash) server(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/tokenized/checkout/token/grant", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.tokenCalls++
		f.mu.Unlock()
		if r.Header.Get("username") == "" || r.Header.Get("password") == "" {
			http.Error(w, "missing credentials", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type": "Bearer", "id_token": "TOKEN-1", "expires_in": 3600,
		})
	})
	mux.HandleFunc("/tokenized/checkout/create", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.createCalls++
		f.lastAuth = r.Header.Get("authorization")
		_ = json.NewDecoder(r.Body).Decode(&f.lastCreate)
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(f.create)
	})
	mux.HandleFunc("/tokenized/checkout/execute", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.execCalls++
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(f.execute)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func newBKash(t *testing.T, f *fakeBKash) *payment.BKash {
	t.Helper()
	return &payment.BKash{
		Username: "01700000000", Password: "secret", AppKey: "key", AppSecret: "secret",
		CallbackURL: "https://fajr.test/v1/payment/{tenant}/bkash/callback",
		BaseURL:     f.server(t),
	}
}

func TestBKashStart(t *testing.T) {
	f := &fakeBKash{create: map[string]any{
		"statusCode": "0000", "paymentID": "TR001", "bkashURL": "https://sandbox.bka.sh/redirect/TR001",
	}}
	p := newBKash(t, f)

	got, err := p.Start(context.Background(), order)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got.Kind != payment.InstructRedirect || got.URL != "https://sandbox.bka.sh/redirect/TR001" {
		t.Fatalf("got %+v", got)
	}
	if f.lastCreate["merchantInvoiceNumber"] != order.Reference {
		t.Errorf("invoice = %v, want %q", f.lastCreate["merchantInvoiceNumber"], order.Reference)
	}
	if f.lastCreate["amount"] != "1500.00" || f.lastCreate["currency"] != "BDT" {
		t.Errorf("got %+v", f.lastCreate)
	}
	want := "https://fajr.test/v1/payment/darul-uloom/bkash/callback"
	if f.lastCreate["callbackURL"] != want {
		t.Errorf("callbackURL = %v, want %q", f.lastCreate["callbackURL"], want)
	}
	if f.lastAuth != "TOKEN-1" {
		t.Errorf("authorization header = %q, want the granted token", f.lastAuth)
	}

	// The token is cached, so a second payment does not re-authenticate.
	if _, err := p.Start(context.Background(), order); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if f.tokenCalls != 1 {
		t.Errorf("granted %d tokens, want 1 reused across calls", f.tokenCalls)
	}
	if f.createCalls != 2 {
		t.Errorf("created %d payments, want 2", f.createCalls)
	}
}

func TestBKashStartRefused(t *testing.T) {
	f := &fakeBKash{create: map[string]any{"statusCode": "2001", "statusMessage": "Invalid app key"}}
	if _, err := newBKash(t, f).Start(context.Background(), order); err == nil {
		t.Fatal("a refused payment should fail")
	}

	usd := order
	usd.Currency = "USD"
	if _, err := newBKash(t, f).Start(context.Background(), usd); err == nil {
		t.Error("bkash should refuse a non-BDT order")
	}
}

func TestBKashVerify(t *testing.T) {
	completed := map[string]any{
		"transactionStatus": "Completed", "trxID": "TRX99", "amount": "1500.00",
		"currency": "BDT", "merchantInvoiceNumber": order.Reference, "statusCode": "0000",
	}
	cb := payment.Callback{Raw: map[string]any{"paymentID": "TR001", "status": "success"}}

	t.Run("a completed payment settles", func(t *testing.T) {
		p := newBKash(t, &fakeBKash{execute: completed})
		got, err := p.Verify(context.Background(), order, cb)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if got.Status != payment.StatusPaid || got.ProviderRef != "TRX99" || got.AmountMinor != 150000 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("an incomplete transaction is rejected", func(t *testing.T) {
		p := newBKash(t, &fakeBKash{execute: maps(completed, "transactionStatus", "Failed")})
		got, err := p.Verify(context.Background(), order, cb)
		if err != nil || got.Status != payment.StatusRejected {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("a mismatched invoice, amount or currency is refused", func(t *testing.T) {
		cases := map[string]map[string]any{
			"invoice":  maps(completed, "merchantInvoiceNumber", "FJ-ZZZZ-ZZZZ"),
			"amount":   maps(completed, "amount", "15.00"),
			"currency": maps(completed, "currency", "USD"),
		}
		for name, exec := range cases {
			p := newBKash(t, &fakeBKash{execute: exec})
			if _, err := p.Verify(context.Background(), order, cb); !errors.Is(err, payment.ErrEventMismatch) {
				t.Errorf("%s: got %v, want ErrEventMismatch", name, err)
			}
		}
	})

	t.Run("a cancelled callback never executes the payment", func(t *testing.T) {
		f := &fakeBKash{execute: completed}
		p := newBKash(t, f)
		got, err := p.Verify(context.Background(), order, payment.Callback{
			Raw: map[string]any{"paymentID": "TR001", "status": "cancel"},
		})
		if err != nil || got.Status != payment.StatusCancelled {
			t.Fatalf("got %+v, %v", got, err)
		}
		if f.execCalls != 0 {
			t.Errorf("a cancelled callback executed the payment %d times", f.execCalls)
		}
	})

	t.Run("a callback with no payment id is rejected without a call", func(t *testing.T) {
		f := &fakeBKash{execute: completed}
		got, err := newBKash(t, f).Verify(context.Background(), order, payment.Callback{Raw: map[string]any{}})
		if err != nil || got.Status != payment.StatusRejected {
			t.Fatalf("got %+v, %v", got, err)
		}
		if f.execCalls != 0 {
			t.Errorf("executed %d times without a payment id", f.execCalls)
		}
	})
}
