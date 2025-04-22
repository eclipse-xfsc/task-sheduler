package listexecutor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	ptr "github.com/eclipse-xfsc/microservice-core-go/pkg/ptr"
	goatasklist "github.com/eclipse-xfsc/task-sheduler/gen/task_list"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
)

type token struct{}

const (
	sequential = "sequential"
	parallel   = "parallel"
)

// Policy client.
type Policy interface {
	Evaluate(ctx context.Context, policy string, data []byte) ([]byte, error)
}

type Cache interface {
	Set(ctx context.Context, key, namespace, scope string, value []byte) error
	Get(ctx context.Context, key, namespace, scope string) ([]byte, error)
}

type ListExecutor struct {
	queue        service.Queue
	policy       Policy
	storage      service.Storage
	cache        Cache
	workers      int
	pollInterval time.Duration

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
	httpClient *http.Client,
	logger *zap.Logger,
) *ListExecutor {
	return &ListExecutor{
		queue:        queue,
		policy:       policy,
		storage:      storage,
		cache:        cache,
		workers:      workers,
		pollInterval: pollInterval,
		httpClient:   httpClient,
		logger:       logger,
	}
}

func (l *ListExecutor) Start(ctx context.Context) error {
	defer l.logger.Info("taskList executor stopped")

	// buffered channel used as a semaphore to limit concurrent executions
	sem := make(chan token, l.workers)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(l.pollInterval):
			sem <- token{} // acquire a semaphore

			taskList, err := l.queue.PollList(ctx)
			if err != nil {
				if !errors.Is(errors.NotFound, err) {
					l.logger.Error("error getting taskList from queue", zap.Error(err))
				}
				<-sem // release the semaphore
				continue
			}

			go func(list *service.TaskList) {
				l.Execute(ctx, list)
				<-sem // release the semaphore
			}(taskList)
		}
	}

	// wait for completion
	for n := l.workers; n > 0; n-- {
		sem <- token{}
	}

	return ctx.Err()
}

func (l *ListExecutor) Execute(ctx context.Context, list *service.TaskList) {
	logger := l.logger.With(
		zap.String("taskListID", list.ID),
		zap.String("taskListName", list.Name),
	)
	list.State = service.Pending
	list.StartedAt = time.Now()

	var state goatasklist.TaskListStatusResponse

	// execute groups sequentially
	for i := range list.Groups {
		groupState, err := l.executeGroup(ctx, &list.Groups[i])
		if err != nil {
			logger.Error("error executing group", zap.Error(err))
			list.Groups[i].State = service.Failed
			list.State = service.Failed
		}
		state.Groups = append(state.Groups, groupState)

		//mark taskList as `Failed` if the group's state is `Failed`
		if *groupState.Status == service.Failed {
			list.State = service.Failed
		}
	}

	if list.State != service.Failed {
		list.State = service.Done
	}
	list.FinishedAt = time.Now()

	state.ID = list.ID
	state.Status = string(list.State)

	value, err := json.Marshal(state)
	if err != nil {
		logger.Error("error marshaling taskList state", zap.Error(err))
	} else {
		if err := l.cache.Set(ctx, list.ID, list.CacheNamespace, list.CacheScope, value); err != nil {
			logger.Error("error storing taskList state in cache", zap.Error(err))
		} else {
			logger.Debug("taskList state is stored in cache")
		}
	}

	if err := l.storage.SaveTaskListHistory(ctx, list); err != nil {
		logger.Error("error saving taskList history", zap.Error(err))
	} else {
		logger.Debug("taskList history is saved")
	}

	if err := l.queue.AckList(ctx, list); err != nil {
		logger.Error("failed to ack taskList in queue", zap.Error(err))
	}
}

func (l *ListExecutor) executeGroup(ctx context.Context, group *service.Group) (*goatasklist.GroupStatus, error) {
	switch exec := group.Execution; exec {
	case sequential:
		return l.executeSequential(ctx, group)
	case parallel:
		return l.executeParallel(ctx, group)
	}

	return nil, errors.New("unknown type of group execution")
}

func (l *ListExecutor) executeSequential(ctx context.Context, group *service.Group) (*goatasklist.GroupStatus, error) {
	group.State = service.Pending
	var state goatasklist.GroupStatus

	tasks, err := l.storage.GetGroupTasks(ctx, group)
	if err != nil {
		return nil, err
	}

	req := group.Request
	for _, task := range tasks {
		task := task
		taskState := goatasklist.TaskStatus{ID: &task.ID}

		logger := l.logger.With(
			zap.String("taskID", task.ID),
			zap.String("taskName", task.Name),
		)

		// mark all subsequent tasks as failed if one task already failed
		if group.State == service.Failed {
			task.State = service.Failed
			taskState.Status = ptr.String(service.Failed)
			state.Tasks = append(state.Tasks, &taskState)
			continue
		}

		task.Request = req
		err := l.executeTask(ctx, task)
		if err != nil {
			task.State = service.Failed
			taskState.Status = ptr.String(service.Failed)
			state.Tasks = append(state.Tasks, &taskState)
			group.State = service.Failed
			logger.Error("error executing task", zap.Error(err))
			continue
		}
		logger.Debug("task execution completed successfully")

		taskState.Status = ptr.String(string(task.State))
		state.Tasks = append(state.Tasks, &taskState)

		// pass the response from current task as an input to the next task
		req = task.Response

		if err := l.cache.Set(
			ctx,
			task.ID,
			task.CacheNamespace,
			task.CacheScope,
			task.Response,
		); err != nil {
			logger.Error("error storing task result in cache", zap.Error(err))
			continue
		}
		logger.Debug("task results are stored in cache")

		if err := l.storage.SaveTaskHistory(ctx, task); err != nil {
			logger.Error("error saving task history", zap.Error(err))
			continue
		}
		logger.Debug("task history is saved")
	}

	// remove tasks from queue
	if err := l.queue.AckGroupTasks(ctx, group); err != nil {
		l.logger.With(zap.String("groupID", group.ID)).Error("failed to ack group tasks in queue", zap.Error(err))
	}

	if group.State != service.Failed {
		group.State = service.Done
	}

	state.ID = &group.ID
	state.Status = ptr.String(string(group.State))

	return &state, nil
}

func (l *ListExecutor) executeParallel(ctx context.Context, group *service.Group) (*goatasklist.GroupStatus, error) {
	group.State = service.Pending
	var state goatasklist.GroupStatus

	tasks, err := l.storage.GetGroupTasks(ctx, group)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(t *service.Task) {
			taskState := goatasklist.TaskStatus{
				ID: &t.ID,
			}

			defer wg.Done()
			logger := l.logger.With(
				zap.String("taskID", t.ID),
				zap.String("taskName", t.Name),
			)
			// pass group request to each task
			t.Request = group.Request

			if err := l.executeTask(ctx, t); err != nil {
				t.State = service.Failed
				taskState.Status = ptr.String(service.Failed)
				state.Tasks = append(state.Tasks, &taskState)
				group.State = service.Failed
				logger.Error("error executing task", zap.Error(err))
				return
			}
			logger.Debug("task execution completed successfully")

			taskState.Status = ptr.String(string(t.State))
			state.Tasks = append(state.Tasks, &taskState)

			if err := l.cache.Set(
				ctx,
				t.ID,
				t.CacheNamespace,
				t.CacheScope,
				t.Response,
			); err != nil {
				logger.Error("error storing task result in cache", zap.Error(err))
				return
			}
			logger.Debug("task results are stored in cache")

			if err := l.storage.SaveTaskHistory(ctx, t); err != nil {
				logger.Error("error saving task history", zap.Error(err))
				return
			}
			logger.Debug("task history is saved")
		}(task)
	}

	// wait for all tasks to be executed
	wg.Wait()

	// remove tasks from queue
	if err := l.queue.AckGroupTasks(ctx, group); err != nil {
		l.logger.With(zap.String("groupID", group.ID)).Error("failed to ack group tasks in queue", zap.Error(err))
	}

	if group.State != service.Failed {
		group.State = service.Done
	}

	state.ID = &group.ID
	state.Status = ptr.String(string(group.State))

	return &state, nil
}

func (l *ListExecutor) executeTask(ctx context.Context, task *service.Task) error {
	task.StartedAt = time.Now()

	var response []byte
	var err error

	// task is executing a request policy OR an HTTP call to predefined URL
	if task.RequestPolicy != "" {
		response, err = l.policy.Evaluate(ctx, task.RequestPolicy, task.Request)
		if err != nil {
			return errors.New("error evaluating request policy", err)
		}
		task.ResponseCode = http.StatusOK
	} else if task.URL != "" && task.Method != "" {
		var status int
		status, response, err = l.doHTTPTask(ctx, task)
		if err != nil {
			return err
		}
		task.ResponseCode = status
	} else {
		return errors.New(errors.Internal, "invalid task: must define either request policy or url")
	}

	task.Response = response

	// evaluate response policy
	if task.ResponsePolicy != "" {
		resp, err := l.policy.Evaluate(ctx, task.ResponsePolicy, task.Response)
		if err != nil {
			return errors.New("error evaluating response policy", err)
		}
		// overwrite task response with the one returned from the policy
		task.Response = resp
	}

	// evaluate finalizer policy
	if task.FinalPolicy != "" {
		resp, err := l.policy.Evaluate(ctx, task.FinalPolicy, task.Response)
		if err != nil {
			return errors.New("error evaluating final policy", err)
		}
		// overwrite task response with the one returned from the policy
		task.Response = resp
	}

	task.State = service.Done
	task.FinishedAt = time.Now()
	return nil
}

func (l *ListExecutor) doHTTPTask(ctx context.Context, task *service.Task) (status int, response []byte, err error) {
	reqBody := task.Request
	if task.Method == http.MethodGet {
		reqBody = nil
	}
	req, err := http.NewRequest(task.Method, task.URL, bytes.NewReader(reqBody))
	if err != nil {
		return 0, nil, errors.New("error creating http request", err)
	}

	resp, err := l.httpClient.Do(req.WithContext(ctx))
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
