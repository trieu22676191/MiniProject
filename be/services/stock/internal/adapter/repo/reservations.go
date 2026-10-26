package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/events"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	stocksql "github.com/trieu22676191/MiniProject/be/services/stock/internal/adapter/repo/sqlc"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
)

type Reservations struct {
	db *db.DB
}

func NewReservations(database *db.DB) *Reservations {
	return &Reservations{db: database}
}

func (r *Reservations) ReserveOrder(ctx context.Context, eventID uuid.UUID, orderID string, lines []domain.ReservationLine) (domain.ReservationResult, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.ReservationResult{}, err
	}
	parsedOrderID, err := uuid.Parse(orderID)
	if err != nil {
		return domain.ReservationResult{}, fmt.Errorf("parse order ID: %w", err)
	}
	var result domain.ReservationResult
	err = r.db.InTx(ctx, func(tx pgx.Tx) error {
		queries := stocksql.New(tx)
		inserted, err := queries.InsertInbox(ctx, stocksql.InsertInboxParams{
			TenantID: pgUUID(identity.TenantID), Consumer: "stock.reserve", EventID: pgUUID(eventID),
		})
		if err != nil {
			return fmt.Errorf("record stock inbox event: %w", err)
		}
		if inserted == 0 {
			result.Duplicate = true
			return nil
		}
		skus := make([]string, 0, len(lines))
		for _, line := range lines {
			skus = append(skus, line.SKU)
		}
		locked, err := queries.LockStockForReservation(ctx, skus)
		if err != nil {
			return fmt.Errorf("lock stock rows: %w", err)
		}
		levels := make(map[string]domain.StockLevel, len(locked))
		for _, item := range locked {
			levels[item.Sku] = domain.StockLevel{OnHand: item.OnHand, Reserved: item.Reserved}
		}
		result.ShortSKUs = domain.ShortSKUs(lines, levels)
		sort.Strings(result.ShortSKUs)
		if len(result.ShortSKUs) == 0 {
			reservationID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate reservation ID: %w", err)
			}
			if err := queries.InsertReservation(ctx, stocksql.InsertReservationParams{
				ID: pgUUID(reservationID), TenantID: pgUUID(identity.TenantID), OrderID: pgUUID(parsedOrderID),
			}); err != nil {
				return fmt.Errorf("insert reservation: %w", err)
			}
			for _, line := range lines {
				if err := queries.IncreaseReserved(ctx, stocksql.IncreaseReservedParams{
					TenantID: pgUUID(identity.TenantID), Sku: line.SKU, Reserved: line.Quantity,
				}); err != nil {
					return fmt.Errorf("increase reserved stock: %w", err)
				}
				if err := queries.InsertReservationLine(ctx, stocksql.InsertReservationLineParams{
					TenantID: pgUUID(identity.TenantID), ReservationID: pgUUID(reservationID),
					Sku: line.SKU, Quantity: line.Quantity,
				}); err != nil {
					return fmt.Errorf("insert reservation line: %w", err)
				}
			}
			result.ReservationID = reservationID.String()
		}
		if err := insertReservationResult(ctx, queries, identity.TenantID, eventID, parsedOrderID, result); err != nil {
			return err
		}
		return nil
	})
	return result, err
}

func insertReservationResult(ctx context.Context, queries *stocksql.Queries, tenantID, causationID, orderID uuid.UUID, result domain.ReservationResult) error {
	eventID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate reservation result event ID: %w", err)
	}
	var eventType string
	var data any
	if len(result.ShortSKUs) == 0 {
		eventType = "mini.stk.reservation.held"
		data = events.ReservationHeld{OrderID: orderID.String(), ReservationID: result.ReservationID}
	} else {
		eventType = "mini.stk.reservation.failed"
		data = events.ReservationFailed{OrderID: orderID.String(), ShortSKUs: result.ShortSKUs}
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal reservation result: %w", err)
	}
	envelope := events.Envelope{
		ID: eventID, Type: eventType, Source: "stock", TenantID: tenantID,
		CorrelationID: orderID.String(), CausationID: &causationID,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), SchemaVersion: 1, Data: dataJSON,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal reservation envelope: %w", err)
	}
	return queries.InsertStockOutbox(ctx, stocksql.InsertStockOutboxParams{
		ID: pgUUID(eventID), TenantID: pgUUID(tenantID), Type: eventType, Column4: payload,
	})
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
