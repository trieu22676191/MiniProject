package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/events"
	"github.com/trieu22676191/MiniProject/be/pkg/money"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	ordersql "github.com/trieu22676191/MiniProject/be/services/order/internal/adapter/repo/sqlc"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
)

type Orders struct {
	db *db.DB
}

func NewOrders(database *db.DB) *Orders {
	return &Orders{db: database}
}

func (r *Orders) GetOrder(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	var result domain.Order
	err := r.db.InTx(ctx, func(tx pgx.Tx) error {
		var err error
		result, err = getOrder(ctx, ordersql.New(tx), id)
		return err
	})
	return result, err
}

func (r *Orders) ListOrders(ctx context.Context, limit int32) ([]domain.Order, error) {
	var result []domain.Order
	err := r.db.InTx(ctx, func(tx pgx.Tx) error {
		rows, err := ordersql.New(tx).ListOrders(ctx, limit)
		if err != nil {
			return fmt.Errorf("list orders: %w", err)
		}
		for _, row := range rows {
			order, err := mapOrder(row.ID, row.Code, row.Status, row.CreatedBy, row.DecidedBy, row.RejectReason, row.TotalAmount, row.CurrencyCode)
			if err != nil {
				return err
			}
			result = append(result, order)
		}
		return nil
	})
	return result, err
}

func (r *Orders) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	id, err := uuid.Parse(order.ID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("parse order ID: %w", err)
	}
	err = r.db.InTx(ctx, func(tx pgx.Tx) error {
		queries := ordersql.New(tx)
		sequence, err := queries.NextOrderCode(ctx)
		if err != nil {
			return fmt.Errorf("allocate order code: %w", err)
		}
		order.Code = fmt.Sprintf("DH-%06d", sequence)
		if err := queries.InsertOrder(ctx, ordersql.InsertOrderParams{
			ID: pgUUID(id), TenantID: pgUUID(identity.TenantID), Code: order.Code,
			Status: order.Status, CreatedBy: order.CreatedBy,
			Column6: money.FormatCents(order.TotalCents), CurrencyCode: order.CurrencyCode,
		}); err != nil {
			return fmt.Errorf("insert order: %w", err)
		}
		for _, line := range order.Lines {
			lineID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate order line ID: %w", err)
			}
			if err := queries.InsertOrderLine(ctx, ordersql.InsertOrderLineParams{
				ID: pgUUID(lineID), TenantID: pgUUID(identity.TenantID), OrderID: pgUUID(id),
				Sku: line.SKU, Quantity: line.Quantity,
				Column6: money.FormatCents(line.UnitPriceCents), Column7: money.FormatCents(line.LineCents),
			}); err != nil {
				return fmt.Errorf("insert order line: %w", err)
			}
		}
		return nil
	})
	return order, err
}

func (r *Orders) ApproveOrder(ctx context.Context, id uuid.UUID, actor string) (domain.Order, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	var result domain.Order
	err = r.db.InTx(ctx, func(tx pgx.Tx) error {
		queries := ordersql.New(tx)
		row, err := queries.LockOrder(ctx, pgUUID(id))
		if err != nil {
			return fmt.Errorf("lock order for approval: %w", err)
		}
		order, err := mapOrder(row.ID, row.Code, row.Status, row.CreatedBy, row.DecidedBy, row.RejectReason, row.TotalAmount, row.CurrencyCode)
		if err != nil {
			return err
		}
		if err := order.Approve(actor); err != nil {
			return err
		}
		if err := queries.UpdateOrderDecision(ctx, ordersql.UpdateOrderDecisionParams{
			ID: pgUUID(id), Status: order.Status, DecidedBy: pgtype.Text{String: actor, Valid: true},
			RejectReason: pgtype.Text{},
		}); err != nil {
			return fmt.Errorf("update approved order: %w", err)
		}
		result, err = getOrder(ctx, queries, id)
		if err != nil {
			return err
		}
		eventID, payload, err := approvedEvent(identity.TenantID, result)
		if err != nil {
			return err
		}
		return queries.InsertOrderOutbox(ctx, ordersql.InsertOrderOutboxParams{
			ID: pgUUID(eventID), TenantID: pgUUID(identity.TenantID),
			Type: "mini.ord.order.approved", Column4: payload,
		})
	})
	return result, err
}

func (r *Orders) RejectOrder(ctx context.Context, id uuid.UUID, actor, reason string) (domain.Order, error) {
	err := r.db.InTx(ctx, func(tx pgx.Tx) error {
		queries := ordersql.New(tx)
		row, err := queries.LockOrder(ctx, pgUUID(id))
		if err != nil {
			return fmt.Errorf("lock order for rejection: %w", err)
		}
		order, err := mapOrder(row.ID, row.Code, row.Status, row.CreatedBy, row.DecidedBy, row.RejectReason, row.TotalAmount, row.CurrencyCode)
		if err != nil {
			return err
		}
		if err := order.Reject(actor, reason); err != nil {
			return err
		}
		return queries.UpdateOrderDecision(ctx, ordersql.UpdateOrderDecisionParams{
			ID: pgUUID(id), Status: order.Status,
			DecidedBy:    pgtype.Text{String: actor, Valid: true},
			RejectReason: pgtype.Text{String: reason, Valid: true},
		})
	})
	if err != nil {
		return domain.Order{}, err
	}
	return r.GetOrder(ctx, id)
}

func (r *Orders) CancelOrder(ctx context.Context, id uuid.UUID, actor string, canApprove bool) (domain.Order, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	var result domain.Order
	err = r.db.InTx(ctx, func(tx pgx.Tx) error {
		queries := ordersql.New(tx)
		row, err := queries.LockOrder(ctx, pgUUID(id))
		if err != nil {
			return fmt.Errorf("lock order for cancellation: %w", err)
		}
		order, err := mapOrder(row.ID, row.Code, row.Status, row.CreatedBy, row.DecidedBy, row.RejectReason, row.TotalAmount, row.CurrencyCode)
		if err != nil {
			return err
		}
		if err := order.Cancel(actor, canApprove); err != nil {
			return err
		}
		if err := queries.SetOrderStatus(ctx, ordersql.SetOrderStatusParams{ID: pgUUID(id), Status: order.Status}); err != nil {
			return fmt.Errorf("cancel order: %w", err)
		}
		result, err = getOrder(ctx, queries, id)
		if err != nil {
			return err
		}
		eventID, payload, err := cancelledEvent(identity.TenantID, id)
		if err != nil {
			return err
		}
		return queries.InsertOrderOutbox(ctx, ordersql.InsertOrderOutboxParams{
			ID: pgUUID(eventID), TenantID: pgUUID(identity.TenantID),
			Type: "mini.ord.order.cancelled", Column4: payload,
		})
	})
	return result, err
}

func (r *Orders) ProcessReservationResult(ctx context.Context, eventID, orderID uuid.UUID, state string) error {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return err
	}
	return r.db.InTx(ctx, func(tx pgx.Tx) error {
		queries := ordersql.New(tx)
		inserted, err := queries.InsertInbox(ctx, ordersql.InsertInboxParams{
			TenantID: pgUUID(identity.TenantID), Consumer: "order.reservation_result", EventID: pgUUID(eventID),
		})
		if err != nil {
			return fmt.Errorf("record order inbox event: %w", err)
		}
		if inserted == 0 {
			return nil
		}
		if _, err := queries.ApplyReservationResult(ctx, ordersql.ApplyReservationResultParams{
			ID: pgUUID(orderID), Status: state,
		}); err != nil {
			return fmt.Errorf("apply reservation result: %w", err)
		}
		return nil
	})
}

func getOrder(ctx context.Context, queries *ordersql.Queries, id uuid.UUID) (domain.Order, error) {
	row, err := queries.GetOrderByID(ctx, pgUUID(id))
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order: %w", err)
	}
	order, err := mapOrder(row.ID, row.Code, row.Status, row.CreatedBy, row.DecidedBy, row.RejectReason, row.TotalAmount, row.CurrencyCode)
	if err != nil {
		return domain.Order{}, err
	}
	lines, err := queries.ListOrderLines(ctx, pgUUID(id))
	if err != nil {
		return domain.Order{}, fmt.Errorf("list order lines: %w", err)
	}
	for _, row := range lines {
		unit, err := money.ParseCents(row.UnitPrice)
		if err != nil {
			return domain.Order{}, fmt.Errorf("parse unit price: %w", err)
		}
		line, err := money.ParseCents(row.LineAmount)
		if err != nil {
			return domain.Order{}, fmt.Errorf("parse line amount: %w", err)
		}
		order.Lines = append(order.Lines, domain.Line{SKU: row.Sku, Quantity: row.Quantity, UnitPriceCents: unit, LineCents: line})
	}
	return order, nil
}

func mapOrder(id pgtype.UUID, code, state, creator string, decided, reason pgtype.Text, total, currency string) (domain.Order, error) {
	totalCents, err := money.ParseCents(total)
	if err != nil {
		return domain.Order{}, fmt.Errorf("parse order total: %w", err)
	}
	return domain.Order{
		ID: uuid.UUID(id.Bytes).String(), Code: code, Status: state, CreatedBy: creator,
		DecidedBy: decided.String, RejectReason: reason.String,
		TotalCents: totalCents, CurrencyCode: strings.TrimSpace(currency),
	}, nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

type approvedEnvelope struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Source        string    `json:"source"`
	TenantID      string    `json:"tenantId"`
	CorrelationID string    `json:"correlationId"`
	CausationID   any       `json:"causationId"`
	OccurredAt    time.Time `json:"occurredAt"`
	SchemaVersion int       `json:"schemaVersion"`
	Data          struct {
		OrderID string `json:"orderId"`
		Lines   []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		} `json:"lines"`
	} `json:"data"`
}

func approvedEvent(tenantID uuid.UUID, order domain.Order) (uuid.UUID, []byte, error) {
	eventID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("generate event ID: %w", err)
	}
	event := approvedEnvelope{
		ID: eventID.String(), Type: "mini.ord.order.approved", Source: "order",
		TenantID: tenantID.String(), CorrelationID: order.ID, CausationID: nil,
		OccurredAt: time.Now().UTC(), SchemaVersion: 1,
	}
	event.Data.OrderID = order.ID
	for _, line := range order.Lines {
		event.Data.Lines = append(event.Data.Lines, struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		}{SKU: line.SKU, Quantity: line.Quantity})
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("marshal approved event: %w", err)
	}
	return eventID, payload, nil
}

func cancelledEvent(tenantID, orderID uuid.UUID) (uuid.UUID, []byte, error) {
	eventID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("generate cancellation event ID: %w", err)
	}
	data, err := json.Marshal(events.OrderCancelled{OrderID: orderID.String()})
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("marshal cancellation event data: %w", err)
	}
	envelope := events.Envelope{
		ID: eventID, Type: "mini.ord.order.cancelled", Source: "order", TenantID: tenantID,
		CorrelationID: orderID.String(), OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		SchemaVersion: 1, Data: data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("marshal cancellation event envelope: %w", err)
	}
	return eventID, payload, nil
}
