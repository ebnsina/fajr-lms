package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BKash is the dominant wallet in Bangladesh. Tokenized checkout: grant a
// token, create a payment, redirect, then execute to confirm.
type BKash struct {
	Username    string
	Password    string
	AppKey      string
	AppSecret   string
	Sandbox     bool
	CallbackURL string
	// Mode 0011 is tokenized checkout without an agreement.
	Mode   string
	Intent string
	HTTP   *http.Client

	// BaseURL overrides the API host, for tests.
	BaseURL string

	mu      sync.Mutex
	token   string
	expires time.Time
}

func (b *BKash) Caps() Caps {
	return Caps{Name: "bkash", Title: "bKash", Currencies: []string{"BDT"}, Redirects: true}
}

func (b *BKash) base() string {
	switch {
	case b.BaseURL != "":
		return strings.TrimSuffix(b.BaseURL, "/")
	case b.Sandbox:
		return "https://tokenized.sandbox.bka.sh/v1.2.0-beta"
	default:
		return "https://tokenized.pay.bka.sh/v1.2.0-beta"
	}
}

func (b *BKash) client() *http.Client {
	if b.HTTP != nil {
		return b.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

type bkashToken struct {
	IDToken   string `json:"id_token"`
	ExpiresIn int64  `json:"expires_in"`
	Message   string `json:"statusMessage"`
}

// idToken returns a cached token, refreshing a minute before it lapses.
func (b *BKash) idToken(ctx context.Context) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.token != "" && time.Now().Before(b.expires) {
		return b.token, nil
	}
	if b.Username == "" || b.AppKey == "" {
		return "", fmt.Errorf("payment: bkash needs a username, password, app key and app secret")
	}

	body, err := json.Marshal(map[string]string{"app_key": b.AppKey, "app_secret": b.AppSecret})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.base()+"/tokenized/checkout/token/grant", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("payment: build bkash token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("username", b.Username)
	req.Header.Set("password", b.Password)

	raw, err := b.do(req, "token grant")
	if err != nil {
		return "", err
	}
	var out bkashToken
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("payment: bkash sent an unreadable token: %w", err)
	}
	if out.IDToken == "" {
		return "", fmt.Errorf("payment: bkash refused the token grant: %s", orDefault(out.Message, "no token returned"))
	}

	ttl := time.Duration(out.ExpiresIn) * time.Second
	if ttl < 2*time.Minute {
		ttl = 2 * time.Minute
	}
	b.token, b.expires = out.IDToken, time.Now().Add(ttl-time.Minute)
	return b.token, nil
}

type bkashPayment struct {
	StatusCode            string `json:"statusCode"`
	StatusMessage         string `json:"statusMessage"`
	PaymentID             string `json:"paymentID"`
	BKashURL              string `json:"bkashURL"`
	TrxID                 string `json:"trxID"`
	TransactionStatus     string `json:"transactionStatus"`
	Amount                string `json:"amount"`
	Currency              string `json:"currency"`
	MerchantInvoiceNumber string `json:"merchantInvoiceNumber"`
}

// Start creates a payment and returns the bKash page to redirect to.
func (b *BKash) Start(ctx context.Context, o Order) (Instruction, error) {
	if !strings.EqualFold(o.Currency, "BDT") {
		return Instruction{}, fmt.Errorf("payment: bkash settles in BDT, not %s", o.Currency)
	}

	token, err := b.idToken(ctx)
	if err != nil {
		return Instruction{}, err
	}

	raw, err := b.call(ctx, token, "/tokenized/checkout/create", map[string]string{
		"mode": orDefault(b.Mode, "0011"), "intent": orDefault(b.Intent, "sale"),
		"currency": "BDT", "amount": majorUnits(o.AmountMinor),
		"merchantInvoiceNumber": o.Reference,
		"payerReference":        o.Reference,
		"callbackURL":           forTenant(b.CallbackURL, o),
	}, "create payment")
	if err != nil {
		return Instruction{}, err
	}

	var out bkashPayment
	if err := json.Unmarshal(raw, &out); err != nil {
		return Instruction{}, fmt.Errorf("payment: bkash sent an unreadable create response: %w", err)
	}
	if out.BKashURL == "" {
		return Instruction{}, fmt.Errorf("payment: bkash refused the payment: %s",
			orDefault(out.StatusMessage, out.StatusCode))
	}
	return Instruction{Kind: InstructRedirect, URL: out.BKashURL, Reference: o.Reference}, nil
}

// Verify executes the payment, which is what actually captures the money. The
// callback only carries a paymentID; every figure comes back from bKash.
func (b *BKash) Verify(ctx context.Context, o Order, cb Callback) (Result, error) {
	paymentID := strings.TrimSpace(stringField(cb.Raw, "paymentID"))
	if paymentID == "" {
		return Result{Status: bkashCallbackStatus(cb)}, nil
	}
	if status := strings.ToLower(stringField(cb.Raw, "status")); status == "cancel" || status == "failure" {
		return Result{Status: bkashCallbackStatus(cb)}, nil
	}

	token, err := b.idToken(ctx)
	if err != nil {
		return Result{}, err
	}

	raw, err := b.call(ctx, token, "/tokenized/checkout/execute", map[string]string{"paymentID": paymentID}, "execute payment")
	if err != nil {
		return Result{}, err
	}

	var out bkashPayment
	if err := json.Unmarshal(raw, &out); err != nil {
		return Result{}, fmt.Errorf("payment: bkash sent an unreadable execute response: %w", err)
	}
	if !strings.EqualFold(out.TransactionStatus, "Completed") {
		return Result{Status: StatusRejected, ProviderRef: out.TrxID}, nil
	}
	if out.MerchantInvoiceNumber != o.Reference {
		return Result{}, fmt.Errorf("%w: bkash reports invoice %q, order is %q",
			ErrEventMismatch, out.MerchantInvoiceNumber, o.Reference)
	}

	paid, err := strconv.ParseFloat(strings.TrimSpace(out.Amount), 64)
	if err != nil {
		return Result{}, fmt.Errorf("%w: unreadable amount %q", ErrEventMismatch, out.Amount)
	}
	if math.Abs(paid*100-float64(o.AmountMinor)) > 1 {
		return Result{}, fmt.Errorf("%w: paid %.2f, order is %s", ErrEventMismatch, paid, majorUnits(o.AmountMinor))
	}
	if out.Currency != "" && !strings.EqualFold(out.Currency, o.Currency) {
		return Result{}, fmt.Errorf("%w: paid in %s, order is in %s", ErrEventMismatch, out.Currency, o.Currency)
	}

	return Result{
		Status: StatusPaid, ProviderRef: orDefault(out.TrxID, paymentID),
		AmountMinor: int64(math.Round(paid * 100)), Currency: "BDT",
	}, nil
}

func (b *BKash) call(ctx context.Context, token, path string, body map[string]string, what string) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.base()+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("payment: build bkash %s request: %w", what, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("authorization", token)
	req.Header.Set("x-app-key", b.AppKey)
	return b.do(req, what)
}

func (b *BKash) do(req *http.Request, what string) ([]byte, error) {
	resp, err := b.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("payment: reach bkash for %s: %w", what, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("payment: read bkash %s response: %w", what, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment: bkash %s returned %d", what, resp.StatusCode)
	}
	return raw, nil
}

func bkashCallbackStatus(cb Callback) Status {
	switch strings.ToLower(strings.TrimSpace(stringField(cb.Raw, "status"))) {
	case "cancel", "cancelled":
		return StatusCancelled
	default:
		return StatusRejected
	}
}

func stringField(raw map[string]any, key string) string {
	v, _ := raw[key].(string)
	return v
}
