package domain

import "errors"

var (
	ErrInvalidAdjustment = errors.New("stock adjustment must be non-zero")
	ErrBelowReserved     = errors.New("on-hand stock cannot be lower than reserved stock")
	ErrOnHandOverflow    = errors.New("on-hand stock adjustment overflows")
)

type Item struct {
	SKU            string
	Name           string
	UnitPriceCents int64
	CurrencyCode   string
	OnHand         int64
	Reserved       int64
}

func (i Item) Available() int64 {
	return i.OnHand - i.Reserved
}

func AdjustOnHand(onHand, reserved, delta int64) (int64, error) {
	if delta == 0 {
		return 0, ErrInvalidAdjustment
	}
	if delta < 0 && delta < -(onHand-reserved) {
		return 0, ErrBelowReserved
	}
	if delta > 0 && onHand > int64(^uint64(0)>>1)-delta {
		return 0, ErrOnHandOverflow
	}
	return onHand + delta, nil
}
