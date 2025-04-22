package storage

import (
	"context"
	"strings"

	"github.com/eclipse-xfsc/task-sheduler/internal/service"

	"github.com/cenkalti/backoff/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
)

const (
	taskDB            = "task"
	taskTemplates     = "taskTemplates"
	taskQueue         = "tasks"
	tasksHistory      = "tasksHistory"
	taskListQueue     = "taskLists"
	taskListTemplates = "taskListTemplates"
	taskListHistory   = "taskListHistory"
	eventTasks        = "eventTasks"
)

type Storage struct {
	eventTasks        *mongo.Collection
	taskTemplates     *mongo.Collection
	tasks             *mongo.Collection
	tasksHistory      *mongo.Collection
	taskLists         *mongo.Collection
	taskListTemplates *mongo.Collection
	taskListHistory   *mongo.Collection
}

func New(db *mongo.Client) *Storage {
	return &Storage{
		eventTasks:        db.Database(taskDB).Collection(eventTasks),
		taskTemplates:     db.Database(taskDB).Collection(taskTemplates),
		tasks:             db.Database(taskDB).Collection(taskQueue),
		tasksHistory:      db.Database(taskDB).Collection(tasksHistory),
		taskListTemplates: db.Database(taskDB).Collection(taskListTemplates),
		taskLists:         db.Database(taskDB).Collection(taskListQueue),
		taskListHistory:   db.Database(taskDB).Collection(taskListHistory),
	}
}

func (s *Storage) TaskTemplate(ctx context.Context, taskName string) (*service.Task, error) {
	result := s.taskTemplates.FindOne(ctx, bson.M{
		"name": taskName,
	})

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "task template not found")
		}
		return nil, result.Err()
	}

	var task service.Task
	if err := result.Decode(&task); err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *Storage) Add(ctx context.Context, task *service.Task) error {
	_, err := s.tasks.InsertOne(ctx, task)
	return err
}

// Poll retrieves one task with empty groupID from the tasks collection
// with the older ones being retrieved first (FIFO). It updates the state
// of the task to "pending", so that consequent calls to Poll would
// not retrieve the same task.
func (s *Storage) Poll(ctx context.Context) (*service.Task, error) {
	opts := options.
		FindOneAndUpdate().
		SetSort(bson.M{"createdAt": 1}).
		SetReturnDocument(options.After)

	filter := bson.M{"state": service.Created, "groupid": ""}
	update := bson.M{"$set": bson.M{"state": service.Pending}}
	result := s.tasks.FindOneAndUpdate(
		ctx,
		filter,
		update,
		opts,
	)

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "task not found")
		}
		return nil, result.Err()
	}

	var task service.Task
	if err := result.Decode(&task); err != nil {
		return nil, err
	}

	return &task, nil
}

// Ack removes a task from the `tasks` collection.
func (s *Storage) Ack(ctx context.Context, task *service.Task) error {
	_, err := s.tasks.DeleteOne(ctx, bson.M{"id": task.ID})
	return err
}

// Unack changes the "pending" state of a task to "created", so that
// it can be retrieved for processing again.
func (s *Storage) Unack(ctx context.Context, t *service.Task) error {
	filter := bson.M{"id": t.ID}
	update := bson.M{"$set": bson.M{"state": service.Created, "retries": t.Retries + 1}}
	_, err := s.tasks.UpdateOne(ctx, filter, update)
	return err
}

// SaveTaskHistory saves a task to the `tasksHistory` collection.
func (s *Storage) SaveTaskHistory(ctx context.Context, task *service.Task) error {
	insert := func() error {
		_, err := s.tasksHistory.InsertOne(ctx, task)
		return err
	}

	b := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
	return backoff.Retry(insert, b)
}

func (s *Storage) Task(ctx context.Context, taskID string) (*service.Task, error) {
	result := s.tasks.FindOne(ctx, bson.M{
		"id": taskID,
	})

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "task not found")
		}
		return nil, result.Err()
	}

	var task service.Task
	if err := result.Decode(&task); err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *Storage) TaskHistory(ctx context.Context, taskID string) (*service.Task, error) {
	result := s.tasksHistory.FindOne(ctx, bson.M{
		"id": taskID,
	})

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "task not found")
		}
		return nil, result.Err()
	}

	var task service.Task
	if err := result.Decode(&task); err != nil {
		return nil, err
	}

	return &task, nil
}

// TaskListTemplate retrieves one taskList definition by name from storage
func (s *Storage) TaskListTemplate(ctx context.Context, taskListName string) (*service.Template, error) {
	result := s.taskListTemplates.FindOne(ctx, bson.M{
		"name": taskListName,
	})

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "taskList template not found")
		}
		return nil, result.Err()
	}

	var tasklist service.Template
	if err := result.Decode(&tasklist); err != nil {
		return nil, err
	}

	return &tasklist, nil
}

// TaskTemplates retrieves task definitions from storage by names.
//
// The result is a map where 'key' is the task name and 'value' is the task definition
func (s *Storage) TaskTemplates(ctx context.Context, names []string) (map[string]*service.Task, error) {
	cursor, err := s.taskTemplates.Find(ctx, bson.M{
		"name": bson.M{"$in": names},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	res := make(map[string]*service.Task)
	for cursor.Next(ctx) {
		var task service.Task
		if err := cursor.Decode(&task); err != nil {
			return nil, err
		}
		res[task.Name] = &task
	}

	return res, nil
}

func (s *Storage) AddTaskList(ctx context.Context, taskList *service.TaskList, tasks []*service.Task) error {
	_, err := s.taskLists.InsertOne(ctx, taskList)
	if err != nil {
		return err
	}

	var ti []interface{}
	for _, task := range tasks {
		ti = append(ti, task)
	}

	_, err = s.tasks.InsertMany(ctx, ti)
	if err != nil {
		if err := s.AckList(ctx, taskList); err != nil { // remove taskList from queue
			return errors.New("failed to ack taskList", err)
		}
		return err
	}

	return nil
}

// AckList removes a taskList from the `tasksLists` collection.
func (s *Storage) AckList(ctx context.Context, taskList *service.TaskList) error {
	_, err := s.taskLists.DeleteOne(ctx, bson.M{"id": taskList.ID})
	return err
}

// PollList retrieves one taskList from the taskLists collection
// with the older ones being retrieved first (FIFO). It updates the state
// of the task to "pending", so that consequent calls to PollList would
// not retrieve the same task.
func (s *Storage) PollList(ctx context.Context) (*service.TaskList, error) {
	opts := options.
		FindOneAndUpdate().
		SetSort(bson.M{"createdAt": 1}).
		SetReturnDocument(options.After)

	filter := bson.M{"state": service.Created}
	update := bson.M{"$set": bson.M{"state": service.Pending}}
	result := s.taskLists.FindOneAndUpdate(
		ctx,
		filter,
		update,
		opts,
	)

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "taskList not found")
		}
		return nil, result.Err()
	}

	var list service.TaskList
	if err := result.Decode(&list); err != nil {
		return nil, err
	}

	return &list, nil
}

// GetGroupTasks fetches all tasks by a groupID
func (s *Storage) GetGroupTasks(ctx context.Context, group *service.Group) ([]*service.Task, error) {
	filter := bson.M{"groupid": group.ID}
	opts := options.Find().SetSort(bson.M{"createdAt": 1})

	cursor, err := s.tasks.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []*service.Task
	for cursor.Next(ctx) {
		var task service.Task
		if err := cursor.Decode(&task); err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// AckGroupTasks removes tasks from tasks collection by groupID
func (s *Storage) AckGroupTasks(ctx context.Context, group *service.Group) error {
	_, err := s.tasks.DeleteMany(ctx, bson.M{"groupid": group.ID})
	return err
}

// SaveTaskListHistory adds a tasklist to the taskListHistory collection
func (s *Storage) SaveTaskListHistory(ctx context.Context, taskList *service.TaskList) error {
	insert := func() error {
		_, err := s.taskListHistory.InsertOne(ctx, taskList)
		return err
	}

	b := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
	return backoff.Retry(insert, b)
}

// TaskList retrieves a tasklist.TaskList from taskLists collection by ID
func (s *Storage) TaskList(ctx context.Context, taskListID string) (*service.TaskList, error) {
	result := s.taskLists.FindOne(ctx, bson.M{
		"id": taskListID,
	})

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "taskList not found")
		}
		return nil, result.Err()
	}

	var list service.TaskList
	if err := result.Decode(&list); err != nil {
		return nil, err
	}

	return &list, nil
}

// TaskListHistory retrieves a tasklist.TaskList from taskListHistory collection by ID
func (s *Storage) TaskListHistory(ctx context.Context, taskListID string) (*service.TaskList, error) {
	result := s.taskListHistory.FindOne(ctx, bson.M{
		"id": taskListID,
	})

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "taskList not found")
		}
		return nil, result.Err()
	}

	var list service.TaskList
	if err := result.Decode(&list); err != nil {
		return nil, err
	}

	return &list, nil
}

func (s *Storage) EventTask(ctx context.Context, key, namespace, scope string) (*service.EventTask, error) {
	result := s.eventTasks.FindOne(ctx, bson.M{
		"key":       key,
		"namespace": namespace,
		"scope":     scope,
	})

	if result.Err() != nil {
		if strings.Contains(result.Err().Error(), "no documents in result") {
			return nil, errors.New(errors.NotFound, "eventTask not found")
		}
		return nil, result.Err()
	}

	var eventTask service.EventTask
	if err := result.Decode(&eventTask); err != nil {
		return nil, err
	}

	return &eventTask, nil
}
