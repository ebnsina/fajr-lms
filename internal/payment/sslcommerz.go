package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SSLCommerz is Bangladesh's dominant gateway: cards, net banking, bKash,
// Nagad and Rocket behind one redirect.
type SSLCommerz struct {
	StoreID     string
	StorePasswd string
	Sandbox     bool
	SuccessURL  string
	FailURL     string
	CancelURL   string
	IPNURL      string
	HTTP        *http.Client

	// BaseURL overrides the gateway host, for tests.
	BaseURL string
}

func (s SSLCommerz) Caps() Caps {
	return Caps{
		Name: "sslcommerz", Title: "Card, bKash, Nagad or bank",
		Currencies: []string{"BDT"}, Redirects: true,
	}
}

func (s SSLCommerz) base() string {
	switch {
	case s.BaseURL != "":
		return strings.TrimSuffix(s.BaseURL, "/")
	case s.Sandbox:
		return "https://sandbox.sslcommerz.com"
	default:
		return "https://securepay.sslcommerz.com"
	}
}

func (s SSLCommerz) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

type sessionResponse struct {
	Status         string `json:"status"`
	FailedReason   string `json:"failedreason"`
	GatewayPageURL string `json:"GatewayPageURL"`
}

// Start opens a gateway session and returns the page to send the payer to.
func (s SSLCommerz) Start(ctx context.Context, o Order) (Instruction, error) {
	if s.StoreID == "" || s.StorePasswd == "" {
		return Instruction{}, fmt.Errorf("payment: sslcommerz needs a store id and password")
	}
	if !strings.EqualFold(o.Currency, "BDT") {
		return Instruction{}, fmt.Errorf("payment: sslcommerz settles in BDT, not %s", o.Currency)
	}

	form := url.Values{
		"store_id": {s.StoreID}, "store_passwd": {s.StorePasswd},
		"total_amount": {majorUnits(o.AmountMinor)}, "currency": {strings.ToUpper(o.Currency)},
		"tran_id":     {o.Reference},
		"success_url": {forTenant(s.SuccessURL, o)}, "fail_url": {forTenant(s.FailURL, o)},
		"cancel_url": {forTenant(s.CancelURL, o)}, "ipn_url": {forTenant(s.IPNURL, o)},
		"cus_name": {orDefault(o.PayerName, "Learner")}, "cus_email": {"noreply@fajr.invalid"},
		"cus_phone": {"01700000000"}, "cus_add1": {"N/A"}, "cus_city": {"N/A"}, "cus_country": {"Bangladesh"},
		"shipping_method": {"NO"}, "num_of_item": {"1"},
		"product_name":     {orDefault(o.Description, "Course enrollment")},
		"product_category": {"education"}, "product_profile": {"non-physical-goods"},
		"value_a": {o.ID},
	}

	body, err := s.post(ctx, s.base()+"/gwprocess/v4/api.php", form)
	if err != nil {
		return Instruction{}, err
	}

	var out sessionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return Instruction{}, fmt.Errorf("payment: sslcommerz sent an unreadable session response: %w", err)
	}
	if !strings.EqualFold(out.Status, "SUCCESS") || out.GatewayPageURL == "" {
		return Instruction{}, fmt.Errorf("payment: sslcommerz refused the session: %s", orDefault(out.FailedReason, out.Status))
	}
	return Instruction{Kind: InstructRedirect, URL: out.GatewayPageURL, Reference: o.Reference}, nil
}

type validationResponse struct {
	Status      string `json:"status"`
	TranID      string `json:"tran_id"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	BankTranID  string `json:"bank_tran_id"`
	ValID       string `json:"val_id"`
	CurrencyAmt string `json:"currency_amount"`
}

// Verify ignores the amounts in the callback and asks SSLCommerz directly.
// A posted IPN body is attacker-controlled; the validation API is not.
func (s SSLCommerz) Verify(ctx context.Context, o Order, cb Callback) (Result, error) {
	valID, _ := cb.Raw["val_id"].(string)
	if strings.TrimSpace(valID) == "" {
		// No validation id means the payer failed or cancelled at the gateway.
		return Result{Status: statusFromCallback(cb)}, nil
	}

	query := url.Values{
		"val_id": {valID}, "store_id": {s.StoreID}, "store_passwd": {s.StorePasswd},
		"format": {"json"}, "v": {"1"},
	}
	endpoint := s.base() + "/validator/api/validationserverAPI.php?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, fmt.Errorf("payment: build validation request: %w", err)
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("payment: reach sslcommerz validation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Result{}, fmt.Errorf("payment: read validation response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("payment: sslcommerz validation returned %d", resp.StatusCode)
	}

	var out validationResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return Result{}, fmt.Errorf("payment: sslcommerz sent an unreadable validation response: %w", err)
	}

	// VALIDATED means we asked twice; both are successful payments.
	if !strings.EqualFold(out.Status, "VALID") && !strings.EqualFold(out.Status, "VALIDATED") {
		return Result{Status: StatusRejected, ProviderRef: out.BankTranID}, nil
	}
	if out.TranID != o.Reference {
		return Result{}, fmt.Errorf("%w: gateway reports %q, order is %q", ErrEventMismatch, out.TranID, o.Reference)
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
		Status: StatusPaid, ProviderRef: orDefault(out.BankTranID, out.ValID),
		AmountMinor: int64(math.Round(paid * 100)), Currency: strings.ToUpper(o.Currency),
	}, nil
}

func (s SSLCommerz) post(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("payment: build session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("payment: reach sslcommerz: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("payment: read sslcommerz response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment: sslcommerz returned %d", resp.StatusCode)
	}
	return body, nil
}

// statusFromCallback maps a gateway redirect with no validation id.
func statusFromCallback(cb Callback) Status {
	raw, _ := cb.Raw["status"].(string)
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "CANCELLED":
		return StatusCancelled
	default:
		return StatusRejected
	}
}

// forTenant fills the callback URL in for this order's tenant, since one
// deployment serves many and the gateway must come back to the right one.
func forTenant(raw string, o Order) string {
	return strings.NewReplacer("{tenant}", o.TenantSlug, "{reference}", o.Reference).Replace(raw)
}

func majorUnits(minor int64) string {
	return fmt.Sprintf("%d.%02d", minor/100, minor%100)
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
