package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	ptr "github.com/eclipse-xfsc/microservice-core-go/pkg/ptr"
	goatask "github.com/eclipse-xfsc/task-sheduler/gen/task"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/servicefakes"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/task"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/task/taskfakes"
)

func TestNew(t *testing.T) {
	svc := task.New(nil, nil, nil, zap.NewNop())
	assert.Implements(t, (*goatask.Service)(nil), svc)
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		req     *goatask.CreateTaskRequest
		storage *servicefakes.FakeStorage
		queue   *servicefakes.FakeQueue
		cache   *taskfakes.FakeCache

		errkind errors.Kind
		errtext string
	}{
		{
			name:    "empty task name",
			req:     &goatask.CreateTaskRequest{},
			errkind: errors.BadRequest,
			errtext: "missing taskName",
		},
		{
			name: "task template not found",
			req:  &goatask.CreateTaskRequest{TaskName: "taskname"},
			storage: &servicefakes.FakeStorage{
				TaskTemplateStub: func(ctx context.Context, taskName string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
			},
			errkind: errors.NotFound,
			errtext: "not found",
		},
		{
			name: "fail to add task to queue",
			req:  &goatask.CreateTaskRequest{TaskName: "taskname"},
			storage: &servicefakes.FakeStorage{
				TaskTemplateStub: func(ctx context.Context, taskName string) (*service.Task, error) {
					return &service.Task{}, nil
				},
			},
			queue: &servicefakes.FakeQueue{
				AddStub: func(ctx context.Context, t *service.Task) error {
					return errors.New("some error")
				},
			},
			errkind: errors.Unknown,
			errtext: "some error",
		},
		{
			name: "successfully add task to queue",
			req:  &goatask.CreateTaskRequest{TaskName: "taskname"},
			storage: &servicefakes.FakeStorage{
				TaskTemplateStub: func(ctx context.Context, taskName string) (*service.Task, error) {
					return &service.Task{}, nil
				},
			},
			queue: &servicefakes.FakeQueue{
				AddStub: func(ctx context.Context, t *service.Task) error {
					return nil
				},
			},
		},
		{
			name: "successfully add task to queue with namespace and scope",
			req: &goatask.CreateTaskRequest{
				TaskName:       "taskname",
				CacheNamespace: ptr.String("login"),
				CacheScope:     ptr.String("user"),
			},
			storage: &servicefakes.FakeStorage{
				TaskTemplateStub: func(ctx context.Context, taskName string) (*service.Task, error) {
					return &service.Task{}, nil
				},
			},
			queue: &servicefakes.FakeQueue{
				AddStub: func(ctx context.Context, t *service.Task) error {
					return nil
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := task.New(test.storage, test.queue, test.cache, zap.NewNop())
			res, err := svc.Create(context.Background(), test.req)
			if err != nil {
				assert.NotEmpty(t, test.errtext)
				e, ok := err.(*errors.Error)
				assert.True(t, ok)
				assert.Equal(t, test.errkind, e.Kind)
				assert.Contains(t, e.Error(), test.errtext)
				assert.Nil(t, res)
			} else {
				assert.Empty(t, test.errtext)
				assert.NotNil(t, res)
				assert.NotEmpty(t, res.TaskID)
			}
		})
	}
}

func TestService_TaskResult(t *testing.T) {
	tests := []struct {
		name    string
		req     *goatask.TaskResultRequest
		storage *servicefakes.FakeStorage
		cache   *taskfakes.FakeCache

		res     interface{}
		errkind errors.Kind
		errtext string
	}{
		{
			name:    "missing taskID",
			req:     &goatask.TaskResultRequest{},
			errkind: errors.BadRequest,
			errtext: "missing taskID",
		},
		{
			name: "error getting task history from storage",
			req:  &goatask.TaskResultRequest{TaskID: "123"},
			storage: &servicefakes.FakeStorage{
				TaskHistoryStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New("some error")
				},
			},
			errkind: errors.Unknown,
			errtext: "some error",
		},
		{
			name: "task not found in history and fail to retrieve it from tasks queue collection too",
			req:  &goatask.TaskResultRequest{TaskID: "123"},
			storage: &servicefakes.FakeStorage{
				TaskHistoryStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New("another error")
				},
			},
			errkind: errors.Unknown,
			errtext: "another error",
		},
		{
			name: "task not found neither in history nor in tasks queue collection",
			req:  &goatask.TaskResultRequest{TaskID: "123"},
			storage: &servicefakes.FakeStorage{
				TaskHistoryStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
			},
			errkind: errors.NotFound,
			errtext: "task is not found",
		},
		{
			name: "task is not yet completed",
			req:  &goatask.TaskResultRequest{TaskID: "123"},
			storage: &servicefakes.FakeStorage{
				TaskHistoryStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return &service.Task{State: service.Pending}, nil
				},
			},
			errkind: errors.NotFound,
			errtext: "no result, task is not completed",
		},
		{
			name: "error getting task result from cache",
			req:  &goatask.TaskResultRequest{TaskID: "123"},
			storage: &servicefakes.FakeStorage{
				TaskHistoryStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return &service.Task{State: service.Done}, nil
				},
			},
			cache: &taskfakes.FakeCache{
				GetStub: func(ctx context.Context, key string, ns string, scope string) ([]byte, error) {
					return nil, errors.New("cache error")
				},
			},
			errkind: errors.Unknown,
			errtext: "cache error",
		},
		{
			name: "getting invalid JSON result from cache",
			req:  &goatask.TaskResultRequest{TaskID: "123"},
			storage: &servicefakes.FakeStorage{
				TaskHistoryStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return &service.Task{State: service.Done}, nil
				},
			},
			cache: &taskfakes.FakeCache{
				GetStub: func(ctx context.Context, key string, ns string, scope string) ([]byte, error) {
					return []byte("asdfads"), nil
				},
			},
			errkind: errors.Unknown,
			errtext: "error decoding result from cache",
		},
		{
			name: "get task result successfully",
			req:  &goatask.TaskResultRequest{TaskID: "123"},
			storage: &servicefakes.FakeStorage{
				TaskHistoryStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskStub: func(ctx context.Context, taskID string) (*service.Task, error) {
					return &service.Task{State: service.Done}, nil
				},
			},
			cache: &taskfakes.FakeCache{
				GetStub: func(ctx context.Context, key string, ns string, scope string) ([]byte, error) {
					return []byte(`{"result":"success"}`), nil
				},
			},
			res: map[string]interface{}{"result": "success"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := task.New(test.storage, nil, test.cache, zap.NewNop())
			res, err := svc.TaskResult(context.Background(), test.req)
			if err != nil {
				assert.NotEmpty(t, test.errtext)
				e, ok := err.(*errors.Error)
				assert.True(t, ok)
				assert.Equal(t, test.errkind, e.Kind)
				assert.Contains(t, e.Error(), test.errtext)
				assert.Nil(t, res)
			} else {
				assert.Empty(t, test.errtext)
				assert.NotNil(t, res)
				assert.Equal(t, test.res, res)
			}
		})
	}
}
