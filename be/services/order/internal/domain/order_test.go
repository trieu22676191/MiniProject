package domain

import (
	"errors"
	"testing"
)

func TestNewOrder_AC01_TotalUsesCentsAndServerPrices(t *testing.T) {
	tests := []struct {
		name      string
		lines     []LineInput
		prices    map[string]Price
		wantTotal int64
		wantLines []int64
		wantErr   error
	}{
		{
			name: "an tao don 2 dong - tinh dung den xu",
			lines: []LineInput{
				{SKU: "BAN-01", Quantity: 2},
				{SKU: "DEN-01", Quantity: 3},
			},
			prices: map[string]Price{
				"BAN-01": {UnitPriceCents: 125_000_000, CurrencyCode: "VND"},
				"DEN-01": {UnitPriceCents: 18_000_000, CurrencyCode: "VND"},
			},
			wantTotal: 304_000_000,
			wantLines: []int64{250_000_000, 54_000_000},
			wantErr:   nil,
		},
		{
			name: "an tao don 1 dong",
			lines: []LineInput{
				{SKU: "BAN-01", Quantity: 1},
			},
			prices: map[string]Price{
				"BAN-01": {UnitPriceCents: 125_000_000, CurrencyCode: "VND"},
			},
			wantTotal: 125_000_000,
			wantLines: []int64{125_000_000},
			wantErr:   nil,
		},
		{
			name: "thieu gia mat hang thi bao loi",
			lines: []LineInput{
				{SKU: "BAN-01", Quantity: 1},
				{SKU: "DEN-99", Quantity: 1},
			},
			prices: map[string]Price{
				"BAN-01": {UnitPriceCents: 125_000_000, CurrencyCode: "VND"},
			},
			wantErr: ErrInvalidOrder,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			order, err := NewOrder("order-id", "an", tc.lines, tc.prices)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("NewOrder() err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewOrder() unexpected err: %v", err)
			}
			if order.TotalCents != tc.wantTotal {
				t.Fatalf("total cents = %d, want %d", order.TotalCents, tc.wantTotal)
			}
			for i, wantLine := range tc.wantLines {
				if order.Lines[i].LineCents != wantLine {
					t.Fatalf("line %d cents = %d, want %d", i, order.Lines[i].LineCents, wantLine)
				}
			}
		})
	}
}

func TestApprove_AC03_SelfApprovalDenied(t *testing.T) {
	tests := []struct {
		name       string
		creator    string
		actor      string
		wantStatus string
		wantErr    error
	}{
		{
			name:       "binh duyet don do chinh binh lap",
			creator:    "binh",
			actor:      "binh",
			wantStatus: StatusPendingApproval,
			wantErr:    ErrSelfApproval,
		},
		{
			name:       "an duyet don do chinh an lap",
			creator:    "an",
			actor:      "an",
			wantStatus: StatusPendingApproval,
			wantErr:    ErrSelfApproval,
		},
		{
			name:       "chi duyet don do binh lap",
			creator:    "binh",
			actor:      "chi",
			wantStatus: StatusApproved,
			wantErr:    nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			order := Order{Status: StatusPendingApproval, CreatedBy: tc.creator}
			err := order.Approve(tc.actor)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Approve() err = %v, want %v", err, tc.wantErr)
				}
				if order.Status != tc.wantStatus || order.DecidedBy != "" {
					t.Fatalf("self-approval modified order: %+v", order)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if order.Status != tc.wantStatus || order.DecidedBy != tc.actor {
				t.Fatalf("approved order = %+v, want status %s by %s", order, tc.wantStatus, tc.actor)
			}
		})
	}
}

func TestApprove_AC09_RejectedOrderCannotBeApproved(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus string
		actor         string
		wantStatus    string
		wantErr       error
	}{
		{
			name:          "duyet don dang REJECTED",
			initialStatus: StatusRejected,
			actor:         "chi",
			wantStatus:    StatusRejected,
			wantErr:       ErrInvalidTransition,
		},
		{
			name:          "duyet don dang APPROVED",
			initialStatus: StatusApproved,
			actor:         "chi",
			wantStatus:    StatusApproved,
			wantErr:       ErrInvalidTransition,
		},
		{
			name:          "duyet don dang RESERVED",
			initialStatus: StatusReserved,
			actor:         "chi",
			wantStatus:    StatusReserved,
			wantErr:       ErrInvalidTransition,
		},
		{
			name:          "duyet don dang OUT_OF_STOCK",
			initialStatus: StatusOutOfStock,
			actor:         "chi",
			wantStatus:    StatusOutOfStock,
			wantErr:       ErrInvalidTransition,
		},
		{
			name:          "duyet don dang CANCELLED",
			initialStatus: StatusCancelled,
			actor:         "chi",
			wantStatus:    StatusCancelled,
			wantErr:       ErrInvalidTransition,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			order := Order{Status: tc.initialStatus, CreatedBy: "an"}
			err := order.Approve(tc.actor)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Approve() err = %v, want %v", err, tc.wantErr)
			}
			if order.Status != tc.wantStatus {
				t.Fatalf("order status = %s, want %s", order.Status, tc.wantStatus)
			}
		})
	}
}

func TestAllowedActions_AC02_OnlyDifferentApproverSeesActions(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		creator    string
		viewer     string
		canApprove bool
		want       []string
	}{
		{
			name:       "an tao don thi khong thay nut duyet cho an",
			status:     StatusPendingApproval,
			creator:    "an",
			viewer:     "an",
			canApprove: false,
			want:       []string{},
		},
		{
			name:       "binh co quyen nhung la nguoi tao cung khong thay nut",
			status:     StatusPendingApproval,
			creator:    "binh",
			viewer:     "binh",
			canApprove: true,
			want:       []string{},
		},
		{
			name:       "chi khong tao don va co quyen thi thay APPROVE va REJECT",
			status:     StatusPendingApproval,
			creator:    "an",
			viewer:     "chi",
			canApprove: true,
			want:       []string{"APPROVE", "REJECT"},
		},
		{
			name:       "nguoi xem khong co quyen duyet thi khong thay nut",
			status:     StatusPendingApproval,
			creator:    "binh",
			viewer:     "an",
			canApprove: false,
			want:       []string{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			order := Order{Status: tc.status, CreatedBy: tc.creator}
			got := order.AllowedActions(tc.viewer, tc.canApprove)
			if len(got) != len(tc.want) {
				t.Fatalf("AllowedActions() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("AllowedActions()[%d] = %s, want %s", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCancel_EX01_OnlyAuthorizedUsersCanCancelReservedOrder(t *testing.T) {
	owner := Order{Status: StatusReserved, CreatedBy: "an"}
	if err := owner.Cancel("an", false); err != nil || owner.Status != StatusCancelled {
		t.Fatalf("owner cancellation: status=%s err=%v", owner.Status, err)
	}

	forbidden := Order{Status: StatusReserved, CreatedBy: "an"}
	if err := forbidden.Cancel("dung", false); !errors.Is(err, ErrCancelForbidden) {
		t.Fatalf("other user's cancellation error = %v, want %v", err, ErrCancelForbidden)
	}
	if forbidden.Status != StatusReserved {
		t.Fatalf("forbidden cancellation changed status to %s", forbidden.Status)
	}

	pending := Order{Status: StatusPendingApproval, CreatedBy: "an"}
	if err := pending.Cancel("an", false); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("pending cancellation error = %v, want %v", err, ErrInvalidTransition)
	}
	if got := (Order{Status: StatusReserved, CreatedBy: "an"}).AllowedActions("an", false); len(got) != 1 || got[0] != "CANCEL" {
		t.Fatalf("owner actions = %v, want CANCEL", got)
	}
}
