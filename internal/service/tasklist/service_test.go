package tasklist_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	goatasklist "github.com/eclipse-xfsc/task-sheduler/gen/task_list"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/servicefakes"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/tasklist"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/tasklist/tasklistfakes"
)

func TestNew(t *testing.T) {
	svc := tasklist.New(nil, nil, nil, zap.NewNop())
	assert.Implements(t, (*goatasklist.Service)(nil), svc)
}

func Test_Create(t *testing.T) {
	tests := []struct {
		name    string
		req     *goatasklist.CreateTaskListRequest
		storage *servicefakes.FakeStorage
		queue   *servicefakes.FakeQueue

		errkind errors.Kind
		errtext string
	}{
		{
			name:    "empty taskList name",
			req:     &goatasklist.CreateTaskListRequest{},
			errkind: errors.BadRequest,
			errtext: "missing taskListName",
		},
		{
			name: "taskList template not found",
			req:  &goatasklist.CreateTaskListRequest{TaskListName: "taskList name"},
			storage: &servicefakes.FakeStorage{
				TaskListTemplateStub: func(ctx context.Context, s string) (*service.Template, error) {
					return nil, errors.New(errors.NotFound)
				},
			},
			errkind: errors.NotFound,
			errtext: "not found",
		},
		{
			name: "error getting task templates form storage",
			req:  &goatasklist.CreateTaskListRequest{TaskListName: "taskList name"},
			storage: &servicefakes.FakeStorage{
				TaskListTemplateStub: func(ctx context.Context, s string) (*service.Template, error) {
					return &service.Template{}, nil
				},
				TaskTemplatesStub: func(ctx context.Context, strings []string) (map[string]*service.Task, error) {
					return nil, errors.New(errors.Internal, "internal error")
				},
			},
			errkind: errors.Internal,
			errtext: "internal error",
		},
		{
			name: "error creating tasks for a taskList, task template not found",
			req:  &goatasklist.CreateTaskListRequest{TaskListName: "taskList name"},
			storage: &servicefakes.FakeStorage{
				TaskListTemplateStub: func(ctx context.Context, s string) (*service.Template, error) {
					return &service.Template{
						Groups: []service.GroupTemplate{
							{
								Tasks: []string{"non-existent task template"},
							},
						},
					}, nil
				},
				TaskTemplatesStub: func(ctx context.Context, strings []string) (map[string]*service.Task, error) {
					return map[string]*service.Task{"template": &service.Task{}}, nil
				},
			},
			errkind: errors.NotFound,
			errtext: "failed to find task template",
		},
		{
			name: "failed to add taskList and tasks to queue",
			req:  &goatasklist.CreateTaskListRequest{TaskListName: "taskList name"},
			storage: &servicefakes.FakeStorage{
				TaskListTemplateStub: func(ctx context.Context, s string) (*service.Template, error) {
					return &service.Template{}, nil
				},
				TaskTemplatesStub: func(ctx context.Context, strings []string) (map[string]*service.Task, error) {
					return map[string]*service.Task{"template": &service.Task{}}, nil
				},
			},
			queue: &servicefakes.FakeQueue{
				AddTaskListStub: func(ctx context.Context, list *service.TaskList, tasks []*service.Task) error {
					return errors.New("storage error")
				},
			},
			errkind: errors.Unknown,
			errtext: "storage error",
		},
		{
			name: "successfully add taskList and tasks to queue",
			req:  &goatasklist.CreateTaskListRequest{TaskListName: "taskList name"},
			storage: &servicefakes.FakeStorage{
				TaskListTemplateStub: func(ctx context.Context, s string) (*service.Template, error) {
					return &service.Template{}, nil
				},
				TaskTemplatesStub: func(ctx context.Context, strings []string) (map[string]*service.Task, error) {
					return map[string]*service.Task{"template": &service.Task{}}, nil
				},
			},
			queue: &servicefakes.FakeQueue{
				AddTaskListStub: func(ctx context.Context, list *service.TaskList, tasks []*service.Task) error {
					return nil
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := tasklist.New(test.storage, test.queue, nil, zap.NewNop())
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
				assert.NotEmpty(t, res.TaskListID)
			}

		})
	}
}

func Test_TaskListStatus(t *testing.T) {
	tests := []struct {
		name    string
		req     *goatasklist.TaskListStatusRequest
		storage *servicefakes.FakeStorage
		queue   *servicefakes.FakeQueue
		cache   *tasklistfakes.FakeCache

		errkind errors.Kind
		errtext string
	}{
		{
			name:    "missing taskList ID",
			req:     &goatasklist.TaskListStatusRequest{},
			errkind: errors.BadRequest,
			errtext: "missing taskListID",
		},
		{
			name: "error getting taskList form history collection",
			req:  &goatasklist.TaskListStatusRequest{TaskListID: "d16996cd-1977-42a9-90b2-b4548a35c1b4"},
			storage: &servicefakes.FakeStorage{
				TaskListHistoryStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return nil, errors.New("some error")
				},
			},
			errkind: errors.Unknown,
			errtext: "some error",
		},
		{
			name: "taskList not found",
			req:  &goatasklist.TaskListStatusRequest{TaskListID: "d16996cd-1977-42a9-90b2-b4548a35c1b4"},
			storage: &servicefakes.FakeStorage{
				TaskListHistoryStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskListStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return nil, errors.New(errors.NotFound)
				},
			},
			errkind: errors.NotFound,
			errtext: "taskList is not found",
		},
		{
			name: "error getting taskList from taskLists collection",
			req:  &goatasklist.TaskListStatusRequest{TaskListID: "d16996cd-1977-42a9-90b2-b4548a35c1b4"},
			storage: &servicefakes.FakeStorage{
				TaskListHistoryStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return nil, errors.New(errors.NotFound)
				},
				TaskListStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return nil, errors.New("some error")
				},
			},
			errkind: errors.Unknown,
			errtext: "some error",
		},
		{
			name: "error calculating taskList state",
			req:  &goatasklist.TaskListStatusRequest{TaskListID: "d16996cd-1977-42a9-90b2-b4548a35c1b4"},
			storage: &servicefakes.FakeStorage{
				TaskListHistoryStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return pendingTaskList, nil
				},
				GetGroupTasksStub: func(ctx context.Context, group *service.Group) ([]*service.Task, error) {
					return nil, errors.New("some error")
				},
			},
			errkind: errors.Unknown,
			errtext: "some error",
		},
		{
			name: "error getting taskList from cache",
			req:  &goatasklist.TaskListStatusRequest{TaskListID: "d16996cd-1977-42a9-90b2-b4548a35c1b4"},
			storage: &servicefakes.FakeStorage{
				TaskListHistoryStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return doneTaskList, nil
				},
			},
			cache: &tasklistfakes.FakeCache{
				GetStub: func(ctx context.Context, key, namespace, scope string) ([]byte, error) {
					return nil, errors.New("some cache error")
				},
			},
			errkind: errors.Unknown,
			errtext: "some cache error",
		},
		{
			name: "successfully get taskList state on pending task",
			req:  &goatasklist.TaskListStatusRequest{TaskListID: "d16996cd-1977-42a9-90b2-b4548a35c1b4"},
			storage: &servicefakes.FakeStorage{
				TaskListHistoryStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return pendingTaskList, nil
				},
				GetGroupTasksStub: func(ctx context.Context, group *service.Group) ([]*service.Task, error) {
					return []*service.Task{}, nil
				},
			},
		},
		{
			name: "successfully get taskList state on executed task",
			req:  &goatasklist.TaskListStatusRequest{TaskListID: "d16996cd-1977-42a9-90b2-b4548a35c1b4"},
			storage: &servicefakes.FakeStorage{
				TaskListHistoryStub: func(ctx context.Context, taskListID string) (*service.TaskList, error) {
					return doneTaskList, nil
				},
			},
			cache: &tasklistfakes.FakeCache{
				GetStub: func(ctx context.Context, key, namespace, scope string) ([]byte, error) {
					return doneTaskState, nil
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := tasklist.New(test.storage, test.queue, test.cache, zap.NewNop())
			res, err := svc.TaskListStatus(context.Background(), test.req)
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
				assert.NotEmpty(t, res.ID)
				assert.NotEmpty(t, res.Status)
				assert.NotEmpty(t, res.Groups)
			}
		})
	}
}

//nolint:gosec
var pendingTaskList = &service.TaskList{
	ID:    "16996cd-1977-42a9-90b2-b4548a35c1b4",
	State: "pending",
	Groups: []service.Group{
		{
			ID:    "074076d5-c995-4d2d-8d38-da57360453d4",
			Tasks: []string{"createdTask", "createdTask2"},
			State: "created",
		},
	},
}

//nolint:gosec
var doneTaskList = &service.TaskList{
	ID:    "16996cd-1977-42a9-90b2-b4548a35c1b4",
	State: "done",
}

//nolint:gosec
var doneTaskState = []byte(`{
  "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
  "status": "done",
  "groups": [
	{
	  "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
	  "type": "sequential",
	  "status": "done",
	  "tasks": [
		{
		  "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
		  "status": "done"
		},
		{
		  "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
		  "status": "done"
		}
	  ]
	}
  ]
}`)
