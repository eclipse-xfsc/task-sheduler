package task

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	goatask "github.com/eclipse-xfsc/task-sheduler/gen/task"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
)

//go:generate counterfeiter . Cache

type Cache interface {
	Get(ctx context.Context, key, namespace, scope string) ([]byte, error)
}

type Service struct {
	storage service.Storage
	queue   service.Queue
	cache   Cache
	logger  *zap.Logger
}

// New creates the task service.
func New(template service.Storage, queue service.Queue, cache Cache, logger *zap.Logger) *Service {
	return &Service{
		storage: template,
		queue:   queue,
		cache:   cache,
		logger:  logger,
	}
}

// Create a new task and put it in a Queue for later execution.
func (s *Service) Create(ctx context.Context, req *goatask.CreateTaskRequest) (res *goatask.CreateTaskResult, err error) {
	if req.TaskName == "" {
		return nil, errors.New(errors.BadRequest, "missing taskName")
	}

	logger := s.logger.With(zap.String("taskName", req.TaskName))

	// get predefined task definition from storage
	task, err := s.storage.TaskTemplate(ctx, req.TaskName)
	if err != nil {
		logger.Error("error getting task template from storage", zap.Error(err))
		return nil, err
	}

	taskRequest, err := json.Marshal(req.Data)
	if err != nil {
		logger.Error("error marshaling request data to JSON", zap.Error(err))
		return nil, errors.New(errors.BadRequest, "error marshaling request data to JSON", err)
	}

	task.ID = uuid.NewString()
	task.State = service.Created
	task.CreatedAt = time.Now()
	task.Request = taskRequest

	// if cache key namespace and scope are given, use them instead of the defaults
	if req.CacheNamespace != nil && *req.CacheNamespace != "" {
		task.CacheNamespace = *req.CacheNamespace
	}
	if req.CacheScope != nil && *req.CacheScope != "" {
		task.CacheScope = *req.CacheScope
	}

	if err := s.queue.Add(ctx, task); err != nil {
		logger.Error("error adding task to queue", zap.Error(err))
		return nil, errors.New("failed to create task", err)
	}

	return &goatask.CreateTaskResult{TaskID: task.ID}, nil
}

// TaskResult retrieves task result from the Cache service.
func (s *Service) TaskResult(ctx context.Context, req *goatask.TaskResultRequest) (res interface{}, err error) {
	if req.TaskID == "" {
		return nil, errors.New(errors.BadRequest, "missing taskID")
	}

	logger := s.logger.With(zap.String("taskID", req.TaskID))

	var task *service.Task
	task, err = s.storage.TaskHistory(ctx, req.TaskID)
	if err != nil && !errors.Is(errors.NotFound, err) {
		logger.Error("error getting task from history collection", zap.Error(err))
		return nil, err
	}

	if task == nil {
		task, err = s.storage.Task(ctx, req.TaskID)
		if err != nil {
			if errors.Is(errors.NotFound, err) {
				return nil, errors.New("task is not found", err)
			}
			logger.Error("error getting task from history collection", zap.Error(err))
			return nil, err
		}
	}

	if task.State != service.Done && task.State != service.Failed {
		return nil, errors.New(errors.NotFound, "no result, task is not completed")
	}

	value, err := s.cache.Get(ctx, task.ID, task.CacheNamespace, task.CacheScope)
	if err != nil {
		logger.Error("error getting task result from cache", zap.Error(err))
		return nil, err
	}

	var result interface{}
	if err := json.NewDecoder(bytes.NewReader(value)).Decode(&result); err != nil {
		logger.Error("error decoding result from cache", zap.Error(err))
		return nil, errors.New("error decoding result from cache", err)
	}

	return result, nil
}
