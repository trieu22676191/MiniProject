package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
)

type ReservationRepository interface {
	ReserveOrder(context.Context, uuid.UUID, string, []domain.ReservationLine) (domain.ReservationResult, error)
}

type Reservations struct {
	repo ReservationRepository
}

func NewReservations(repo ReservationRepository) *Reservations {
	return &Reservations{repo: repo}
}

func (u *Reservations) ProcessOrderApproved(ctx context.Context, eventID uuid.UUID, orderID string, lines []domain.ReservationLine) (domain.ReservationResult, error) {
	if _, err := tenant.FromContext(ctx); err != nil {
		return domain.ReservationResult{}, err
	}
	if eventID == uuid.Nil || orderID == "" || domain.ValidateReservationLines(lines) != nil {
		return domain.ReservationResult{}, domain.ErrInvalidReservationEvent
	}
	return u.repo.ReserveOrder(ctx, eventID, orderID, lines)
}
