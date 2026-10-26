package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	stockv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/stock/v1"
	httpadapter "github.com/trieu22676191/MiniProject/be/services/bff/internal/adapter/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	orderConn, err := grpc.NewClient(address("ORDER_GRPC_ADDR", "127.0.0.1:50051"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("connect order service", "error", err)
		os.Exit(1)
	}
	defer orderConn.Close()
	stockConn, err := grpc.NewClient(address("STOCK_GRPC_ADDR", "127.0.0.1:50052"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("connect stock service", "error", err)
		os.Exit(1)
	}
	defer stockConn.Close()
	server := &http.Server{
		Addr:              address("BFF_ADDR", "127.0.0.1:8080"),
		Handler:           httpadapter.NewServer(orderv1.NewOrderServiceClient(orderConn), stockv1.NewStockServiceClient(stockConn), logger),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown HTTP server", "error", err)
		}
	}()
	logger.Info("BFF listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("serve BFF", "error", err)
		os.Exit(1)
	}
}

func address(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
