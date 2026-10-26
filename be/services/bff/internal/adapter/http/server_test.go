package httpadapter

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestBFF_AC02_RequiresKnownDevUserAndForwardsIdentity(t *testing.T) {
	orders := &fakeOrderClient{}
	handler := newTestHandler(orders)

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	request.Header.Set("X-Dev-User", "chi")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if orders.metadata.Get("x-tenant-id")[0] != "11111111-1111-1111-1111-111111111111" || orders.metadata.Get("x-user-id")[0] != "chi" || orders.metadata.Get("x-capabilities")[0] != "order.approve" {
		t.Fatalf("unexpected forwarded metadata: %v", orders.metadata)
	}
	var body struct {
		Orders []map[string]any `json:"orders"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Orders) != 1 || body.Orders[0]["createdBy"] != "an" || body.Orders[0]["allowedActions"] == nil {
		t.Fatalf("unexpected camelCase order response: %s", response.Body.String())
	}
}

func TestBFF_AC01_CreateOrderPassesOnlySKUAndQuantity(t *testing.T) {
	orders := &fakeOrderClient{}
	handler := newTestHandler(orders)
	request := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"lines":[{"sku":"BAN-01","quantity":2}]}`))
	request.Header.Set("X-Dev-User", "an")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
	}
	if len(orders.created.GetLines()) != 1 || orders.created.GetLines()[0].GetSku() != "BAN-01" || orders.created.GetLines()[0].GetQuantity() != 2 {
		t.Fatalf("unexpected gRPC request: %+v", orders.created)
	}
}

func TestBFF_AC03_MapsSelfApprovalToForbiddenJSON(t *testing.T) {
	orders := &fakeOrderClient{approveError: selfApprovalStatus()}
	handler := newTestHandler(orders)
	request := httptest.NewRequest(http.MethodPost, "/api/orders/order-id/approve", nil)
	request.Header.Set("X-Dev-User", "chi")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", response.Code, response.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "SELF_APPROVAL" || !strings.Contains(body["message"], "không thể tự duyệt") {
		t.Fatalf("unexpected error body: %+v", body)
	}
}

func TestBFF_EmptyAllowedActionsSerializesAsArray(t *testing.T) {
	encoded, err := json.Marshal(mapOrder(&orderv1.Order{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"allowedActions":[]`) {
		t.Fatalf("empty allowed actions should be an array: %s", encoded)
	}
}

func TestBFF_CancelOrderForwardsIdentity(t *testing.T) {
	orders := &fakeOrderClient{}
	handler := newTestHandler(orders)
	request := httptest.NewRequest(http.MethodPost, "/api/orders/order-id/cancel", nil)
	request.Header.Set("X-Dev-User", "an")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if orders.metadata.Get("x-user-id")[0] != "an" {
		t.Fatalf("cancellation identity not forwarded: %v", orders.metadata)
	}
}

func TestBFF_AdjustStockForwardsWarehouseCapability(t *testing.T) {
	stock := &fakeStockClient{}
	handler := NewServer(&fakeOrderClient{}, stock, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/stock-items/BAN-01/adjust", strings.NewReader(`{"deltaOnHand":-2}`))
	request.Header.Set("X-Dev-User", "kho")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if stock.adjustment.GetSku() != "BAN-01" || stock.adjustment.GetDeltaOnHand() != -2 || stock.metadata.Get("x-capabilities")[0] != "stock.adjust" {
		t.Fatalf("adjustment or capability not forwarded: request=%+v metadata=%v", stock.adjustment, stock.metadata)
	}
}

func newTestHandler(orders *fakeOrderClient) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(orders, &fakeStockClient{}, logger)
}

func selfApprovalStatus() error {
	base := status.New(codes.PermissionDenied, "permission denied")
	withDetails, err := base.WithDetails(&errdetails.ErrorInfo{Reason: "SELF_APPROVAL", Domain: "neo-mini"})
	if err != nil {
		return err
	}
	return withDetails.Err()
}

type fakeOrderClient struct {
	metadata     metadata.MD
	created      *orderv1.CreateOrderRequest
	approveError error
}

func (f *fakeOrderClient) CreateOrder(ctx context.Context, request *orderv1.CreateOrderRequest, _ ...grpc.CallOption) (*orderv1.CreateOrderResponse, error) {
	f.metadata, _ = metadata.FromOutgoingContext(ctx)
	f.created = request
	return &orderv1.CreateOrderResponse{Order: sampleOrder()}, nil
}

func (f *fakeOrderClient) GetOrder(ctx context.Context, _ *orderv1.GetOrderRequest, _ ...grpc.CallOption) (*orderv1.GetOrderResponse, error) {
	f.metadata, _ = metadata.FromOutgoingContext(ctx)
	return &orderv1.GetOrderResponse{Order: sampleOrder()}, nil
}

func (f *fakeOrderClient) ListOrders(ctx context.Context, _ *orderv1.ListOrdersRequest, _ ...grpc.CallOption) (*orderv1.ListOrdersResponse, error) {
	f.metadata, _ = metadata.FromOutgoingContext(ctx)
	return &orderv1.ListOrdersResponse{Orders: []*orderv1.Order{sampleOrder()}}, nil
}

func (f *fakeOrderClient) ApproveOrder(ctx context.Context, _ *orderv1.ApproveOrderRequest, _ ...grpc.CallOption) (*orderv1.ApproveOrderResponse, error) {
	f.metadata, _ = metadata.FromOutgoingContext(ctx)
	if f.approveError != nil {
		return nil, f.approveError
	}
	return &orderv1.ApproveOrderResponse{Order: sampleOrder()}, nil
}

func (f *fakeOrderClient) RejectOrder(ctx context.Context, _ *orderv1.RejectOrderRequest, _ ...grpc.CallOption) (*orderv1.RejectOrderResponse, error) {
	f.metadata, _ = metadata.FromOutgoingContext(ctx)
	return &orderv1.RejectOrderResponse{Order: sampleOrder()}, nil
}

func (f *fakeOrderClient) CancelOrder(ctx context.Context, _ *orderv1.CancelOrderRequest, _ ...grpc.CallOption) (*orderv1.CancelOrderResponse, error) {
	f.metadata, _ = metadata.FromOutgoingContext(ctx)
	return &orderv1.CancelOrderResponse{Order: sampleOrder()}, nil
}

type fakeStockClient struct {
	metadata   metadata.MD
	adjustment *stockv1.AdjustStockRequest
}

func (*fakeStockClient) ListItems(context.Context, *stockv1.ListItemsRequest, ...grpc.CallOption) (*stockv1.ListItemsResponse, error) {
	return &stockv1.ListItemsResponse{}, nil
}

func (*fakeStockClient) GetItems(context.Context, *stockv1.GetItemsRequest, ...grpc.CallOption) (*stockv1.GetItemsResponse, error) {
	return &stockv1.GetItemsResponse{}, nil
}

func (f *fakeStockClient) AdjustStock(ctx context.Context, request *stockv1.AdjustStockRequest, _ ...grpc.CallOption) (*stockv1.AdjustStockResponse, error) {
	f.metadata, _ = metadata.FromOutgoingContext(ctx)
	f.adjustment = request
	return &stockv1.AdjustStockResponse{Item: &stockv1.StockItem{Sku: request.GetSku(), OnHand: 3}}, nil
}

func sampleOrder() *orderv1.Order {
	return &orderv1.Order{
		Id: "order-id", Code: "DH-000001", Status: "PENDING_APPROVAL", CreatedBy: "an",
		TotalAmount: &orderv1.Money{Amount: "100.00", CurrencyCode: "VND"},
		Lines: []*orderv1.OrderLine{{Sku: "BAN-01", Quantity: 1,
			UnitPrice:  &orderv1.Money{Amount: "100.00", CurrencyCode: "VND"},
			LineAmount: &orderv1.Money{Amount: "100.00", CurrencyCode: "VND"},
		}},
		AllowedActions: []string{"APPROVE", "REJECT"},
	}
}
