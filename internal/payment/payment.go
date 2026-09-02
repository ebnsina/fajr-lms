// Package payment keeps gateways behind one interface. Manual bank transfer
// ships first because it is how most school fees in South Asia are actually paid.
package payment

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
)

type Status string

const (
	StatusPending        Status = "pending"
	StatusAwaitingReview Status = "awaiting_review"
	StatusPaid           Status = "paid"
	StatusRejected       Status = "rejected"
	StatusCancelled      Status = "cancelled"
	StatusRefunded       Status = "refunded"
)

var (
	ErrUnknownProvider = errors.New("payment: unknown provider")
	ErrNotVerifiable   = errors.New("payment: this provider settles by human review")
	ErrEventMismatch   = errors.New("payment: callback does not match the order")
)

// Order is what a provider needs to start a payment. Amounts are minor units.
type Order struct {
	ID          string
	TenantID    string
	Reference   string
	AmountMinor int64
	Currency    string
	Description string
	PayerName   string
}

// InstructionKind tells the client what to render.
type InstructionKind string

const (
	InstructRedirect InstructionKind = "redirect"
	InstructManual   InstructionKind = "manual"
)

// Instruction is how the payer completes this order.
type Instruction struct {
	Kind       InstructionKind   `json:"kind"`
	URL        string            `json:"url,omitempty"`
	Reference  string            `json:"reference"`
	Steps      []string          `json:"steps,omitempty"`
	Fields     map[string]string `json:"fields,omitempty"`
	NeedsProof bool              `json:"needs_proof"`
}

// Callback is one inbound gateway message, already parsed by the transport.
type Callback struct {
	EventID string
	Kind    string
	Raw     map[string]any
}

// Result is a provider's verdict on a callback.
type Result struct {
	Status      Status
	ProviderRef string
	AmountMinor int64
	Currency    string
}

// Caps describes a provider so the UI and the review queue can adapt.
type Caps struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Currencies  []string `json:"currencies"`
	NeedsReview bool     `json:"needs_review"`
	Redirects   bool     `json:"redirects"`
}

// Provider is the seam. bKash, SSLCommerz, Tap and Stripe implement the same
// two methods; a manual provider returns ErrNotVerifiable from Verify.
type Provider interface {
	Start(ctx context.Context, o Order) (Instruction, error)
	Verify(ctx context.Context, o Order, cb Callback) (Result, error)
	Caps() Caps
}

// Registry resolves providers by name and knows which one to offer by default.
type Registry struct {
	providers map[string]Provider
	fallback  string
}

func NewRegistry(fallback string, providers ...Provider) (*Registry, error) {
	r := &Registry{providers: make(map[string]Provider, len(providers)), fallback: fallback}
	for _, p := range providers {
		r.providers[p.Caps().Name] = p
	}
	if _, ok := r.providers[fallback]; !ok {
		return nil, fmt.Errorf("payment: default provider %q is not registered", fallback)
	}
	return r, nil
}

// Get returns a provider by name, or the default when name is empty.
func (r *Registry) Get(name string) (Provider, error) {
	if name = strings.TrimSpace(name); name == "" {
		name = r.fallback
	}
	p, ok := r.providers[name]
	if !ok {
		return nil, ErrUnknownProvider
	}
	return p, nil
}

func (r *Registry) Capabilities() []Caps {
	out := make([]Caps, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p.Caps())
	}
	return out
}

// NewReference returns a short human-quotable code for a deposit slip.
func NewReference() string {
	raw := rand.Text()
	return "FJ-" + raw[:4] + "-" + raw[4:8]
}
