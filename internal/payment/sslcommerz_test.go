package payment_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/payment"
)

// fakeGateway stands in for SSLCommerz so the adapter is exercised for real.
type fakeGateway struct {
	session    map[string]any
	validation map[string]any
	lastForm   map[string]string
}

func (f *fakeGateway) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/gwprocess/v4/api.php", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		f.lastForm = map[string]string{}
		for k := range r.PostForm {
			f.lastForm[k] = r.PostForm.Get(k)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.session)
	})
	mux.HandleFunc("/validator/api/validationserverAPI.php", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("store_id") == "" || r.URL.Query().Get("val_id") == "" {
			http.Error(w, "missing credentials", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.validation)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newSSLCommerz(t *testing.T, f *fakeGateway) payment.SSLCommerz {
	t.Helper()
	return payment.SSLCommerz{
		StoreID: "testbox", StorePasswd: "qwerty", BaseURL: f.server(t).URL,
		SuccessURL: "https://fajr.test/ok", FailURL: "https://fajr.test/fail",
		CancelURL: "https://fajr.test/cancel",
		IPNURL:    "https://fajr.test/v1/payment/{tenant}/sslcommerz/callback",
	}
}

var order = payment.Order{
	ID: "order-1", TenantSlug: "darul-uloom", Reference: "FJ-ABCD-EFGH",
	AmountMinor: 150000, Currency: "BDT",
	PayerName: "আয়েশা রহমান", Description: "Advanced Tajweed",
}

func TestSSLCommerzStart(t *testing.T) {
	f := &fakeGateway{session: map[string]any{
		"status": "SUCCESS", "GatewayPageURL": "https://sandbox.sslcommerz.com/pay/abc123",
	}}
	p := newSSLCommerz(t, f)

	got, err := p.Start(context.Background(), order)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got.Kind != payment.InstructRedirect || got.URL != "https://sandbox.sslcommerz.com/pay/abc123" {
		t.Fatalf("got %+v", got)
	}
	if f.lastForm["tran_id"] != order.Reference {
		t.Errorf("tran_id = %q, want the order reference %q", f.lastForm["tran_id"], order.Reference)
	}
	if f.lastForm["total_amount"] != "1500.00" {
		t.Errorf("total_amount = %q, want 1500.00", f.lastForm["total_amount"])
	}
	if f.lastForm["store_passwd"] != "qwerty" {
		t.Errorf("session form is missing credentials: %+v", f.lastForm)
	}
	// One deployment serves many tenants, so the callback must name this one.
	want := "https://fajr.test/v1/payment/darul-uloom/sslcommerz/callback"
	if f.lastForm["ipn_url"] != want {
		t.Errorf("ipn_url = %q, want %q", f.lastForm["ipn_url"], want)
	}
}

func TestSSLCommerzStartRefused(t *testing.T) {
	f := &fakeGateway{session: map[string]any{"status": "FAILED", "failedreason": "Invalid store"}}
	if _, err := newSSLCommerz(t, f).Start(context.Background(), order); err == nil {
		t.Fatal("a refused session should fail")
	}

	// A non-BDT order must never reach the gateway.
	usd := order
	usd.Currency = "USD"
	if _, err := newSSLCommerz(t, f).Start(context.Background(), usd); err == nil {
		t.Error("sslcommerz should refuse a non-BDT order")
	}
}

func TestSSLCommerzVerify(t *testing.T) {
	valid := map[string]any{
		"status": "VALID", "tran_id": order.Reference, "amount": "1500.00",
		"currency": "BDT", "bank_tran_id": "BANK123", "val_id": "VAL1",
	}
	cb := payment.Callback{EventID: "VAL1", Raw: map[string]any{"val_id": "VAL1"}}

	t.Run("a validated payment settles", func(t *testing.T) {
		p := newSSLCommerz(t, &fakeGateway{validation: valid})
		got, err := p.Verify(context.Background(), order, cb)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if got.Status != payment.StatusPaid || got.ProviderRef != "BANK123" || got.AmountMinor != 150000 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("VALIDATED is also a payment", func(t *testing.T) {
		v := maps(valid, "status", "VALIDATED")
		p := newSSLCommerz(t, &fakeGateway{validation: v})
		got, err := p.Verify(context.Background(), order, cb)
		if err != nil || got.Status != payment.StatusPaid {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("an underpayment is refused", func(t *testing.T) {
		p := newSSLCommerz(t, &fakeGateway{validation: maps(valid, "amount", "15.00")})
		if _, err := p.Verify(context.Background(), order, cb); !errors.Is(err, payment.ErrEventMismatch) {
			t.Errorf("got %v, want ErrEventMismatch", err)
		}
	})

	t.Run("a payment for another order is refused", func(t *testing.T) {
		p := newSSLCommerz(t, &fakeGateway{validation: maps(valid, "tran_id", "FJ-ZZZZ-ZZZZ")})
		if _, err := p.Verify(context.Background(), order, cb); !errors.Is(err, payment.ErrEventMismatch) {
			t.Errorf("got %v, want ErrEventMismatch", err)
		}
	})

	t.Run("a wrong currency is refused", func(t *testing.T) {
		p := newSSLCommerz(t, &fakeGateway{validation: maps(valid, "currency", "USD")})
		if _, err := p.Verify(context.Background(), order, cb); !errors.Is(err, payment.ErrEventMismatch) {
			t.Errorf("got %v, want ErrEventMismatch", err)
		}
	})

	t.Run("an invalid transaction is rejected, not paid", func(t *testing.T) {
		p := newSSLCommerz(t, &fakeGateway{validation: maps(valid, "status", "INVALID_TRANSACTION")})
		got, err := p.Verify(context.Background(), order, cb)
		if err != nil || got.Status != payment.StatusRejected {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("amounts in the callback body are ignored", func(t *testing.T) {
		// The gateway says 1500; a forged callback claiming otherwise changes nothing.
		p := newSSLCommerz(t, &fakeGateway{validation: valid})
		forged := payment.Callback{EventID: "VAL1", Raw: map[string]any{
			"val_id": "VAL1", "amount": "1.00", "status": "VALID",
		}}
		got, err := p.Verify(context.Background(), order, forged)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if got.AmountMinor != 150000 {
			t.Errorf("trusted the callback body: got %d, want 150000", got.AmountMinor)
		}
	})

	t.Run("a cancelled redirect carries no validation id", func(t *testing.T) {
		p := newSSLCommerz(t, &fakeGateway{validation: valid})
		got, err := p.Verify(context.Background(), order, payment.Callback{
			Raw: map[string]any{"status": "CANCELLED"},
		})
		if err != nil || got.Status != payment.StatusCancelled {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
}

func maps(base map[string]any, key string, value any) map[string]any {
	out := make(map[string]any, len(base))
	for k, v := range base {
		out[k] = v
	}
	out[key] = value
	return out
}
