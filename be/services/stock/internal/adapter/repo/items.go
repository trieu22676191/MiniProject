package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/money"
	stocksql "github.com/trieu22676191/MiniProject/be/services/stock/internal/adapter/repo/sqlc"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
)

type Items struct {
	db *db.DB
}

func NewItems(database *db.DB) *Items {
	return &Items{db: database}
}

func (r *Items) ListItems(ctx context.Context) ([]domain.Item, error) {
	var result []domain.Item
	err := r.db.InTx(ctx, func(tx pgx.Tx) error {
		rows, err := stocksql.New(tx).ListStockItems(ctx)
		if err != nil {
			return fmt.Errorf("list stock items: %w", err)
		}
		for _, row := range rows {
			item, err := toDomain(row.Sku, row.Name, row.UnitPrice, row.CurrencyCode, row.OnHand, row.Reserved)
			if err != nil {
				return err
			}
			result = append(result, item)
		}
		return nil
	})
	return result, err
}

func (r *Items) GetItems(ctx context.Context, skus []string) ([]domain.Item, error) {
	var result []domain.Item
	err := r.db.InTx(ctx, func(tx pgx.Tx) error {
		rows, err := stocksql.New(tx).GetStockItems(ctx, skus)
		if err != nil {
			return fmt.Errorf("get stock items: %w", err)
		}
		for _, row := range rows {
			item, err := toDomain(row.Sku, row.Name, row.UnitPrice, row.CurrencyCode, row.OnHand, row.Reserved)
			if err != nil {
				return err
			}
			result = append(result, item)
		}
		return nil
	})
	return result, err
}

func (r *Items) AdjustStock(ctx context.Context, sku string, delta int64) (domain.Item, error) {
	var result domain.Item
	err := r.db.InTx(ctx, func(tx pgx.Tx) error {
		queries := stocksql.New(tx)
		locked, err := queries.LockStockForAdjustment(ctx, sku)
		if err != nil {
			return fmt.Errorf("lock stock item for adjustment: %w", err)
		}
		onHand, err := domain.AdjustOnHand(locked.OnHand, locked.Reserved, delta)
		if err != nil {
			return err
		}
		if err := queries.AdjustStockOnHand(ctx, stocksql.AdjustStockOnHandParams{OnHand: onHand, Sku: sku}); err != nil {
			return fmt.Errorf("adjust stock item: %w", err)
		}
		rows, err := queries.GetStockItems(ctx, []string{sku})
		if err != nil {
			return fmt.Errorf("read adjusted stock item: %w", err)
		}
		if len(rows) != 1 {
			return pgx.ErrNoRows
		}
		result, err = toDomain(rows[0].Sku, rows[0].Name, rows[0].UnitPrice, rows[0].CurrencyCode, rows[0].OnHand, rows[0].Reserved)
		return err
	})
	return result, err
}

func toDomain(sku, name, amount, currency string, onHand, reserved int64) (domain.Item, error) {
	cents, err := money.ParseCents(amount)
	if err != nil {
		return domain.Item{}, fmt.Errorf("parse database money: %w", err)
	}
	return domain.Item{
		SKU: sku, Name: name, UnitPriceCents: cents,
		CurrencyCode: strings.TrimSpace(currency), OnHand: onHand, Reserved: reserved,
	}, nil
}
