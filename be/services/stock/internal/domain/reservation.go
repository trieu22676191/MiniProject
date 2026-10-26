package domain

import "errors"

var ErrInvalidReservationEvent = errors.New("invalid reservation event")

type ReservationLine struct {
	SKU      string
	Quantity int64
}

type StockLevel struct {
	OnHand   int64
	Reserved int64
}

type ReservationResult struct {
	Duplicate     bool
	ReservationID string
	ShortSKUs     []string
}

func ValidateReservationLines(lines []ReservationLine) error {
	if len(lines) == 0 {
		return ErrInvalidReservationEvent
	}
	seen := make(map[string]bool, len(lines))
	for _, line := range lines {
		if line.SKU == "" || line.Quantity <= 0 || seen[line.SKU] {
			return ErrInvalidReservationEvent
		}
		seen[line.SKU] = true
	}
	return nil
}

func ShortSKUs(lines []ReservationLine, items map[string]StockLevel) []string {
	short := make([]string, 0)
	for _, line := range lines {
		item, ok := items[line.SKU]
		if !ok || item.OnHand-item.Reserved < line.Quantity {
			short = append(short, line.SKU)
		}
	}
	return short
}
