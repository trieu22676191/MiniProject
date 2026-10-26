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
	ListOrders(context.Context, int32) ([]domain.Order, error)
	CreateOrder(context.Context, domain.Order) (domain.Order, error)
	ApproveOrder(context.Context, uuid.UUID, string) (domain.Order, error)
	RejectOrder(context.Context, uuid.UUID, string, string) (domain.Order, error)
	ProcessReservationResult(context.Context, uuid.UUID, uuid.UUID, string) error
}

type Stock interface {
	GetItems(context.Context, []string) (map[string]domain.Price, error)
}

type Orders struct {
	repo  Repository
	stock Stock
}

func NewOrders(repo Repository, stock Stock) *Orders {
	return &Orders{repo: repo, stock: stock}
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

func (u *Orders) ListOrders(ctx context.Context) ([]domain.Order, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if !identity.Has("order.create") && !identity.Has("order.approve") {
		return nil, ErrForbidden
	}
	return u.repo.ListOrders(ctx, 50)
}

func (u *Orders) CreateOrder(ctx context.Context, inputs []domain.LineInput) (domain.Order, []string, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if !identity.Has("order.create") {
		return domain.Order{}, nil, ErrForbidden
	}
	if err := domain.ValidateInputs(inputs); err != nil {
		return domain.Order{}, nil, err
	}
	skus := make([]string, 0, len(inputs))
	for _, input := range inputs {
		skus = append(skus, input.SKU)
	}
	prices, err := u.stock.GetItems(ctx, skus)
	if err != nil {
		return domain.Order{}, nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Order{}, nil, err
	}
	order, err := domain.NewOrder(id.String(), identity.UserID, inputs, prices)
	if err != nil {
		return domain.Order{}, nil, err
	}
	created, err := u.repo.CreateOrder(ctx, order)
	if err != nil {
		return domain.Order{}, nil, err
	}
	return created, created.AllowedActions(identity.UserID, identity.Has("order.approve")), nil
}

func (u *Orders) ApproveOrder(ctx context.Context, id uuid.UUID) (domain.Order, []string, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if !identity.Has("order.approve") {
		return domain.Order{}, nil, ErrForbidden
	}
	order, err := u.repo.ApproveOrder(ctx, id, identity.UserID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	return order, order.AllowedActions(identity.UserID, true), nil
}

func (u *Orders) RejectOrder(ctx context.Context, id uuid.UUID, reason string) (domain.Order, []string, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if !identity.Has("order.approve") {
		return domain.Order{}, nil, ErrForbidden
	}
	order, err := u.repo.RejectOrder(ctx, id, identity.UserID, reason)
	if err != nil {
		return domain.Order{}, nil, err
	}
	return order, order.AllowedActions(identity.UserID, true), nil
}

func (u *Orders) ProcessReservationResult(ctx context.Context, eventID, orderID uuid.UUID, status string) error {
	if _, err := tenant.FromContext(ctx); err != nil {
		return err
	}
	if status != domain.StatusReserved && status != domain.StatusOutOfStock {
		return domain.ErrInvalidOrder
	}
	return u.repo.ProcessReservationResult(ctx, eventID, orderID, status)
}
