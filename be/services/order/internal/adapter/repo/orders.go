package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/money"
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
		queries := ordersql.New(tx)
		pgID := pgtype.UUID{Bytes: [16]byte(id), Valid: true}
		row, err := queries.GetOrderByID(ctx, pgID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}
		result.TotalCents, err = money.ParseCents(row.TotalAmount)
		if err != nil {
			return fmt.Errorf("parse order total: %w", err)
		}
		result.ID, result.Code, result.Status = id.String(), row.Code, row.Status
		result.CreatedBy, result.DecidedBy = row.CreatedBy, row.DecidedBy.String
		result.RejectReason = row.RejectReason.String
		result.CurrencyCode = strings.TrimSpace(row.CurrencyCode)
		lines, err := queries.ListOrderLines(ctx, pgID)
		if err != nil {
			return fmt.Errorf("list order lines: %w", err)
		}
		for _, row := range lines {
			unit, err := money.ParseCents(row.UnitPrice)
			if err != nil {
				return fmt.Errorf("parse unit price: %w", err)
			}
			line, err := money.ParseCents(row.LineAmount)
			if err != nil {
				return fmt.Errorf("parse line amount: %w", err)
			}
			result.Lines = append(result.Lines, domain.Line{SKU: row.Sku, Quantity: row.Quantity, UnitPriceCents: unit, LineCents: line})
		}
		return nil
	})
	return result, err
}
