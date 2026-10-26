package consumer

import (
	"context"

	"github.com/google/uuid"
	"github.com/trieu22676191/MiniProject/be/pkg/events"
	"github.com/trieu22676191/MiniProject/be/pkg/inbox"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/usecase"
)

type OrderCancelled struct {
	reservations *usecase.Reservations
}

func NewOrderCancelled(reservations *usecase.Reservations) *OrderCancelled {
	return &OrderCancelled{reservations: reservations}
}

func (c *OrderCancelled) Handle(ctx context.Context, payload []byte) error {
	envelope, data, err := events.Decode[events.OrderCancelled](payload, "mini.ord.order.cancelled")
	if err != nil {
		return err
	}
	orderID, err := uuid.Parse(data.OrderID)
	if err != nil {
		return err
	}
	identity := tenant.Identity{TenantID: envelope.TenantID, UserID: "system"}
	return c.reservations.ProcessOrderCancelled(tenant.WithIdentity(ctx, identity), envelope.ID, orderID)
}

var _ inbox.Handler = (*OrderCancelled)(nil).Handle
