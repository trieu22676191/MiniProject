package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
)

var ErrForbidden = errors.New("order access denied")

type Repository interface {
	GetOrder(context.Context, uuid.UUID) (domain.Order, error)
}

type Orders struct {
	repo Repository
}

func NewOrders(repo Repository) *Orders {
	return &Orders{repo: repo}
}

func (u *Orders) GetOrder(ctx context.Context, id uuid.UUID) (domain.Order, []string, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if !identity.Has("order.create") && !identity.Has("order.approve") {
		return domain.Order{}, nil, ErrForbidden
	}
	order, err := u.repo.GetOrder(ctx, id)
	if err != nil {
		return domain.Order{}, nil, err
	}
	return order, order.AllowedActions(identity.UserID, identity.Has("order.approve")), nil
}
