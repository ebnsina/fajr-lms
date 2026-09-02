package payment

import (
	"context"
	"fmt"
	"strings"
)

// BankTransfer settles by human review: the payer deposits into a named account,
// uploads the slip, and a member of staff approves it. No gateway, no fees.
type BankTransfer struct {
	AccountName   string
	AccountNumber string
	BankName      string
	BranchName    string
	Instructions  []string
	Currencies    []string
}

func (b BankTransfer) Caps() Caps {
	currencies := b.Currencies
	if len(currencies) == 0 {
		currencies = []string{"BDT", "PKR", "AED", "SAR", "USD"}
	}
	return Caps{
		Name: "bank_transfer", Title: "Bank transfer or mobile wallet",
		Currencies: currencies, NeedsReview: true,
	}
}

func (b BankTransfer) Start(_ context.Context, o Order) (Instruction, error) {
	if strings.TrimSpace(b.AccountNumber) == "" {
		return Instruction{}, fmt.Errorf("payment: bank transfer needs an account number configured")
	}

	steps := b.Instructions
	if len(steps) == 0 {
		steps = []string{
			fmt.Sprintf("Send %s to the account below.", Money(o.AmountMinor, o.Currency)),
			fmt.Sprintf("Write the reference %s on the deposit slip or in the payment note.", o.Reference),
			"Upload a photo of the slip or a screenshot of the transfer.",
			"Your place is confirmed once a member of staff approves it.",
		}
	}

	return Instruction{
		Kind: InstructManual, Reference: o.Reference, Steps: steps, NeedsProof: true,
		Fields: map[string]string{
			"account_name": b.AccountName, "account_number": b.AccountNumber,
			"bank": b.BankName, "branch": b.BranchName,
			"amount": Money(o.AmountMinor, o.Currency),
		},
	}, nil
}

// Verify is never called: this provider has no gateway to hear from.
func (BankTransfer) Verify(context.Context, Order, Callback) (Result, error) {
	return Result{}, ErrNotVerifiable
}

// Money renders minor units for display. The frontend uses Intl for the real
// thing; this is for slips and SMS, where no formatter is available.
func Money(minor int64, currency string) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	return fmt.Sprintf("%s%s %d.%02d", sign, strings.ToUpper(currency), minor/100, minor%100)
}
