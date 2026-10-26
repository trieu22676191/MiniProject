package consumer

import (
	"context"

	"github.com/trieu22676191/MiniProject/be/pkg/events"
	"github.com/trieu22676191/MiniProject/be/pkg/inbox"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/usecase"
)

type OrderApproved struct {
	reservations *usecase.Reservations
}

func NewOrderApproved(reservations *usecase.Reservations) *OrderApproved {
	return &OrderApproved{reservations: reservations}
}

func (c *OrderApproved) Handle(ctx context.Context, payload []byte) error {
	envelope, data, err := events.Decode[events.OrderApproved](payload, "mini.ord.order.approved")
	if err != nil {
		return err
	}
	lines := make([]domain.ReservationLine, 0, len(data.Lines))
	for _, line := range data.Lines {
		lines = append(lines, domain.ReservationLine{SKU: line.SKU, Quantity: line.Quantity})
	}
	identity := tenant.Identity{TenantID: envelope.TenantID, UserID: "system"}
	eventCtx := tenant.WithIdentity(ctx, identity)
	_, err = c.reservations.ProcessOrderApproved(eventCtx, envelope.ID, data.OrderID, lines)
	return err
}

var _ inbox.Handler = (*OrderApproved)(nil).Handle
