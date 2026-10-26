package domain

import "testing"

func TestNewOrder_AC01_TotalUsesCentsAndServerPrices(t *testing.T) {
	order, err := NewOrder("order-id", "an", []LineInput{
		{SKU: "BAN-01", Quantity: 2},
		{SKU: "DEN-01", Quantity: 3},
	}, map[string]Price{
		"BAN-01": {UnitPriceCents: 125_000_000, CurrencyCode: "VND"},
		"DEN-01": {UnitPriceCents: 18_000_000, CurrencyCode: "VND"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.TotalCents != 304_000_000 {
		t.Fatalf("total cents = %d, want %d", order.TotalCents, 304_000_000)
	}
	if order.Lines[0].LineCents != 250_000_000 || order.Lines[1].LineCents != 54_000_000 {
		t.Fatalf("line totals = %d, %d; want 250000000, 54000000", order.Lines[0].LineCents, order.Lines[1].LineCents)
	}
}

func TestApprove_AC03_SelfApprovalDenied(t *testing.T) {
	order := Order{Status: StatusPendingApproval, CreatedBy: "binh"}
	if err := order.Approve("binh"); err != ErrSelfApproval {
		t.Fatalf("Approve(self) error = %v, want %v", err, ErrSelfApproval)
	}
	if order.Status != StatusPendingApproval || order.DecidedBy != "" {
		t.Fatalf("self-approval changed order: %+v", order)
	}
}

func TestApprove_AC09_RejectedOrderCannotBeApproved(t *testing.T) {
	order := Order{Status: StatusRejected, CreatedBy: "an", DecidedBy: "chi"}
	if err := order.Approve("dung"); err != ErrInvalidTransition {
		t.Fatalf("Approve(rejected) error = %v, want %v", err, ErrInvalidTransition)
	}
	if order.Status != StatusRejected || order.DecidedBy != "chi" {
		t.Fatalf("invalid transition changed order: %+v", order)
	}
}

func TestAllowedActions_AC02_OnlyDifferentApproverSeesActions(t *testing.T) {
	order := Order{Status: StatusPendingApproval, CreatedBy: "an"}
	if got := order.AllowedActions("an", true); len(got) != 0 {
		t.Fatalf("creator allowed actions = %v, want none", got)
	}
	if got := order.AllowedActions("chi", false); len(got) != 0 {
		t.Fatalf("user without capability allowed actions = %v, want none", got)
	}
	got := order.AllowedActions("chi", true)
	if len(got) != 2 || got[0] != "APPROVE" || got[1] != "REJECT" {
		t.Fatalf("approver actions = %v, want APPROVE and REJECT", got)
	}
}
