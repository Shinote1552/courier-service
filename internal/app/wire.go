//go:build wireinject
// +build wireinject

package app

import (
	"context"
	"time"

	orderGateway "service/internal/gateway/grpc/order"
	proto "service/internal/generated/proto/clients"
	courier_get "service/internal/handlers/rest/courier_get"
	courier_post "service/internal/handlers/rest/courier_post"
	courier_put "service/internal/handlers/rest/courier_put"
	couriers_get "service/internal/handlers/rest/couriers_get"
	delivery_assign_post "service/internal/handlers/rest/delivery_assign_post"
	delivery_unassign_post "service/internal/handlers/rest/delivery_unassign_post"
	"service/internal/handlers/tasks/delivery_cleanup"
	"service/internal/pkg/config"
	"service/internal/pkg/factory/delivery_deadline"
	"service/internal/pkg/factory/order_handle"

	courierRepo "service/internal/repository/courier"
	deliveryRepo "service/internal/repository/delivery"
	courierService "service/internal/service/courier"
	deliveryService "service/internal/service/delivery"
	orderService "service/internal/service/order"

	"service/pkg/background"
	"service/pkg/logger"
	"service/pkg/querier"
	"service/pkg/tx"

	"github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type (
	CleanupInterval time.Duration
)

type Application struct {
	ServiceCourier    ServiceCourier
	ServiceDelivery   ServiceDelivery
	BackgroundWorkers *background.Worker
}

type ServiceCourier interface {
	courier_get.Service
	courier_post.Service
	courier_put.Service
	couriers_get.Service
}

type ServiceDelivery interface {
	delivery_assign_post.Service
	delivery_unassign_post.Service
}

// InitializeApplication для HTTP сервиса (cmd/service)
func InitializeApplication(
	ctx context.Context,
	log logger.Logger,
	pool *pgxpool.Pool,
	getter *pgxv5.CtxGetter,
	conn *grpc.ClientConn,
	cfg *config.Config,
) (*Application, error) {
	wire.Build(
		provideTxManager,
		provideQuerier,
		provideCleanupInterval,

		provideCourierRepository,
		provideDeliveryRepository,

		provideServiceCourier,
		provideServiceDelivery,
		delivery_deadline.New,

		provideDeliveryCleanupTask,
		provideTaskList,
		provideBackgroundWorkers,

		wire.Struct(new(Application), "*"),

		wire.Bind(new(ServiceCourier), new(*courierService.Courier)),
		wire.Bind(new(ServiceDelivery), new(*deliveryService.Delivery)),

		wire.Bind(new(courierService.Repository), new(*courierRepo.Repository)),
		wire.Bind(new(deliveryService.Repository), new(*deliveryRepo.Repository)),
		wire.Bind(new(deliveryService.CourierService), new(*courierService.Courier)),
		wire.Bind(new(deliveryService.DeliveryTimeFactory), new(*delivery_deadline.DeliveryTimeFactory)),

		wire.Bind(new(courierService.TxManager), new(*tx.Manager)),
		wire.Bind(new(deliveryService.TxManager), new(*tx.Manager)),

		wire.Bind(new(delivery_cleanup.Service), new(*deliveryService.Delivery)),
	)
	return &Application{}, nil
}

type KafkaWorkerApp struct {
	OrderService *orderService.Service
}

// InitializeKafkaWorkerApp для Kafka воркера (cmd/worker-order-status-changed)
func InitializeKafkaWorkerApp(
	ctx context.Context,
	log logger.Logger,
	pool *pgxpool.Pool,
	getter *pgxv5.CtxGetter,
	conn *grpc.ClientConn,
	cfg *config.Config,
) (*KafkaWorkerApp, error) {
	wire.Build(
		provideTxManager,
		provideQuerier,

		provideCourierRepository,
		provideDeliveryRepository,

		provideServiceCourier,
		provideServiceDelivery,
		delivery_deadline.New,

		provideOrderServiceClient,
		provideOrderGateway,
		provideStatusHandlerFabric,
		provideOrderService,

		wire.Bind(new(courierService.Repository), new(*courierRepo.Repository)),
		wire.Bind(new(deliveryService.Repository), new(*deliveryRepo.Repository)),
		wire.Bind(new(deliveryService.CourierService), new(*courierService.Courier)),
		wire.Bind(new(deliveryService.DeliveryTimeFactory), new(*delivery_deadline.DeliveryTimeFactory)),
		wire.Bind(new(orderService.HandlerFactory), new(*order_handle.StatusHandlerFactory)),

		wire.Bind(new(courierService.TxManager), new(*tx.Manager)),
		wire.Bind(new(deliveryService.TxManager), new(*tx.Manager)),

		wire.Struct(new(KafkaWorkerApp), "*"),
	)
	return nil, nil
}

func provideTxManager(pool *pgxpool.Pool) *tx.Manager {
	return tx.New(pool)
}

func provideQuerier(pool *pgxpool.Pool, getter *pgxv5.CtxGetter) *querier.Querier {
	return querier.New(pool, getter)
}

func provideCourierRepository(querier *querier.Querier) *courierRepo.Repository {
	return courierRepo.New(querier)
}

func provideDeliveryRepository(querier *querier.Querier) *deliveryRepo.Repository {
	return deliveryRepo.New(querier)
}

func provideServiceCourier(
	repository courierService.Repository,
	txManager courierService.TxManager,
) *courierService.Courier {
	return courierService.New(repository, txManager)
}

func provideServiceDelivery(
	repository deliveryService.Repository,
	courierService deliveryService.CourierService,
	timeFactory deliveryService.DeliveryTimeFactory,
	txManager deliveryService.TxManager,
) *deliveryService.Delivery {
	return deliveryService.New(
		repository,
		courierService,
		timeFactory,
		txManager,
	)
}

func provideCleanupInterval(cfg *config.Config) CleanupInterval {
	return CleanupInterval(cfg.Tasks.CouriersStatusUpdateInterval)
}

func provideOrderServiceClient(conn *grpc.ClientConn) proto.OrdersServiceClient {
	return proto.NewOrdersServiceClient(conn)
}

func provideOrderGateway(client proto.OrdersServiceClient) *orderGateway.OrderGateway {
	return orderGateway.New(client)
}

// provideOrderService создает orderService для обработки событий Kafka
func provideOrderService(
	orderGateway *orderGateway.OrderGateway,
	deliveryService *deliveryService.Delivery,
	handlerFactory orderService.HandlerFactory,
) *orderService.Service {
	return orderService.New(orderGateway, deliveryService, handlerFactory)
}

func provideStatusHandlerFabric(deliveryService *deliveryService.Delivery) *order_handle.StatusHandlerFactory {
	return order_handle.NewStatusHandlerFactory(deliveryService)
}

func provideDeliveryCleanupTask(
	log logger.Logger,
	deliveryService delivery_cleanup.Service,
	interval CleanupInterval,
) *delivery_cleanup.DeliveryCleanup {
	return delivery_cleanup.NewDeliveryCleanup(log, deliveryService, time.Duration(interval))
}

func provideTaskList(
	deliveryCleanupTask *delivery_cleanup.DeliveryCleanup,
) []background.Task {
	return []background.Task{
		deliveryCleanupTask,
	}
}

func provideBackgroundWorkers(ctx context.Context, log logger.Logger, tasks []background.Task) (*background.Worker, error) {
	return background.New(ctx, log, tasks)
}
