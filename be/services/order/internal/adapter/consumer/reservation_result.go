package consumer

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/trieu22676191/MiniProject/be/pkg/events"
	"github.com/trieu22676191/MiniProject/be/pkg/inbox"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/usecase"
)

type ReservationResult struct {
	orders *usecase.Orders
}

func NewReservationResult(orders *usecase.Orders) *ReservationResult {
	return &ReservationResult{orders: orders}
}

func (c *ReservationResult) Handle(ctx context.Context, payload []byte) error {
	var header events.Envelope
	if err := json.Unmarshal(payload, &header); err != nil {
		return err
	}
	var orderID string
	var state string
	switch header.Type {
	case "mini.stk.reservation.held":
		_, data, err := events.Decode[events.ReservationHeld](payload, header.Type)
		if err != nil {
			return err
		}
		orderID, state = data.OrderID, "RESERVED"
	case "mini.stk.reservation.failed":
		_, data, err := events.Decode[events.ReservationFailed](payload, header.Type)
		if err != nil {
			return err
		}
		orderID, state = data.OrderID, "OUT_OF_STOCK"
	default:
		return events.ErrUnsupportedType
	}
	parsedOrderID, err := uuid.Parse(orderID)
	if err != nil {
		return err
	}
	identity := tenant.Identity{TenantID: header.TenantID, UserID: "system"}
	return c.orders.ProcessReservationResult(tenant.WithIdentity(ctx, identity), header.ID, parsedOrderID, state)
}

var _ inbox.Handler = (*ReservationResult)(nil).Handle
