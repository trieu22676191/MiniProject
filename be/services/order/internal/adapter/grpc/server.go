package grpcadapter

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/money"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	orders *usecase.Orders
}

func NewServer(orders *usecase.Orders) *Server {
	return &Server{orders: orders}
}

func (s *Server) GetOrder(ctx context.Context, request *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}
	order, actions, err := s.orders.GetOrder(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrForbidden):
			return nil, status.Error(codes.PermissionDenied, "order access denied")
		case errors.Is(err, pgx.ErrNoRows):
			return nil, status.Error(codes.NotFound, "order not found")
		default:
			return nil, status.Error(codes.Internal, "could not get order")
		}
	}
	return &orderv1.GetOrderResponse{Order: toProto(order, actions)}, nil
}

func toProto(order domain.Order, actions []string) *orderv1.Order {
	result := &orderv1.Order{
		Id: order.ID, Code: order.Code, Status: order.Status,
		CreatedBy: order.CreatedBy, DecidedBy: order.DecidedBy,
		RejectReason: order.RejectReason,
		TotalAmount: &orderv1.Money{
			Amount: money.FormatCents(order.TotalCents), CurrencyCode: order.CurrencyCode,
		},
		AllowedActions: actions,
	}
	for _, line := range order.Lines {
		result.Lines = append(result.Lines, &orderv1.OrderLine{
			Sku: line.SKU, Quantity: line.Quantity,
			UnitPrice:  &orderv1.Money{Amount: money.FormatCents(line.UnitPriceCents), CurrencyCode: order.CurrencyCode},
			LineAmount: &orderv1.Money{Amount: money.FormatCents(line.LineCents), CurrencyCode: order.CurrencyCode},
		})
	}
	return result
}
