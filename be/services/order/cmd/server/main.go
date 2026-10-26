package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	grpcadapter "github.com/trieu22676191/MiniProject/be/services/order/internal/adapter/grpc"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/adapter/repo"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	databaseURL := os.Getenv("ORDER_DATABASE_URL")
	if databaseURL == "" {
		logger.Error("ORDER_DATABASE_URL is required")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Error("ping database", "error", err)
		os.Exit(1)
	}
	stockAddress := os.Getenv("STOCK_GRPC_ADDR")
	if stockAddress == "" {
		stockAddress = "127.0.0.1:50052"
	}
	stockConn, err := grpc.NewClient(stockAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("create stock client", "error", err)
		os.Exit(1)
	}
	defer stockConn.Close()
	address := os.Getenv("ORDER_GRPC_ADDR")
	if address == "" {
		address = "127.0.0.1:50051"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("listen", "error", err)
		os.Exit(1)
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(tenant.UnaryServerInterceptor))
	orders := usecase.NewOrders(repo.NewOrders(db.New(pool)), grpcadapter.NewStockClient(stockConn))
	orderv1.RegisterOrderServiceServer(server, grpcadapter.NewServer(orders))
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()
	logger.Info("order service listening", "address", address)
	if err := server.Serve(listener); err != nil && ctx.Err() == nil {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}
