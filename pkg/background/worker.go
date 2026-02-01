package background

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"service/pkg/logger"

	"golang.org/x/sync/errgroup"
)

// Task определяет интерфейс для фоновых задач, которые могут выполняться периодически.
type Task interface {
	// TTL возвращает интервал между выполнениями задачи.
	TTL() time.Duration

	// Do выполняет логику задачи.
	Do(context.Context) error

	// Info возвращает читаемое описание задачи для логгирования и отладки.
	Info() string
}

type handlerLogger interface {
	Info(msg string, fields ...logger.Field)
	Warn(msg string, fields ...logger.Field)
	Error(msg string, fields ...logger.Field)
	With(fields ...logger.Field) logger.Logger
}

// Worker управляет выполнением набора фоновых задач.
type Worker struct {
	log   handlerLogger
	tasks []Task
}

// New создает и запускает Worker для выполнения фоновых задач.
//
// Поведение функции:
//  1. Все задачи сначала выполняются синхронно для инициализации (так называемый "прогрев").
//     Это гарантирует, что при старте приложения все задачи будут выполнены хотя бы один раз,
//     и любые ошибки инициализации будут возвращены немедленно.
//  2. Если любая задача завершается с ошибкой или паникой на этапе инициализации,
//     New возвращает ошибку и Worker не создается.
//  3. Задачи выполняются в фоне до тех пор, пока не будет отменен переданный контекст.
func New(ctx context.Context, log handlerLogger, tasks []Task) (*Worker, error) {
	if len(tasks) == 0 {
		return &Worker{
			log:   log,
			tasks: tasks,
		}, nil
	}

	initGroup, initCtx := errgroup.WithContext(ctx)
	for i := 0; i < len(tasks); i++ {
		task := tasks[i]
		initGroup.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					err = fmt.Errorf("init panic: %v\n%s", r, stack)
					log.Error("Task panic during init",
						logger.NewField("task", task.Info()),
						logger.NewField("recover", r),
						logger.NewField("stack", stack),
					)
				}
			}()
			log.Info("Initializing",
				logger.NewField("task", task.Info()),
			)
			return task.Do(initCtx)
		})
	}

	if err := initGroup.Wait(); err != nil {
		return nil, fmt.Errorf("failed to initialize tasks: %w", err)
	}

	worker := &Worker{
		log:   log,
		tasks: tasks,
	}

	for i := 0; i < len(tasks); i++ {
		task := tasks[i]
		go worker.runBackgroundTask(ctx, task)
	}

	return worker, nil
}

func (w *Worker) runBackgroundTask(ctx context.Context, task Task) {
	ttl := task.TTL()
	if ttl <= 0 {
		w.log.Warn("invalid TTL, skipping periodic execution",
			logger.NewField("task", task.Info()),
			logger.NewField("TTL", ttl),
		)
		return
	}
	w.log.Warn("Starting periodic execution",
		logger.NewField("task", task.Info()),
		logger.NewField("TTL", ttl),
	)

	ticker := time.NewTicker(ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Warn("Stopping task (context cancelled)",
				logger.NewField("task", task.Info()),
			)
			return
		case <-ticker.C:
			w.executeTaskSafely(ctx, task)
		}
	}
}

func (w *Worker) executeTaskSafely(ctx context.Context, task Task) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()

			w.log.Error("Background task panic",
				logger.NewField("task", task.Info()),
				logger.NewField("recover", r),
				logger.NewField("stack", stack),
			)
		}
	}()

	if err := task.Do(ctx); err != nil {
		w.log.Error("Background task failed",
			logger.NewField("task", task.Info()),
			logger.NewField("error", err),
		)
	}
}
