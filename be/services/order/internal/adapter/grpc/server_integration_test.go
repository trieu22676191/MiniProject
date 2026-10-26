//go:build integration

package grpcadapter

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/adapter/repo"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestGetOrder_AC04_OtherTenantNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ownerPool, err := pgxpool.New(ctx, os.Getenv("MINI_TEST_OWNER_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer ownerPool.Close()
	appPool, err := pgxpool.New(ctx, os.Getenv("MINI_TEST_APP_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ctxA := tenant.WithIdentity(ctx, tenant.Identity{TenantID: tenantA, UserID: "an"})
	if err := db.New(ownerPool).InTx(ctxA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctxA, `INSERT INTO mini_order.orders
			(id, tenant_id, code, status, created_by, total_amount, currency_code)
			VALUES ($1, $2, 'AC04-ORDER', 'PENDING_APPROVAL', 'an', 0, 'VND')`, id, tenantA)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(tenant.UnaryServerInterceptor))
	orderv1.RegisterOrderServiceServer(server, NewServer(usecase.NewOrders(repo.NewOrders(db.New(appPool)), testStock{})))
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := orderv1.NewOrderServiceClient(conn)
	request := &orderv1.GetOrderRequest{Id: id.String()}
	ctxB := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-tenant-id", "22222222-2222-2222-2222-222222222222",
		"x-user-id", "dung", "x-capabilities", "order.create,order.approve",
	))
	_, err = client.GetOrder(ctxB, request)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("tenant B GetOrder: %v; want NOT_FOUND", err)
	}
	ctxChi := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-tenant-id", tenantA.String(), "x-user-id", "chi", "x-capabilities", "order.approve",
	))
	response, err := client.GetOrder(ctxChi, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Order.Id != id.String() || len(response.Order.AllowedActions) != 2 {
		t.Fatalf("unexpected order response: %+v", response.Order)
	}
}

type testStock struct{}

func (testStock) GetItems(context.Context, []string) (map[string]domain.Price, error) {
	return nil, nil
}
