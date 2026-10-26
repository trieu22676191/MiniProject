package grpcadapter

import (
	"context"
	"testing"

	"github.com/google/uuid"
	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAdjustStockRequiresCapability(t *testing.T) {
	repository := &fakeItemRepository{}
	server := NewServer(usecase.NewItems(repository))
	ctx := tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), UserID: "viewer",
	})
	_, err := server.AdjustStock(ctx, &stockv1.AdjustStockRequest{Sku: "BAN-01", DeltaOnHand: 1})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("AdjustStock error = %v, want PERMISSION_DENIED", err)
	}
	if repository.adjusted {
		t.Fatal("repository was called without stock.adjust capability")
	}
}

func TestAdjustStockReturnsUpdatedItem(t *testing.T) {
	repository := &fakeItemRepository{item: domain.Item{
		SKU: "BAN-01", Name: "Bàn", UnitPriceCents: 10000, CurrencyCode: "VND", OnHand: 5, Reserved: 2,
	}}
	server := NewServer(usecase.NewItems(repository))
	ctx := tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), UserID: "operator",
		Capabilities: map[string]bool{"stock.adjust": true},
	})
	response, err := server.AdjustStock(ctx, &stockv1.AdjustStockRequest{Sku: "BAN-01", DeltaOnHand: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.adjusted || response.GetItem().GetOnHand() != 5 || response.GetItem().GetAvailable() != 3 {
		t.Fatalf("unexpected adjustment response: %+v", response)
	}
}

type fakeItemRepository struct {
	adjusted bool
	item     domain.Item
}

func (*fakeItemRepository) ListItems(context.Context) ([]domain.Item, error) { return nil, nil }
func (*fakeItemRepository) GetItems(context.Context, []string) ([]domain.Item, error) {
	return nil, nil
}
func (f *fakeItemRepository) AdjustStock(_ context.Context, _ string, _ int64) (domain.Item, error) {
	f.adjusted = true
	return f.item, nil
}
