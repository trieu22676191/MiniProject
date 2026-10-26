//go:build integration

package repo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
)

func TestReservationResult_AC05_AC06_UpdatesApprovedOrderOnce(t *testing.T) {
	url := os.Getenv("MINI_TEST_APP_DATABASE_URL")
	if url == "" {
		t.Fatal("MINI_TEST_APP_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	identity := tenant.Identity{
		TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:   "system",
	}
	ctx = tenant.WithIdentity(ctx, identity)
	repository := NewOrders(db.New(database))

	for _, tc := range []struct {
		name, firstStatus, duplicateStatus string
	}{
		{"held", domain.StatusReserved, domain.StatusOutOfStock},
		{"shortage", domain.StatusOutOfStock, domain.StatusReserved},
	} {
		t.Run(tc.name, func(t *testing.T) {
			orderID, err := uuid.NewV7()
			if err != nil {
				t.Fatal(err)
			}
			order := domain.Order{
				ID: orderID.String(), Status: domain.StatusApproved,
				CreatedBy: "an", CurrencyCode: "VND",
			}
			if _, err := repository.CreateOrder(ctx, order); err != nil {
				t.Fatal(err)
			}
			eventID, err := uuid.NewV7()
			if err != nil {
				t.Fatal(err)
			}
			if err := repository.ProcessReservationResult(ctx, eventID, orderID, tc.firstStatus); err != nil {
				t.Fatal(err)
			}
			if err := repository.ProcessReservationResult(ctx, eventID, orderID, tc.duplicateStatus); err != nil {
				t.Fatal(err)
			}
			got, err := repository.GetOrder(ctx, orderID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.firstStatus {
				t.Fatalf("order status = %s, want %s", got.Status, tc.firstStatus)
			}
		})
	}
}
