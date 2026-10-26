package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	grpcadapter "github.com/trieu22676191/MiniProject/be/services/stock/internal/adapter/grpc"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/adapter/repo"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/usecase"
	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	databaseURL := os.Getenv("STOCK_DATABASE_URL")
	if databaseURL == "" {
		logger.Error("STOCK_DATABASE_URL is required")
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
	address := os.Getenv("STOCK_GRPC_ADDR")
	if address == "" {
		address = "127.0.0.1:50052"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("listen", "error", err)
		os.Exit(1)
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(tenant.UnaryServerInterceptor))
	stockv1.RegisterStockServiceServer(server, grpcadapter.NewServer(usecase.NewItems(repo.NewItems(db.New(pool)))))
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()
	logger.Info("stock service listening", "address", address)
	if err := server.Serve(listener); err != nil && ctx.Err() == nil {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}
