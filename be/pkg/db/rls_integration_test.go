//go:build integration

package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
)

func TestOrder_AC04_RLSHidesOtherTenant(t *testing.T) {
	ownerURL := os.Getenv("MINI_TEST_OWNER_DATABASE_URL")
	appURL := os.Getenv("MINI_TEST_APP_DATABASE_URL")
	if ownerURL == "" || appURL == "" {
		t.Fatal("test database URLs are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ownerPool, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer ownerPool.Close()
	appPool, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	orderID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenantB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ctxA := tenant.WithIdentity(ctx, tenant.Identity{TenantID: tenantA, UserID: "an"})
	if err := New(ownerPool).InTx(ctxA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctxA, `INSERT INTO mini_order.orders
			(id, tenant_id, code, status, created_by, total_amount, currency_code)
			VALUES ($1, $2, $3, 'PENDING_APPROVAL', 'an', 0, 'VND')`, orderID, tenantA, "AC04-"+orderID.String())
		return err
	}); err != nil {
		t.Fatal(err)
	}
	ctxB := tenant.WithIdentity(ctx, tenant.Identity{TenantID: tenantB, UserID: "dung"})
	if err := New(appPool).InTx(ctxB, func(tx pgx.Tx) error {
		var found uuid.UUID
		return tx.QueryRow(ctxB, "SELECT id FROM mini_order.orders WHERE id = $1", orderID).Scan(&found)
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("tenant B read tenant A order: %v; want pgx.ErrNoRows", err)
	}
}
