package usecase

import (
	"context"
	"errors"

	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
)

var ErrForbidden = errors.New("stock access denied")

type ItemRepository interface {
	ListItems(context.Context) ([]domain.Item, error)
	GetItems(context.Context, []string) ([]domain.Item, error)
	AdjustStock(context.Context, string, int64) (domain.Item, error)
}

type Items struct {
	repo ItemRepository
}

func NewItems(repo ItemRepository) *Items {
	return &Items{repo: repo}
}

func (u *Items) ListItems(ctx context.Context) ([]domain.Item, error) {
	return u.repo.ListItems(ctx)
}

func (u *Items) GetItems(ctx context.Context, skus []string) ([]domain.Item, error) {
	return u.repo.GetItems(ctx, skus)
}

func (u *Items) AdjustStock(ctx context.Context, sku string, delta int64) (domain.Item, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Item{}, err
	}
	if !identity.Has("stock.adjust") {
		return domain.Item{}, ErrForbidden
	}
	return u.repo.AdjustStock(ctx, sku, delta)
}
