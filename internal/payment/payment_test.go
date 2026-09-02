package payment_test

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/payment"
)

func TestBankTransferStart(t *testing.T) {
	p := payment.BankTransfer{
		AccountName: "Darul Uloom", AccountNumber: "1234567890",
		BankName: "Islami Bank", BranchName: "Sylhet",
	}
	order := payment.Order{Reference: "FJ-ABCD-EFGH", AmountMinor: 150000, Currency: "BDT"}

	got, err := p.Start(context.Background(), order)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got.Kind != payment.InstructManual || !got.NeedsProof {
		t.Fatalf("got %+v, want a manual instruction needing proof", got)
	}
	if got.Reference != order.Reference || got.Fields["account_number"] != "1234567890" {
		t.Errorf("got %+v", got)
	}
	if len(got.Steps) == 0 {
		t.Error("payer was given no steps to follow")
	}

	if _, err := (payment.BankTransfer{}).Start(context.Background(), order); err == nil {
		t.Error("start without an account number should fail")
	}
}

func TestBankTransferNeverVerifies(t *testing.T) {
	_, err := payment.BankTransfer{}.Verify(context.Background(), payment.Order{}, payment.Callback{})
	if !errors.Is(err, payment.ErrNotVerifiable) {
		t.Errorf("got %v, want ErrNotVerifiable", err)
	}
}

func TestMoney(t *testing.T) {
	cases := map[int64]string{150000: "BDT 1500.00", 5: "BDT 0.05", 0: "BDT 0.00", -250: "-BDT 2.50"}
	for minor, want := range cases {
		if got := payment.Money(minor, "bdt"); got != want {
			t.Errorf("Money(%d) = %q, want %q", minor, got, want)
		}
	}
}

func TestReferenceShape(t *testing.T) {
	// The schema constrains references; a generated one must satisfy it.
	valid := regexp.MustCompile(`^[A-Z0-9-]{6,32}$`)
	seen := map[string]bool{}
	for range 200 {
		ref := payment.NewReference()
		if !valid.MatchString(ref) {
			t.Fatalf("reference %q does not match the column constraint", ref)
		}
		if seen[ref] {
			t.Fatalf("reference %q was generated twice", ref)
		}
		seen[ref] = true
	}
}

func TestRegistry(t *testing.T) {
	r, err := payment.NewRegistry("bank_transfer", payment.BankTransfer{AccountNumber: "1"})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if _, err := r.Get(""); err != nil {
		t.Errorf("empty name should resolve the default: %v", err)
	}
	if _, err := r.Get("bkash"); !errors.Is(err, payment.ErrUnknownProvider) {
		t.Errorf("got %v, want ErrUnknownProvider", err)
	}
	if _, err := payment.NewRegistry("stripe", payment.BankTransfer{}); err == nil {
		t.Error("an unregistered default should fail")
	}
}
