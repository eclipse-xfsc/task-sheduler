package tasklist

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	ptr "github.com/eclipse-xfsc/microservice-core-go/pkg/ptr"
	goatasklist "github.com/eclipse-xfsc/task-sheduler/gen/task_list"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
)

//go:generate counterfeiter . Cache

type Queue interface {
	AddTaskList(ctx context.Context, taskList *service.TaskList, tasks []*service.Task) error
}

type Cache interface {
	Get(ctx context.Context, key, namespace, scope string) ([]byte, error)
}

type Service struct {
	storage service.Storage
	queue   Queue
	cache   Cache

	logger *zap.Logger
}

func New(template service.Storage, queue Queue, cache Cache, logger *zap.Logger) *Service {
	return &Service{
		storage: template,
		queue:   queue,
		cache:   cache,
		logger:  logger,
	}
}

// Create a taskList and corresponding tasks and put them in
// respective queues for execution.
func (s *Service) Create(ctx context.Context, req *goatasklist.CreateTaskListRequest) (*goatasklist.CreateTaskListResult, error) {
	if req.TaskListName == "" {
		return nil, errors.New(errors.BadRequest, "missing taskListName")
	}

	logger := s.logger.With(zap.String("taskListName", req.TaskListName))

	// get predefined taskList definition from storage
	template, err := s.storage.TaskListTemplate(ctx, req.TaskListName)
	if err != nil {
		logger.Error("error getting taskList template from storage", zap.Error(err))
		return nil, err
	}

	// get predefined task definitions from storage
	taskTemplates, err := s.storage.TaskTemplates(ctx, taskNamesFromTaskListTemplate(template))
	if err != nil {
		logger.Error("error getting task templates from storage")
		return nil, err
	}

	taskListRequest, err := json.Marshal(req.Data)
	if err != nil {
		logger.Error("error marshaling request data to JSON", zap.Error(err))
		return nil, errors.New(errors.BadRequest, "error marshaling request data to JSON", err)
	}

	taskList := &service.TaskList{
		ID:             uuid.NewString(),
		Groups:         createGroups(template, taskListRequest),
		Name:           template.Name,
		Request:        taskListRequest,
		CacheScope:     template.CacheScope,
		CacheNamespace: template.CacheNamespace,
		State:          service.Created,
		CreatedAt:      time.Now(),
	}

	// if cache namespace and scope are given, use them instead of the defaults
	if req.CacheNamespace != nil && *req.CacheNamespace != "" {
		taskList.CacheNamespace = *req.CacheNamespace
	}
	if req.CacheScope != nil && *req.CacheScope != "" {
		taskList.CacheScope = *req.CacheScope
	}

	tasks, err := createTasks(taskList, taskTemplates)
	if err != nil {
		logger.Error("failed to create tasks for taskList", zap.Error(err))
		return nil, errors.New("failed to create tasks for taskList", err)
	}

	if err := s.queue.AddTaskList(ctx, taskList, tasks); err != nil {
		logger.Error("error adding taskList to queue", zap.Error(err))
		return nil, errors.New("error adding taskList to queue", err)
	}

	return &goatasklist.CreateTaskListResult{
		TaskListID: taskList.ID,
	}, nil
}

// TaskListStatus retrieves a taskList result containing all tasks' unique IDs
// and statuses from the Cache service.
func (s *Service) TaskListStatus(ctx context.Context, req *goatasklist.TaskListStatusRequest) (res *goatasklist.TaskListStatusResponse, err error) {
	if req.TaskListID == "" {
		return nil, errors.New(errors.BadRequest, "missing taskListID")
	}

	logger := s.logger.With(zap.String("taskListID", req.TaskListID))

	var list *service.TaskList
	list, err = s.storage.TaskListHistory(ctx, req.TaskListID)
	if err != nil && !errors.Is(errors.NotFound, err) {
		logger.Error("error getting taskList from history collection", zap.Error(err))
		return nil, err
	}

	if list == nil {
		list, err = s.storage.TaskList(ctx, req.TaskListID)
		if err != nil {
			if errors.Is(errors.NotFound, err) {
				return nil, errors.New("taskList is not found", err)
			}
			logger.Error("error getting taskList from taskLists collection", zap.Error(err))
			return nil, err
		}
	}

	var result *goatasklist.TaskListStatusResponse
	if list.State != service.Done && list.State != service.Failed {
		// taskList is not executed yet
		result, err = s.calculateState(ctx, list)
		if err != nil {
			logger.Error("error calculating taskList state", zap.Error(err))
			return nil, err
		}
	} else {
		// taskList is already executed
		var value []byte
		value, err = s.cache.Get(ctx, list.ID, list.CacheNamespace, list.CacheScope)
		if err != nil {
			logger.Error("error getting taskList result from cache", zap.Error(err))
			return nil, err
		}

		if err := json.NewDecoder(bytes.NewReader(value)).Decode(&result); err != nil {
			logger.Error("error decoding result from cache", zap.Error(err))
			return nil, errors.New("error decoding result from cache", err)
		}
	}

	return result, nil
}

func createGroups(t *service.Template, req []byte) []service.Group {
	var groups []service.Group
	for _, group := range t.Groups {
		g := service.Group{
			ID:          uuid.NewString(),
			Execution:   group.Execution,
			Tasks:       group.Tasks,
			State:       service.Created,
			Request:     req,
			FinalPolicy: group.FinalPolicy,
		}
		groups = append(groups, g)
	}

	return groups
}

// createTasks creates task.Task instances out of task templates
// in order to be added to queue for execution
func createTasks(t *service.TaskList, templates map[string]*service.Task) ([]*service.Task, error) {
	var tasks []*service.Task
	for _, group := range t.Groups {
		for _, taskName := range group.Tasks {
			template, ok := templates[taskName]
			if !ok {
				return nil, errors.New(errors.NotFound, "failed to find task template")
			}

			task := service.Task{
				ID:             uuid.NewString(),
				GroupID:        group.ID,
				Name:           taskName,
				State:          service.Created,
				URL:            template.URL,
				Method:         template.Method,
				RequestPolicy:  template.RequestPolicy,
				ResponsePolicy: template.ResponsePolicy,
				FinalPolicy:    template.FinalPolicy,
				CacheNamespace: template.CacheNamespace,
				CacheScope:     template.CacheScope,
				CreatedAt:      time.Now(),
			}

			// if cache namespace and scope are set in the taskList, use them instead of the defaults
			if t.CacheNamespace != "" {
				task.CacheNamespace = t.CacheNamespace
			}
			if t.CacheScope != "" {
				task.CacheScope = t.CacheScope
			}

			tasks = append(tasks, &task)
		}
	}

	return tasks, nil
}

func (s *Service) calculateState(ctx context.Context, list *service.TaskList) (*goatasklist.TaskListStatusResponse, error) {
	result := &goatasklist.TaskListStatusResponse{
		ID:     list.ID,
		Status: string(list.State),
	}

	for i := range list.Groups {
		groupState := goatasklist.GroupStatus{
			ID:     &list.Groups[i].ID,
			Status: ptr.String(string(list.Groups[i].State)),
		}

		tasks, err := s.storage.GetGroupTasks(ctx, &list.Groups[i])
		if err != nil {
			return nil, err
		}
		for j := range tasks {
			taskState := goatasklist.TaskStatus{
				ID:     &tasks[j].ID,
				Status: ptr.String(string(tasks[j].State)),
			}
			groupState.Tasks = append(groupState.Tasks, &taskState)
		}

		result.Groups = append(result.Groups, &groupState)
	}

	return result, nil
}

// taskNamesFromTaskListTemplate returns the names of all tasks within
// one taskList template
func taskNamesFromTaskListTemplate(template *service.Template) []string {
	var names []string
	for _, group := range template.Groups {
		names = append(names, group.Tasks...)
	}

	return names
}
