package events

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrUnsupportedType = errors.New("unsupported event type")

type Envelope struct {
	ID            uuid.UUID       `json:"id"`
	Type          string          `json:"type"`
	Source        string          `json:"source"`
	TenantID      uuid.UUID       `json:"tenantId"`
	CorrelationID string          `json:"correlationId"`
	CausationID   *uuid.UUID      `json:"causationId"`
	OccurredAt    string          `json:"occurredAt"`
	SchemaVersion int             `json:"schemaVersion"`
	Data          json.RawMessage `json:"data"`
}

type OrderApproved struct {
	OrderID string `json:"orderId"`
	Lines   []struct {
		SKU      string `json:"sku"`
		Quantity int64  `json:"quantity"`
	} `json:"lines"`
}

type OrderCancelled struct {
	OrderID string `json:"orderId"`
}

type ReservationHeld struct {
	OrderID       string `json:"orderId"`
	ReservationID string `json:"reservationId"`
}

type ReservationFailed struct {
	OrderID   string   `json:"orderId"`
	ShortSKUs []string `json:"shortSkus"`
}

func Decode[T any](payload []byte, expectedType string) (Envelope, T, error) {
	var envelope Envelope
	var data T
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return envelope, data, fmt.Errorf("decode event envelope: %w", err)
	}
	if envelope.ID == uuid.Nil || envelope.TenantID == uuid.Nil || envelope.Type != expectedType || envelope.SchemaVersion != 1 {
		return envelope, data, fmt.Errorf("invalid event envelope for %s", expectedType)
	}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return envelope, data, fmt.Errorf("decode %s data: %w", expectedType, err)
	}
	return envelope, data, nil
}
