package grpcadapter

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/money"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/usecase"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
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
		return nil, toStatus(err)
	}
	return &orderv1.GetOrderResponse{Order: toProto(order, actions)}, nil
}

func (s *Server) CreateOrder(ctx context.Context, request *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	inputs := make([]domain.LineInput, 0, len(request.GetLines()))
	for _, line := range request.GetLines() {
		inputs = append(inputs, domain.LineInput{SKU: line.GetSku(), Quantity: line.GetQuantity()})
	}
	order, actions, err := s.orders.CreateOrder(ctx, inputs)
	if err != nil {
		return nil, toStatus(err)
	}
	return &orderv1.CreateOrderResponse{Order: toProto(order, actions)}, nil
}

func (s *Server) ListOrders(ctx context.Context, _ *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	orders, err := s.orders.ListOrders(ctx)
	if err != nil {
		return nil, toStatus(err)
	}
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, toStatus(err)
	}
	response := &orderv1.ListOrdersResponse{Orders: make([]*orderv1.Order, 0, len(orders))}
	for _, order := range orders {
		actions := order.AllowedActions(identity.UserID, identity.Has("order.approve"))
		response.Orders = append(response.Orders, toProto(order, actions))
	}
	return response, nil
}

func (s *Server) ApproveOrder(ctx context.Context, request *orderv1.ApproveOrderRequest) (*orderv1.ApproveOrderResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}
	order, actions, err := s.orders.ApproveOrder(ctx, id)
	if err != nil {
		return nil, toStatus(err)
	}
	return &orderv1.ApproveOrderResponse{Order: toProto(order, actions)}, nil
}

func (s *Server) RejectOrder(ctx context.Context, request *orderv1.RejectOrderRequest) (*orderv1.RejectOrderResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}
	order, actions, err := s.orders.RejectOrder(ctx, id, request.GetReason())
	if err != nil {
		return nil, toStatus(err)
	}
	return &orderv1.RejectOrderResponse{Order: toProto(order, actions)}, nil
}

func (s *Server) CancelOrder(ctx context.Context, request *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}
	order, actions, err := s.orders.CancelOrder(ctx, id)
	if err != nil {
		return nil, toStatus(err)
	}
	return &orderv1.CancelOrderResponse{Order: toProto(order, actions)}, nil
}

func toStatus(err error) error {
	code, reason, message := codes.Internal, "INTERNAL", "could not process order"
	switch {
	case errors.Is(err, tenant.ErrMissing):
		code, reason, message = codes.Unauthenticated, "UNAUTHENTICATED", "missing identity"
	case errors.Is(err, usecase.ErrForbidden):
		code, reason, message = codes.PermissionDenied, "PERMISSION_DENIED", "permission denied"
	case errors.Is(err, domain.ErrSelfApproval):
		code, reason, message = codes.PermissionDenied, "SELF_APPROVAL", "you cannot decide your own order"
	case errors.Is(err, domain.ErrCancelForbidden):
		code, reason, message = codes.PermissionDenied, "CANCELLATION_FORBIDDEN", "you cannot cancel this order"
	case errors.Is(err, domain.ErrInvalidTransition):
		code, reason, message = codes.FailedPrecondition, "INVALID_TRANSITION", "order state does not allow this action"
	case errors.Is(err, domain.ErrInvalidOrder):
		code, reason, message = codes.InvalidArgument, "INVALID_ARGUMENT", "invalid order input"
	case errors.Is(err, domain.ErrInsufficientStock):
		code, reason, message = codes.FailedPrecondition, "INSUFFICIENT_STOCK", "requested quantity exceeds available stock"
	case errors.Is(err, pgx.ErrNoRows):
		code, reason, message = codes.NotFound, "NOT_FOUND", "order not found"
	default:
		var grpcStatus interface{ GRPCStatus() *status.Status }
		if errors.As(err, &grpcStatus) {
			upstream := grpcStatus.GRPCStatus()
			if upstream.Code() != codes.Unknown {
				return upstream.Err()
			}
		}
	}
	result := status.New(code, message)
	withDetails, detailErr := result.WithDetails(&errdetails.ErrorInfo{Reason: reason, Domain: "neo-mini"})
	if detailErr != nil {
		return result.Err()
	}
	return withDetails.Err()
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
