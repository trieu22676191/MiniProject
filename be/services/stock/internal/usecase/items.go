package usecase

import (
	"context"

	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
)

type ItemRepository interface {
	ListItems(context.Context) ([]domain.Item, error)
	GetItems(context.Context, []string) ([]domain.Item, error)
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
