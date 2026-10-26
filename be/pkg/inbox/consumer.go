package inbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type Handler func(context.Context, []byte) error

type Runner struct {
	consume jetstream.ConsumeContext
}

func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: "MINI", Subjects: []string{"mini.>"},
		Retention: jetstream.LimitsPolicy, Storage: jetstream.FileStorage,
	})
	return err
}

func Start(ctx context.Context, js jetstream.JetStream, durable, subject string, handler Handler, logger *slog.Logger) (*Runner, error) {
	setupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	consumer, err := js.CreateOrUpdateConsumer(setupCtx, "MINI", jetstream.ConsumerConfig{
		Durable: durable, FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverAllPolicy, AckPolicy: jetstream.AckExplicitPolicy,
		AckWait: 15 * time.Second, MaxDeliver: 5,
		BackOff: []time.Duration{time.Second, 5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute},
	})
	if err != nil {
		return nil, err
	}
	consumeCtx, err := consumer.Consume(func(message jetstream.Msg) {
		deliveryCtx, deliveryCancel := context.WithTimeout(ctx, 10*time.Second)
		defer deliveryCancel()
		if err := handler(deliveryCtx, message.Data()); err != nil {
			metadata, metaErr := message.Metadata()
			if metaErr == nil && metadata.NumDelivered >= 5 {
				logger.Error("event permanently failed", "consumer", durable, "deliveries", metadata.NumDelivered, "error", err)
				_ = message.Term()
				return
			}
			logger.Error("event processing failed", "consumer", durable, "error", err)
			_ = message.NakWithDelay(time.Second)
			return
		}
		if err := message.Ack(); err != nil {
			logger.Error("ack event", "consumer", durable, "error", err)
		}
	}, jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
		logger.Error("JetStream consumer error", "consumer", durable, "error", err)
	}))
	if err != nil {
		return nil, err
	}
	return &Runner{consume: consumeCtx}, nil
}

func (r *Runner) Stop() {
	if r == nil || r.consume == nil {
		return
	}
	r.consume.Stop()
	select {
	case <-r.consume.Closed():
	case <-time.After(5 * time.Second):
	}
}
