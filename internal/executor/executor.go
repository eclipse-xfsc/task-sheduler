package executor

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
)

// Policy client.
type Policy interface {
	Evaluate(ctx context.Context, policy string, data []byte) ([]byte, error)
}

type Cache interface {
	Set(ctx context.Context, key, namespace, scope string, value []byte) error
	Get(ctx context.Context, key, namespace, scope string) ([]byte, error)
}

type Executor struct {
	queue          service.Queue
	policy         Policy
	storage        service.Storage
	cache          Cache
	workers        int
	pollInterval   time.Duration
	maxTaskRetries int

	httpClient *http.Client
	logger     *zap.Logger
}

func New(
	queue service.Queue,
	policy Policy,
	storage service.Storage,
	cache Cache,
	workers int,
	pollInterval time.Duration,
	maxTaskRetries int,
	httpClient *http.Client,
	logger *zap.Logger,
) *Executor {
	return &Executor{
		queue:          queue,
		policy:         policy,
		storage:        storage,
		cache:          cache,
		workers:        workers,
		pollInterval:   pollInterval,
		maxTaskRetries: maxTaskRetries,
		httpClient:     httpClient,
		logger:         logger,
	}
}

func (e *Executor) Start(ctx context.Context) error {
	defer e.logger.Info("task executor stopped")

	var wg sync.WaitGroup
	tasks := make(chan *service.Task)
	for i := 0; i < e.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker := newWorker(tasks, e.queue, e.policy, e.storage, e.cache, e.maxTaskRetries, e.httpClient, e.logger)
			worker.Start(ctx)
		}()
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(e.pollInterval):
			t, err := e.queue.Poll(ctx)
			if err != nil {
				if !errors.Is(errors.NotFound, err) {
					e.logger.Error("error getting task from queue", zap.Error(err))
				}
				continue
			}
			tasks <- t // send task to the workers for execution
		}
	}

	wg.Wait() // wait all workers to stop

	return ctx.Err()
}
