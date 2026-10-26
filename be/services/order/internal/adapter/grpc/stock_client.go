package grpcadapter

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/money"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type StockClient struct {
	client stockv1.StockServiceClient
}

func NewStockClient(conn grpc.ClientConnInterface) *StockClient {
	return &StockClient{client: stockv1.NewStockServiceClient(conn)}
}

func (c *StockClient) GetItems(ctx context.Context, skus []string) (map[string]domain.Price, error) {
	identity, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	caps := make([]string, 0, len(identity.Capabilities))
	for capability := range identity.Capabilities {
		caps = append(caps, capability)
	}
	sort.Strings(caps)
	outgoing := metadata.AppendToOutgoingContext(ctx,
		"x-tenant-id", identity.TenantID.String(),
		"x-user-id", identity.UserID,
		"x-capabilities", strings.Join(caps, ","),
		"x-service-id", "order",
	)
	callCtx, cancel := context.WithTimeout(outgoing, 3*time.Second)
	defer cancel()
	response, err := c.client.GetItems(callCtx, &stockv1.GetItemsRequest{Skus: skus})
	if err != nil {
		return nil, fmt.Errorf("get prices from stock: %w", err)
	}
	prices := make(map[string]domain.Price, len(response.GetItems()))
	for _, item := range response.GetItems() {
		cents, err := money.ParseCents(item.GetUnitPrice().GetAmount())
		if err != nil {
			return nil, fmt.Errorf("parse stock price: %w", err)
		}
		prices[item.GetSku()] = domain.Price{UnitPriceCents: cents, CurrencyCode: item.GetUnitPrice().GetCurrencyCode(), Available: item.GetAvailable()}
	}
	if len(prices) != len(skus) {
		return nil, fmt.Errorf("stock returned %d prices for %d SKUs", len(prices), len(skus))
	}
	return prices, nil
}
