package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	orderv1 "github.com/trieu22676191/MiniProject/be/gen/go/mini/order/v1"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/inbox"
	"github.com/trieu22676191/MiniProject/be/pkg/outbox"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/adapter/consumer"
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
	relayURL := os.Getenv("OUTBOX_DATABASE_URL")
	if relayURL == "" {
		logger.Error("OUTBOX_DATABASE_URL is required")
		os.Exit(1)
	}
	relayPool, err := pgxpool.New(ctx, relayURL)
	if err != nil {
		logger.Error("open outbox database", "error", err)
		os.Exit(1)
	}
	defer relayPool.Close()
	if err := relayPool.Ping(ctx); err != nil {
		logger.Error("ping outbox database", "error", err)
		os.Exit(1)
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	natsConn, err := nats.Connect(natsURL, nats.Timeout(5*time.Second), nats.Name("mini-order"))
	if err != nil {
		logger.Error("connect NATS", "error", err)
		os.Exit(1)
	}
	defer natsConn.Close()
	js, err := jetstream.New(natsConn)
	if err != nil {
		logger.Error("create JetStream client", "error", err)
		os.Exit(1)
	}
	setupCtx, cancelSetup := context.WithTimeout(ctx, 5*time.Second)
	if err := inbox.EnsureStream(setupCtx, js); err != nil {
		cancelSetup()
		logger.Error("ensure JetStream stream", "error", err)
		os.Exit(1)
	}
	cancelSetup()
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
	reservationHandler := consumer.NewReservationResult(orders)
	consumerRunner, err := inbox.Start(ctx, js, "order-reservation-result", "mini.stk.reservation.*", reservationHandler.Handle, logger)
	if err != nil {
		logger.Error("start reservation result consumer", "error", err)
		os.Exit(1)
	}
	defer consumerRunner.Stop()
	go func() {
		if err := outbox.Run(ctx, relayPool, js, "mini_order", logger); err != nil {
			logger.Error("order outbox relay stopped", "error", err)
		}
	}()
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
