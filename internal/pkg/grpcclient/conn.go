package grpcclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/timestamppb"
	proto "service/internal/generated/proto/clients"
	"service/internal/pkg/config"
	"service/pkg/logger"
	retrierconfig "service/pkg/retrier"
	"service/pkg/retrier/backoff_adapter"
)

const (
	KeepaliveTime                = 5 * time.Minute
	KeepaliveTimeout             = 3 * time.Second
	KeepalivePermitWithoutStream = false

	initialInterval = 1 * time.Second
	maxInterval     = 30 * time.Second
	maxElapsedTime  = 2 * time.Minute
	randomization   = 0.5
	multiplier      = 2
)

func NewConnClient(ctx context.Context, log logger.Logger, cfg *config.OrderService) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		cfg.GRPCHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                KeepaliveTime,
			Timeout:             KeepaliveTimeout,
			PermitWithoutStream: KeepalivePermitWithoutStream,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	grpcLog := log.With(
		logger.NewField("component", "grpc-client"),
		logger.NewField("host", cfg.GRPCHost),
	)

	err = pingGRPC(ctx, grpcLog, conn)
	if err != nil {
		connCloseErr := conn.Close()
		if connCloseErr != nil {
			return nil, fmt.Errorf("gRPC connection: %w (failed to close: %v)", err, connCloseErr)
		}
		return nil, fmt.Errorf("gRPC connection: %w", err)
	}

	return conn, nil
}

func pingGRPC(ctx context.Context, log logger.Logger, conn *grpc.ClientConn) error {
	client := proto.NewOrdersServiceClient(conn)

	retryConfig := retrierconfig.Config{
		InitialInterval: initialInterval,
		MaxInterval:     maxInterval,
		MaxElapsedTime:  maxElapsedTime,
		Randomization:   randomization,
		Multiplier:      multiplier,
		ShouldRetry:     nil, // все ошибки ретраим
	}

	retrier := backoff_adapter.New(retryConfig)

	var attempt uint64
	err := retrier.ExecuteWithContext(ctx, func(ctx context.Context) error {
		attempt++
		log.With(
			logger.NewField("attempt", attempt),
		).Info("attempting gRPC connection")

		_, err := client.GetOrders(ctx, &proto.GetOrdersRequest{
			From: timestamppb.New(time.Now().Add(-1 * time.Second)),
		})
		return err
	})
	if err != nil {
		log.With(
			logger.NewField("error", err),
			logger.NewField("attempts", attempt),
		).Error("gRPC connection failed after retries")
		return fmt.Errorf("failed to establish gRPC connection: %w", err)
	}

	log.With(logger.NewField(
		"attempts", attempt),
	).Info("gRPC connection established")
	return nil
}
