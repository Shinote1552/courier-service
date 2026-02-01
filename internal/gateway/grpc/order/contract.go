//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=order_test
package order

import (
	"context"

	"google.golang.org/grpc"
	proto "service/internal/generated/proto/clients"
)

type client interface {
	GetOrders(ctx context.Context, in *proto.GetOrdersRequest, opts ...grpc.CallOption) (*proto.GetOrdersResponse, error)
	GetOrderById(ctx context.Context, in *proto.GetOrderByIdRequest, opts ...grpc.CallOption) (*proto.GetOrderByIdResponse, error)
}

type retrier interface {
	ExecuteWithContext(ctx context.Context, fn func(context.Context) error) error
}
