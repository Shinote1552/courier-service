package main

import (
	"context"
	"fmt"
	pb "github.com/nikolaev/service-order/internal/proto"
	fetchOrders "github.com/nikolaev/service-order/internal/rpc/fetch_orders"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/dig"

	"github.com/nikolaev/service-order/internal/gateway/kafka"
	"github.com/nikolaev/service-order/internal/handlers"
	repo "github.com/nikolaev/service-order/internal/repository/order"
	seed "github.com/nikolaev/service-order/internal/usecase/debug/seed"
	ucase "github.com/nikolaev/service-order/internal/usecase/order"
)

type sysClock struct{}

func (sysClock) Now() time.Time { return time.Now().UTC() }

func main() {
	c := dig.New()

	_ = c.Provide(provideInMemory)
	_ = c.Provide(provideRepo)
	_ = c.Provide(provideProducer)
	_ = c.Provide(provideService)
	_ = c.Provide(provideSeeder)
	_ = c.Provide(provideOrderHandler)
	_ = c.Provide(provideRouter)
	_ = c.Provide(provideGRPCHandler)

	err := c.Invoke(func(r *chi.Mux, h *handlers.OrderHandler, grpch *fetchOrders.Handler, mem *repo.InMemory, prod ucase.Producer) error {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			runStatusWorker(ctx, mem, prod)
		}()

		r.Mount("/public/api/v1", h.Routes())

		eg, _ := errgroup.WithContext(ctx)
		eg.Go(func() error {
			log.Println("service started on :8080")
			if err := http.ListenAndServe(":8080", r); err != nil {
				return fmt.Errorf("failed to listen 8080: %v", err)
			}

			return nil
		})

		// -----------------
		lis, err := net.Listen("tcp", ":50051") // Порты TCP для прослушивания
		if err != nil {
			log.Fatalf("failed to listen 50051: %v", err)
		}

		s := grpc.NewServer()
		pb.RegisterOrdersServiceServer(s, grpch)

		eg.Go(func() error {
			log.Println("service started on :50051")
			if err := s.Serve(lis); err != nil {
				return fmt.Errorf("failed to listen 50051: %v", err)
			}

			return nil
		})

		err = eg.Wait()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func provideSeeder(mem *repo.InMemory) seed.Service {
	return seed.New(mem, sysClock{})
}

func provideOrderHandler(svc ucase.Service, dbg seed.Service) *handlers.OrderHandler {
	return handlers.NewOrderHandler(svc, dbg)
}

func provideRouter() *chi.Mux {
	r := chi.NewMux()
	r.Use(middleware.Logger)
	return r
}

func runStatusWorker(ctx context.Context, mem *repo.InMemory, prod ucase.Producer) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			changed := mem.AdvanceStatuses(now)
			for _, o := range changed {
				_ = prod.OrderUpdated(ctx, o)
			}
		}
	}
}

func provideInMemory() *repo.InMemory { return repo.NewInMemory() }

func provideRepo(mem *repo.InMemory) ucase.Repository { return mem }

func provideProducer() ucase.Producer {
	if os.Getenv("KAFKA_BROKERS") != "" {
		p, err := kafka.NewSaramaProducer()
		if err == nil {
			return p
		}
		log.Printf("failed to init sarama producer, fallback to noop: %v", err)
	}
	return kafka.NoopProducer{}
}

func provideService(r ucase.Repository, p ucase.Producer) ucase.Service {
	return ucase.New(r, p)
}

func provideGRPCHandler(r ucase.Repository) *fetchOrders.Handler {
	return fetchOrders.New(r)
}
