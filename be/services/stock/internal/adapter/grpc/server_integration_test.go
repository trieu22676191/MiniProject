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
	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/adapter/repo"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestListItems_AC04_TenantIsolation(t *testing.T) {
	databaseURL := os.Getenv("MINI_TEST_APP_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("MINI_TEST_APP_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tenantA, tenantB := uuid.New(), uuid.New()
	insertStockFixture(t, pool, ctx, tenantA, []string{"A-BAN", "A-GHE", "A-DEN"})
	insertStockFixture(t, pool, ctx, tenantB, []string{"B-BAN", "B-GHE"})
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(tenant.UnaryServerInterceptor))
	stockv1.RegisterStockServiceServer(server, NewServer(usecase.NewItems(repo.NewItems(db.New(pool)))))
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := stockv1.NewStockServiceClient(conn)
	for _, tc := range []struct {
		name, tenantID string
		wantSKUs       []string
	}{
		{"tenant A", tenantA.String(), []string{"A-BAN", "A-GHE", "A-DEN"}},
		{"tenant B", tenantB.String(), []string{"B-BAN", "B-GHE"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requestCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("x-tenant-id", tc.tenantID, "x-user-id", "test"))
			response, err := client.ListItems(requestCtx, &stockv1.ListItemsRequest{})
			if err != nil {
				t.Fatal(err)
			}
			found := make(map[string]bool, len(response.Items))
			for _, item := range response.Items {
				found[item.Sku] = true
				if item.Available != item.OnHand-item.Reserved {
					t.Fatalf("wrong available for %s", item.Sku)
				}
			}
			for _, sku := range tc.wantSKUs {
				if !found[sku] {
					t.Fatalf("tenant %s cannot see its fixture SKU %s", tc.name, sku)
				}
			}
			for sku := range found {
				if len(sku) > 0 && sku[0] != tc.name[len("tenant ")] {
					t.Fatalf("tenant %s can see foreign fixture SKU %s", tc.name, sku)
				}
			}
		})
	}
	userCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-tenant-id", tenantA.String(), "x-user-id", "an",
	))
	_, err = client.GetItems(userCtx, &stockv1.GetItemsRequest{Skus: []string{"BAN-01"}})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("GetItems by user returned %v, want PERMISSION_DENIED", err)
	}
}

func insertStockFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, skus []string) {
	t.Helper()
	identity := tenant.Identity{TenantID: tenantID, UserID: "test"}
	callCtx := tenant.WithIdentity(ctx, identity)
	err := db.New(pool).InTx(callCtx, func(tx pgx.Tx) error {
		for _, sku := range skus {
			if _, err := tx.Exec(callCtx, `INSERT INTO mini_stock.stock_items
				(tenant_id, sku, name, unit_price, currency_code, on_hand, reserved)
				VALUES ($1, $2, $2, 100.00, 'VND', 2, 0)`, tenantID, sku); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
