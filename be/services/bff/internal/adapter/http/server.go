package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	orders orderv1.OrderServiceClient
	stock  stockv1.StockServiceClient
	logger *slog.Logger
}

func NewServer(orders orderv1.OrderServiceClient, stock stockv1.StockServiceClient, logger *slog.Logger) http.Handler {
	server := &Server{orders: orders, stock: stock, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/me", server.me)
	mux.HandleFunc("GET /api/orders", server.listOrders)
	mux.HandleFunc("POST /api/orders", server.createOrder)
	mux.HandleFunc("GET /api/orders/{id}", server.getOrder)
	mux.HandleFunc("POST /api/orders/{id}/approve", server.approveOrder)
	mux.HandleFunc("POST /api/orders/{id}/reject", server.rejectOrder)
	mux.HandleFunc("GET /api/stock-items", server.listStockItems)
	return server.requireDevUser(mux)
}

type identityKey struct{}

func (s *Server) requireDevUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := lookupDevUser(r.Header.Get("X-Dev-User"))
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Hãy chọn người dùng để tiếp tục.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, user)))
	})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(identityKey{}).(devUser)
	capabilities := make([]string, 0, len(user.identity.Capabilities))
	for capability := range user.identity.Capabilities {
		capabilities = append(capabilities, capability)
	}
	sort.Strings(capabilities)
	writeJSON(w, http.StatusOK, map[string]any{
		"id": user.identity.UserID, "name": user.name,
		"tenantId": user.identity.TenantID.String(), "capabilities": capabilities,
	})
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.rpcContext(r)
	defer cancel()
	response, err := s.orders.ListOrders(ctx, &orderv1.ListOrdersRequest{})
	if err != nil {
		s.writeRPCError(w, err)
		return
	}
	orders := make([]orderDTO, 0, len(response.GetOrders()))
	for _, order := range response.GetOrders() {
		orders = append(orders, mapOrder(order))
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Lines []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		} `json:"lines"`
	}
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Dữ liệu đơn hàng không hợp lệ.")
		return
	}
	lines := make([]*orderv1.OrderLineInput, 0, len(request.Lines))
	for _, line := range request.Lines {
		lines = append(lines, &orderv1.OrderLineInput{Sku: line.SKU, Quantity: line.Quantity})
	}
	ctx, cancel := s.rpcContext(r)
	defer cancel()
	response, err := s.orders.CreateOrder(ctx, &orderv1.CreateOrderRequest{Lines: lines})
	if err != nil {
		s.writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"order": mapOrder(response.GetOrder())})
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.rpcContext(r)
	defer cancel()
	response, err := s.orders.GetOrder(ctx, &orderv1.GetOrderRequest{Id: r.PathValue("id")})
	if err != nil {
		s.writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": mapOrder(response.GetOrder())})
}

func (s *Server) approveOrder(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.rpcContext(r)
	defer cancel()
	response, err := s.orders.ApproveOrder(ctx, &orderv1.ApproveOrderRequest{Id: r.PathValue("id")})
	if err != nil {
		s.writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": mapOrder(response.GetOrder())})
}

func (s *Server) rejectOrder(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Dữ liệu từ chối đơn không hợp lệ.")
		return
	}
	ctx, cancel := s.rpcContext(r)
	defer cancel()
	response, err := s.orders.RejectOrder(ctx, &orderv1.RejectOrderRequest{Id: r.PathValue("id"), Reason: request.Reason})
	if err != nil {
		s.writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": mapOrder(response.GetOrder())})
}

func (s *Server) listStockItems(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.rpcContext(r)
	defer cancel()
	response, err := s.stock.ListItems(ctx, &stockv1.ListItemsRequest{})
	if err != nil {
		s.writeRPCError(w, err)
		return
	}
	items := make([]stockItemDTO, 0, len(response.GetItems()))
	for _, item := range response.GetItems() {
		items = append(items, stockItemDTO{
			SKU: item.GetSku(), Name: item.GetName(), UnitPrice: mapMoney(item.GetUnitPrice()),
			OnHand: item.GetOnHand(), Reserved: item.GetReserved(), Available: item.GetAvailable(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) rpcContext(r *http.Request) (context.Context, context.CancelFunc) {
	user := r.Context().Value(identityKey{}).(devUser)
	capabilities := make([]string, 0, len(user.identity.Capabilities))
	for capability := range user.identity.Capabilities {
		capabilities = append(capabilities, capability)
	}
	sort.Strings(capabilities)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	ctx = metadata.AppendToOutgoingContext(ctx,
		"x-tenant-id", user.identity.TenantID.String(),
		"x-user-id", user.identity.UserID,
		"x-capabilities", strings.Join(capabilities, ","),
	)
	return ctx, cancel
}

type moneyDTO struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}

type orderDTO struct {
	ID             string         `json:"id"`
	Code           string         `json:"code"`
	Status         string         `json:"status"`
	CreatedBy      string         `json:"createdBy"`
	DecidedBy      string         `json:"decidedBy"`
	RejectReason   string         `json:"rejectReason"`
	TotalAmount    moneyDTO       `json:"totalAmount"`
	Lines          []orderLineDTO `json:"lines"`
	AllowedActions []string       `json:"allowedActions"`
}

type orderLineDTO struct {
	SKU        string   `json:"sku"`
	Quantity   int64    `json:"quantity"`
	UnitPrice  moneyDTO `json:"unitPrice"`
	LineAmount moneyDTO `json:"lineAmount"`
}

type stockItemDTO struct {
	SKU       string   `json:"sku"`
	Name      string   `json:"name"`
	UnitPrice moneyDTO `json:"unitPrice"`
	OnHand    int64    `json:"onHand"`
	Reserved  int64    `json:"reserved"`
	Available int64    `json:"available"`
}

func mapOrder(order *orderv1.Order) orderDTO {
	if order == nil {
		return orderDTO{Lines: []orderLineDTO{}, AllowedActions: []string{}}
	}
	lines := make([]orderLineDTO, 0, len(order.GetLines()))
	for _, line := range order.GetLines() {
		lines = append(lines, orderLineDTO{
			SKU: line.GetSku(), Quantity: line.GetQuantity(),
			UnitPrice: mapMoney(line.GetUnitPrice()), LineAmount: mapMoney(line.GetLineAmount()),
		})
	}
	return orderDTO{
		ID: order.GetId(), Code: order.GetCode(), Status: order.GetStatus(),
		CreatedBy: order.GetCreatedBy(), DecidedBy: order.GetDecidedBy(), RejectReason: order.GetRejectReason(),
		TotalAmount: mapMoney(order.GetTotalAmount()), Lines: lines, AllowedActions: order.GetAllowedActions(),
	}
}

func mapMoney(money interface {
	GetAmount() string
	GetCurrencyCode() string
}) moneyDTO {
	if money == nil {
		return moneyDTO{}
	}
	return moneyDTO{Amount: money.GetAmount(), CurrencyCode: money.GetCurrencyCode()}
}

func decodeBody(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request must contain one JSON value")
	}
	return nil
}

func (s *Server) writeRPCError(w http.ResponseWriter, err error) {
	s.logger.Error("gRPC request failed", "error", err)
	grpcStatus, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Đã xảy ra lỗi máy chủ.")
		return
	}
	httpStatus, code, message := http.StatusInternalServerError, "INTERNAL", "Đã xảy ra lỗi máy chủ."
	switch grpcStatus.Code() {
	case codes.InvalidArgument:
		httpStatus, code, message = http.StatusBadRequest, "INVALID_ARGUMENT", "Thông tin gửi lên không hợp lệ."
	case codes.Unauthenticated:
		httpStatus, code, message = http.StatusUnauthorized, "UNAUTHENTICATED", "Phiên người dùng không hợp lệ."
	case codes.PermissionDenied:
		httpStatus, code, message = http.StatusForbidden, "PERMISSION_DENIED", "Bạn không có quyền thực hiện thao tác này."
	case codes.NotFound:
		httpStatus, code, message = http.StatusNotFound, "NOT_FOUND", "Không tìm thấy dữ liệu."
	case codes.FailedPrecondition, codes.AlreadyExists, codes.Aborted:
		httpStatus, code, message = http.StatusConflict, "FAILED_PRECONDITION", "Trạng thái hiện tại không cho phép thao tác này."
	case codes.DeadlineExceeded:
		httpStatus, code, message = http.StatusGatewayTimeout, "DEADLINE_EXCEEDED", "Dịch vụ xử lý quá thời gian chờ."
	case codes.Unavailable:
		httpStatus, code, message = http.StatusServiceUnavailable, "UNAVAILABLE", "Dịch vụ tạm thời chưa sẵn sàng."
	}
	var details interface{ GetReason() string }
	for _, detail := range grpcStatus.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			details = info
			break
		}
	}
	if details != nil {
		code = details.GetReason()
		if code == "SELF_APPROVAL" {
			message = "Bạn không thể tự duyệt đơn hàng của mình."
		}
	}
	writeError(w, httpStatus, code, message)
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, map[string]string{"code": code, "message": message})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		return
	}
}
