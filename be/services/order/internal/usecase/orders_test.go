package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
)

type fakeRepo struct {
	orders   map[string]domain.Order
	created  *domain.Order
	approved *domain.Order
}

func (f *fakeRepo) GetOrder(_ context.Context, id uuid.UUID) (domain.Order, error) {
	if o, ok := f.orders[id.String()]; ok {
		return o, nil
	}
	return domain.Order{}, errors.New("not found")
}

func (f *fakeRepo) ListOrders(_ context.Context, _ int32) ([]domain.Order, error) {
	result := make([]domain.Order, 0, len(f.orders))
	for _, o := range f.orders {
		result = append(result, o)
	}
	return result, nil
}

func (f *fakeRepo) CreateOrder(_ context.Context, o domain.Order) (domain.Order, error) {
	f.created = &o
	if f.orders == nil {
		f.orders = make(map[string]domain.Order)
	}
	f.orders[o.ID] = o
	return o, nil
}

func (f *fakeRepo) ApproveOrder(_ context.Context, id uuid.UUID, actor string) (domain.Order, error) {
	o, ok := f.orders[id.String()]
	if !ok {
		return domain.Order{}, errors.New("not found")
	}
	if err := o.Approve(actor); err != nil {
		return domain.Order{}, err
	}
	f.orders[id.String()] = o
	f.approved = &o
	return o, nil
}

func (f *fakeRepo) RejectOrder(_ context.Context, id uuid.UUID, actor, reason string) (domain.Order, error) {
	o, ok := f.orders[id.String()]
	if !ok {
		return domain.Order{}, errors.New("not found")
	}
	if err := o.Reject(actor, reason); err != nil {
		return domain.Order{}, err
	}
	f.orders[id.String()] = o
	return o, nil
}

func (f *fakeRepo) CancelOrder(_ context.Context, id uuid.UUID, actor string, canApprove bool) (domain.Order, error) {
	o, ok := f.orders[id.String()]
	if !ok {
		return domain.Order{}, errors.New("not found")
	}
	if err := o.Cancel(actor, canApprove); err != nil {
		return domain.Order{}, err
	}
	f.orders[id.String()] = o
	return o, nil
}

func (f *fakeRepo) ProcessReservationResult(_ context.Context, _, orderID uuid.UUID, status string) error {
	o, ok := f.orders[orderID.String()]
	if !ok {
		return errors.New("not found")
	}
	o.Status = status
	f.orders[orderID.String()] = o
	return nil
}

type fakeStock struct {
	prices map[string]domain.Price
}

func (f *fakeStock) GetItems(_ context.Context, skus []string) (map[string]domain.Price, error) {
	result := make(map[string]domain.Price, len(skus))
	for _, sku := range skus {
		price, ok := f.prices[sku]
		if !ok {
			return nil, errors.New("item not found")
		}
		result[sku] = price
	}
	return result, nil
}

func TestCreateOrder_AC01_ServerPricesAndClientPricesIgnored(t *testing.T) {
	repo := &fakeRepo{}
	stock := &fakeStock{
		prices: map[string]domain.Price{
			"BAN-01": {UnitPriceCents: 125_000_000, CurrencyCode: "VND", Available: 10},
			"DEN-01": {UnitPriceCents: 18_000_000, CurrencyCode: "VND", Available: 10},
		},
	}
	uc := NewOrders(repo, stock)
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ctx := tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: tenantID, UserID: "an", Capabilities: map[string]bool{"order.create": true},
	})

	inputs := []domain.LineInput{
		{SKU: "BAN-01", Quantity: 2},
		{SKU: "DEN-01", Quantity: 3},
	}
	order, _, err := uc.CreateOrder(ctx, inputs)
	if err != nil {
		t.Fatalf("CreateOrder() unexpected err: %v", err)
	}
	if order.CreatedBy != "an" {
		t.Fatalf("CreatedBy = %s, want an", order.CreatedBy)
	}
	if order.Status != domain.StatusPendingApproval {
		t.Fatalf("Status = %s, want %s", order.Status, domain.StatusPendingApproval)
	}
	if order.TotalCents != 304_000_000 {
		t.Fatalf("TotalCents = %d, want 304000000", order.TotalCents)
	}
	if len(order.Lines) != 2 {
		t.Fatalf("Lines count = %d, want 2", len(order.Lines))
	}
}

func TestApproveOrder_AC02_PermissionDeniedWithoutCapability(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		capabilities map[string]bool
		wantErr      error
	}{
		{
			name:         "an khong co capability order.approve bi tu choi",
			userID:       "an",
			capabilities: map[string]bool{"order.create": true},
			wantErr:      ErrForbidden,
		},
		{
			name:         "nguoi dung khong co capability nao",
			userID:       "guest",
			capabilities: map[string]bool{},
			wantErr:      ErrForbidden,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orderID := uuid.Must(uuid.NewV7())
			repo := &fakeRepo{
				orders: map[string]domain.Order{
					orderID.String(): {ID: orderID.String(), Status: domain.StatusPendingApproval, CreatedBy: "someone"},
				},
			}
			uc := NewOrders(repo, &fakeStock{})
			ctx := tenant.WithIdentity(context.Background(), tenant.Identity{
				TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				UserID:   tc.userID, Capabilities: tc.capabilities,
			})
			_, _, err := uc.ApproveOrder(ctx, orderID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ApproveOrder() err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestApproveOrder_AC03_SelfApprovalDenied(t *testing.T) {
	orderID := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{
		orders: map[string]domain.Order{
			orderID.String(): {ID: orderID.String(), Status: domain.StatusPendingApproval, CreatedBy: "binh"},
		},
	}
	uc := NewOrders(repo, &fakeStock{})
	ctx := tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:   "binh", Capabilities: map[string]bool{"order.create": true, "order.approve": true},
	})

	_, _, err := uc.ApproveOrder(ctx, orderID)
	if !errors.Is(err, domain.ErrSelfApproval) {
		t.Fatalf("ApproveOrder(self) err = %v, want %v", err, domain.ErrSelfApproval)
	}
	stored := repo.orders[orderID.String()]
	if stored.Status != domain.StatusPendingApproval {
		t.Fatalf("order status = %s, want PENDING_APPROVAL", stored.Status)
	}
}

func TestApproveOrder_AC09_RejectedOrderCannotBeApproved(t *testing.T) {
	orderID := uuid.Must(uuid.NewV7())
	repo := &fakeRepo{
		orders: map[string]domain.Order{
			orderID.String(): {
				ID: orderID.String(), Status: domain.StatusRejected,
				CreatedBy: "an", DecidedBy: "chi", RejectReason: "Het han su dung san pham",
			},
		},
	}
	uc := NewOrders(repo, &fakeStock{})
	ctx := tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:   "chi", Capabilities: map[string]bool{"order.approve": true},
	})

	_, _, err := uc.ApproveOrder(ctx, orderID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("ApproveOrder(rejected) err = %v, want %v", err, domain.ErrInvalidTransition)
	}
	stored := repo.orders[orderID.String()]
	if stored.Status != domain.StatusRejected {
		t.Fatalf("order status = %s, want REJECTED", stored.Status)
	}
}
