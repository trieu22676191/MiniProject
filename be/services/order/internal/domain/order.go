package domain

import (
	"errors"

	"github.com/trieu22676191/MiniProject/be/pkg/money"
)

const (
	StatusPendingApproval = "PENDING_APPROVAL"
	StatusApproved        = "APPROVED"
	StatusReserved        = "RESERVED"
	StatusOutOfStock      = "OUT_OF_STOCK"
	StatusRejected        = "REJECTED"
)

var (
	ErrInvalidOrder      = errors.New("invalid order")
	ErrInvalidTransition = errors.New("invalid order state transition")
	ErrSelfApproval      = errors.New("order creator cannot approve their own order")
)

type LineInput struct {
	SKU      string
	Quantity int64
}

type Price struct {
	UnitPriceCents int64
	CurrencyCode   string
}

type Line struct {
	SKU            string
	Quantity       int64
	UnitPriceCents int64
	LineCents      int64
}

type Order struct {
	ID           string
	Code         string
	Status       string
	CreatedBy    string
	DecidedBy    string
	RejectReason string
	TotalCents   int64
	CurrencyCode string
	Lines        []Line
}

func NewOrder(id, creator string, inputs []LineInput, prices map[string]Price) (Order, error) {
	if creator == "" || ValidateInputs(inputs) != nil {
		return Order{}, ErrInvalidOrder
	}
	order := Order{ID: id, Status: StatusPendingApproval, CreatedBy: creator, CurrencyCode: "VND"}
	for _, input := range inputs {
		price, ok := prices[input.SKU]
		if !ok || price.CurrencyCode != order.CurrencyCode || price.UnitPriceCents < 0 {
			return Order{}, ErrInvalidOrder
		}
		if input.Quantity > money.MaxDatabaseCents/max(price.UnitPriceCents, 1) {
			return Order{}, ErrInvalidOrder
		}
		lineCents := input.Quantity * price.UnitPriceCents
		if order.TotalCents > money.MaxDatabaseCents-lineCents {
			return Order{}, ErrInvalidOrder
		}
		order.TotalCents += lineCents
		order.Lines = append(order.Lines, Line{
			SKU: input.SKU, Quantity: input.Quantity,
			UnitPriceCents: price.UnitPriceCents, LineCents: lineCents,
		})
	}
	return order, nil
}

func ValidateInputs(inputs []LineInput) error {
	if len(inputs) == 0 {
		return ErrInvalidOrder
	}
	seen := make(map[string]bool, len(inputs))
	for _, input := range inputs {
		if input.SKU == "" || input.Quantity <= 0 || seen[input.SKU] {
			return ErrInvalidOrder
		}
		seen[input.SKU] = true
	}
	return nil
}

func (o *Order) Approve(actor string) error {
	if o.Status != StatusPendingApproval {
		return ErrInvalidTransition
	}
	if actor == o.CreatedBy {
		return ErrSelfApproval
	}
	o.Status, o.DecidedBy = StatusApproved, actor
	return nil
}

func (o *Order) Reject(actor, reason string) error {
	if o.Status != StatusPendingApproval {
		return ErrInvalidTransition
	}
	if actor == o.CreatedBy {
		return ErrSelfApproval
	}
	if len([]rune(reason)) < 10 || len([]rune(reason)) > 500 {
		return ErrInvalidOrder
	}
	o.Status, o.DecidedBy, o.RejectReason = StatusRejected, actor, reason
	return nil
}

func (o Order) AllowedActions(userID string, canApprove bool) []string {
	if o.Status == StatusPendingApproval && canApprove && o.CreatedBy != userID {
		return []string{"APPROVE", "REJECT"}
	}
	return []string{}
}
