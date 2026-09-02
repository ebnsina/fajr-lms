package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/api"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/media"
	"github.com/ebnsina/fajr-lms/internal/payment"
)

// gatewayStub answers the validation call with whatever the test needs.
type gatewayStub struct {
	validation map[string]any
	calls      int
}

func (g *gatewayStub) server(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/gwprocess/v4/api.php", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "SUCCESS", "GatewayPageURL": "https://gateway.test/pay/1",
		})
	})
	mux.HandleFunc("/validator/api/validationserverAPI.php", func(w http.ResponseWriter, r *http.Request) {
		g.calls++
		_ = json.NewEncoder(w).Encode(g.validation)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

// gatewayHarness wires a server whose default payment method is the stub gateway.
func gatewayHarness(t *testing.T, g *gatewayStub) (http.Handler, *captureChannel, *database.Store) {
	t.Helper()
	h, ch, store := newHarness(t)
	_ = h

	payments, err := payment.NewRegistry("sslcommerz",
		payment.SSLCommerz{StoreID: "testbox", StorePasswd: "qwerty", BaseURL: g.server(t)},
		payment.BankTransfer{AccountNumber: "1"},
	)
	if err != nil {
		t.Fatalf("build payment registry: %v", err)
	}
	return api.NewServer(store, identity.New(store, ch), mediaRegistry(t), payments, "https://fajr.test").Routes(), ch, store
}

func mediaRegistry(t *testing.T) *media.Registry {
	t.Helper()
	return testRegistry(t)
}

func postForm(t *testing.T, h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGatewayCallback(t *testing.T) {
	g := &gatewayStub{}
	h, ch, store := gatewayHarness(t, g)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	courseID := paidCourse(t, h, owner, 150000)

	rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create order: got %d: %s", rec.Code, rec.Body)
	}
	var order orderBody
	if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if order.Instruction.Kind != payment.InstructRedirect || order.Instruction.URL == "" {
		t.Fatalf("got %+v, want a redirect", order.Instruction)
	}

	callback := "/v1/payment/" + owner.slug + "/sslcommerz/callback"
	// Event ids are globally unique per provider, so they cannot repeat between runs.
	valForged, valGenuine := "FORGED-"+order.Reference, "VAL-"+order.Reference

	t.Run("an unknown tenant or provider is not found", func(t *testing.T) {
		form := url.Values{"tran_id": {order.Reference}, "val_id": {"U-" + order.Reference}}
		if rec := postForm(t, h, "/v1/payment/nope/sslcommerz/callback", form); rec.Code != http.StatusNotFound {
			t.Errorf("tenant: got %d, want 404", rec.Code)
		}
		if rec := postForm(t, h, "/v1/payment/"+owner.slug+"/paypal/callback", form); rec.Code != http.StatusNotFound {
			t.Errorf("provider: got %d, want 404", rec.Code)
		}
	})

	t.Run("a callback for an unknown order is not found", func(t *testing.T) {
		rec := postForm(t, h, callback, url.Values{"tran_id": {"FJ-ZZZZ-ZZZZ"}, "val_id": {"Z-" + order.Reference}})
		if rec.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a callback with no reference is refused", func(t *testing.T) {
		if rec := postForm(t, h, callback, url.Values{"val_id": {"N-" + order.Reference}}); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("got %d, want 422: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a forged amount does not settle the order", func(t *testing.T) {
		g.validation = map[string]any{
			"status": "VALID", "tran_id": order.Reference, "amount": "1.00", "currency": "BDT",
		}
		rec := postForm(t, h, callback, url.Values{
			"tran_id": {order.Reference}, "val_id": {valForged}, "amount": {"1500.00"}, "status": {"VALID"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if settledFlag(t, rec) {
			t.Error("an underpayment settled the order")
		}
		assertOrderStatus(t, h, student, owner.slug, "pending")
	})

	t.Run("a genuine payment settles and enrols", func(t *testing.T) {
		g.validation = map[string]any{
			"status": "VALID", "tran_id": order.Reference, "amount": "1500.00",
			"currency": "BDT", "bank_tran_id": "BANK-9",
		}
		rec := postForm(t, h, callback, url.Values{
			"tran_id": {order.Reference}, "val_id": {valGenuine}, "status": {"VALID"},
		})
		if rec.Code != http.StatusOK || !settledFlag(t, rec) {
			t.Fatalf("got %d %s", rec.Code, rec.Body)
		}
		assertOrderStatus(t, h, student, owner.slug, "paid")

		if rec := do(t, h, "GET", "/v1/courses/"+courseID+"/progress", student.token, owner.slug, nil); rec.Code != http.StatusOK {
			t.Fatalf("learner not enrolled after payment: got %d: %s", rec.Code, rec.Body)
		}
	})

	t.Run("a replayed callback is acknowledged but does nothing", func(t *testing.T) {
		before := g.calls
		rec := postForm(t, h, callback, url.Values{
			"tran_id": {order.Reference}, "val_id": {valGenuine}, "status": {"VALID"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		if settledFlag(t, rec) {
			t.Error("a replay settled the order a second time")
		}
		if g.calls != before {
			t.Errorf("a replay re-queried the gateway %d times", g.calls-before)
		}
	})

	t.Run("a manual method cannot be settled by callback", func(t *testing.T) {
		buyer := enrolIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", buyer.token, owner.slug,
			map[string]any{"provider": "bank_transfer"})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create order: got %d: %s", rec.Code, rec.Body)
		}
		var manual orderBody
		if err := json.Unmarshal(rec.Body.Bytes(), &manual); err != nil {
			t.Fatalf("decode: %v", err)
		}
		got := postForm(t, h, "/v1/payment/"+owner.slug+"/bank_transfer/callback",
			url.Values{"tran_id": {manual.Reference}, "val_id": {"X-" + manual.Reference}})
		if got.Code != http.StatusConflict {
			t.Errorf("got %d, want 409: %s", got.Code, got.Body)
		}
	})

	t.Run("a cancelled redirect cancels the order", func(t *testing.T) {
		buyer := enrolIn(t, h, ch, store, owner.slug, "student")
		rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", buyer.token, owner.slug, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create order: got %d: %s", rec.Code, rec.Body)
		}
		var pending orderBody
		if err := json.Unmarshal(rec.Body.Bytes(), &pending); err != nil {
			t.Fatalf("decode: %v", err)
		}
		got := postForm(t, h, callback, url.Values{
			"tran_id": {pending.Reference}, "status": {"CANCELLED"},
		})
		if got.Code != http.StatusOK {
			t.Fatalf("got %d: %s", got.Code, got.Body)
		}
		assertOrderStatus(t, h, buyer, owner.slug, "cancelled")
	})
}

func settledFlag(t *testing.T, rec *httptest.ResponseRecorder) bool {
	t.Helper()
	var got struct {
		Settled bool `json:"settled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode callback response: %v", err)
	}
	return got.Settled
}

func assertOrderStatus(t *testing.T, h http.Handler, a actor, slug, want string) {
	t.Helper()
	rec := do(t, h, "GET", "/v1/orders", a.token, slug, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list orders: got %d: %s", rec.Code, rec.Body)
	}
	var got struct {
		Orders []struct {
			Order struct {
				Status string `json:"status"`
			} `json:"order"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode orders: %v", err)
	}
	if len(got.Orders) == 0 || got.Orders[0].Order.Status != want {
		t.Fatalf("order status = %+v, want %q", got.Orders, want)
	}
}

func TestBrowserReturnRedirects(t *testing.T) {
	g := &gatewayStub{}
	h, ch, store := gatewayHarness(t, g)
	owner := enrol(t, h, ch, store, "owner")
	student := enrolIn(t, h, ch, store, owner.slug, "student")
	courseID := paidCourse(t, h, owner, 150000)

	rec := do(t, h, "POST", "/v1/courses/"+courseID+"/orders", student.token, owner.slug, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create order: got %d: %s", rec.Code, rec.Body)
	}
	var order orderBody
	if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
		t.Fatalf("decode: %v", err)
	}

	g.validation = map[string]any{
		"status": "VALID", "tran_id": order.Reference, "amount": "1500.00", "currency": "BDT",
	}

	// The payer's browser comes back by GET, not a server-to-server POST.
	path := "/v1/payment/" + owner.slug + "/sslcommerz/callback?tran_id=" +
		url.QueryEscape(order.Reference) + "&val_id=GET-" + url.QueryEscape(order.Reference) + "&status=VALID"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	got := httptest.NewRecorder()
	h.ServeHTTP(got, req)

	if got.Code != http.StatusSeeOther {
		t.Fatalf("got %d, want 303: %s", got.Code, got.Body)
	}
	location := got.Header().Get("Location")
	if !strings.HasPrefix(location, "https://fajr.test/pay/paid") {
		t.Errorf("Location = %q, want a paid landing page", location)
	}
	if !strings.Contains(location, url.QueryEscape(order.Reference)) {
		t.Errorf("Location = %q, want it to carry the reference", location)
	}
	assertOrderStatus(t, h, student, owner.slug, "paid")
}
