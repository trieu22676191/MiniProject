//go:build integration

package grpcadapter

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

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
		wantCount      int
		wantDEN        bool
	}{
		{"tenant A", "11111111-1111-1111-1111-111111111111", 3, true},
		{"tenant B", "22222222-2222-2222-2222-222222222222", 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requestCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("x-tenant-id", tc.tenantID, "x-user-id", "test"))
			response, err := client.ListItems(requestCtx, &stockv1.ListItemsRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if len(response.Items) != tc.wantCount {
				t.Fatalf("item count = %d, want %d", len(response.Items), tc.wantCount)
			}
			foundDEN := false
			for _, item := range response.Items {
				foundDEN = foundDEN || item.Sku == "DEN-01"
				if item.Available != item.OnHand-item.Reserved {
					t.Fatalf("wrong available for %s", item.Sku)
				}
			}
			if foundDEN != tc.wantDEN {
				t.Fatalf("DEN-01 visible = %v, want %v", foundDEN, tc.wantDEN)
			}
		})
	}
	userCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-tenant-id", "11111111-1111-1111-1111-111111111111", "x-user-id", "an",
	))
	_, err = client.GetItems(userCtx, &stockv1.GetItemsRequest{Skus: []string{"BAN-01"}})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("GetItems by user returned %v, want PERMISSION_DENIED", err)
	}
}
