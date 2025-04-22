package executor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
)

type Worker struct {
	tasks          chan *service.Task
	queue          service.Queue
	policy         Policy
	storage        service.Storage
	cache          Cache
	maxTaskRetries int
	httpClient     *http.Client
	logger         *zap.Logger
}

func newWorker(
	tasks chan *service.Task,
	queue service.Queue,
	policy Policy,
	storage service.Storage,
	cache Cache,
	maxTaskRetries int,
	httpClient *http.Client,
	logger *zap.Logger,
) *Worker {
	return &Worker{
		tasks:          tasks,
		queue:          queue,
		policy:         policy,
		storage:        storage,
		cache:          cache,
		maxTaskRetries: maxTaskRetries,
		httpClient:     httpClient,
		logger:         logger,
	}
}

func (w *Worker) Start(ctx context.Context) {
	defer w.logger.Debug("task worker stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-w.tasks:
			logger := w.logger.With(
				zap.String("taskID", t.ID),
				zap.String("taskName", t.Name),
			)

			if t.Retries >= w.maxTaskRetries {
				if err := w.queue.Ack(ctx, t); err != nil {
					logger.Error("failed to ack task in queue", zap.Error(err))
				} else {
					logger.Error("task removed from queue due to too many failed executions")
				}
				continue
			}

			executed, err := w.Execute(ctx, t)
			if err != nil {
				logger.Error("error executing task", zap.Error(err))
				if err := w.queue.Unack(ctx, t); err != nil {
					logger.Error("failed to unack task in queue", zap.Error(err))
				}
				continue
			}
			logger.Debug("task execution completed successfully")

			if err := w.cache.Set(
				ctx,
				executed.ID,
				executed.CacheNamespace,
				executed.CacheScope,
				executed.Response,
			); err != nil {
				logger.Error("error storing task result in cache", zap.Error(err))
				if err := w.queue.Unack(ctx, t); err != nil {
					logger.Error("failed to unack task in queue", zap.Error(err))
				}
				continue
			}
			logger.Debug("task results are stored in cache")

			if err := w.storage.SaveTaskHistory(ctx, executed); err != nil {
				logger.Error("error saving task history", zap.Error(err))
				continue
			}
			logger.Debug("task history is saved")

			// remove task from queue
			if err := w.queue.Ack(ctx, executed); err != nil {
				logger.Error("failed to ack task in queue", zap.Error(err))
			}
		}
	}
}

func (w *Worker) Execute(ctx context.Context, task *service.Task) (*service.Task, error) {
	task.StartedAt = time.Now()

	var response []byte
	var err error

	// task is executing a request policy OR an HTTP call to predefined URL
	if task.RequestPolicy != "" {
		response, err = w.policy.Evaluate(ctx, task.RequestPolicy, task.Request)
		if err != nil {
			return nil, errors.New("error evaluating request policy", err)
		}
		task.ResponseCode = http.StatusOK
	} else if task.URL != "" && task.Method != "" {
		var status int
		status, response, err = w.doHTTPTask(ctx, task)
		if err != nil {
			return nil, err
		}
		task.ResponseCode = status
	} else {
		return nil, errors.New(errors.Internal, "invalid task: must define either request policy or url")
	}

	task.Response = response

	// evaluate response policy
	if task.ResponsePolicy != "" {
		resp, err := w.policy.Evaluate(ctx, task.ResponsePolicy, task.Response)
		if err != nil {
			return nil, errors.New("error evaluating response policy", err)
		}
		// overwrite task response with the one returned from the policy
		task.Response = resp
	}

	// evaluate finalizer policy
	if task.FinalPolicy != "" {
		resp, err := w.policy.Evaluate(ctx, task.FinalPolicy, task.Response)
		if err != nil {
			return nil, errors.New("error evaluating final policy", err)
		}
		// overwrite task response with the one returned from the policy
		task.Response = resp
	}

	task.State = service.Done
	task.FinishedAt = time.Now()
	return task, nil
}

func (w *Worker) doHTTPTask(ctx context.Context, task *service.Task) (status int, response []byte, err error) {
	req, err := http.NewRequest(task.Method, task.URL, bytes.NewReader(task.Request))
	if err != nil {
		return 0, nil, errors.New("error creating http request", err)
	}

	resp, err := w.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return 0, nil, errors.New("error executing http request", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	response, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, errors.New("error reading response body", err)
	}

	return resp.StatusCode, response, nil
}
