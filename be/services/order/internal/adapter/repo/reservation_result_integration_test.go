//go:build integration

package repo

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func TestCancelOrder_EX01_EmitsCompensationInSameTransaction(t *testing.T) {
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
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ctx = tenant.WithIdentity(ctx, tenant.Identity{TenantID: tenantID, UserID: "an"})
	repository := NewOrders(db.New(database))
	orderID := uuid.Must(uuid.NewV7())
	if _, err := repository.CreateOrder(ctx, domain.Order{
		ID: orderID.String(), Status: domain.StatusApproved, CreatedBy: "an", CurrencyCode: "VND",
	}); err != nil {
		t.Fatal(err)
	}
	if err := repository.ProcessReservationResult(ctx, uuid.Must(uuid.NewV7()), orderID, domain.StatusReserved); err != nil {
		t.Fatal(err)
	}
	order, err := repository.CancelOrder(ctx, orderID, "an", false)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.StatusCancelled {
		t.Fatalf("cancelled status = %s, want %s", order.Status, domain.StatusCancelled)
	}
	var eventType string
	var payload []byte
	if err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT type, payload FROM mini_order.outbox
			WHERE tenant_id = $1 AND type = 'mini.ord.order.cancelled'
			AND payload->>'correlationId' = $2`, tenantID, orderID.String()).Scan(&eventType, &payload)
	}); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Type string `json:"type"`
		Data struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	if eventType != "mini.ord.order.cancelled" || envelope.Type != eventType || envelope.Data.OrderID != orderID.String() {
		t.Fatalf("cancellation outbox event mismatch: type=%s envelope=%+v", eventType, envelope)
	}
	if _, err := repository.CancelOrder(ctx, orderID, "an", false); err != domain.ErrInvalidTransition {
		t.Fatalf("second cancellation error = %v, want invalid transition", err)
	}
}
