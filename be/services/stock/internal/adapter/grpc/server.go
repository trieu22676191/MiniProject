package grpcadapter

import (
	"context"
	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/money"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	stockv1.UnimplementedStockServiceServer
	items *usecase.Items
}

func NewServer(items *usecase.Items) *Server {
	return &Server{items: items}
}

func (s *Server) ListItems(ctx context.Context, _ *stockv1.ListItemsRequest) (*stockv1.ListItemsResponse, error) {
	items, err := s.items.ListItems(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list items")
	}
	response := &stockv1.ListItemsResponse{Items: make([]*stockv1.StockItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toProto(item))
	}
	return response, nil
}

func (s *Server) GetItems(ctx context.Context, request *stockv1.GetItemsRequest) (*stockv1.GetItemsResponse, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing identity")
	}
	if identity.ServiceID != "order" {
		return nil, status.Error(codes.PermissionDenied, "GetItems is internal")
	}
	if len(request.GetSkus()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "skus are required")
	}
	items, err := s.items.GetItems(ctx, request.GetSkus())
	if err != nil {
		return nil, status.Error(codes.Internal, "could not get items")
	}
	if len(items) != len(request.GetSkus()) {
		return nil, status.Error(codes.NotFound, "one or more items not found")
	}
	response := &stockv1.GetItemsResponse{Items: make([]*stockv1.StockItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toProto(item))
	}
	return response, nil
}

func toProto(item domain.Item) *stockv1.StockItem {
	return &stockv1.StockItem{
		Sku: item.SKU, Name: item.Name,
		UnitPrice: &stockv1.Money{
			Amount:       money.FormatCents(item.UnitPriceCents),
			CurrencyCode: item.CurrencyCode,
		},
		OnHand: item.OnHand, Reserved: item.Reserved, Available: item.Available(),
	}
}
