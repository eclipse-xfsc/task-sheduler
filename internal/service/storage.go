package service

import (
	"context"
)

//go:generate counterfeiter . Storage

type Storage interface {
	// Task related methods
	Task(ctx context.Context, taskID string) (*Task, error)
	TaskTemplate(ctx context.Context, taskName string) (*Task, error)
	TaskHistory(ctx context.Context, taskID string) (*Task, error)
	SaveTaskHistory(ctx context.Context, task *Task) error

	// TaskList related methods
	TaskList(ctx context.Context, taskListID string) (*TaskList, error)
	TaskListTemplate(ctx context.Context, taskListName string) (*Template, error)
	TaskTemplates(ctx context.Context, names []string) (map[string]*Task, error)
	TaskListHistory(ctx context.Context, taskListID string) (*TaskList, error)
	GetGroupTasks(ctx context.Context, group *Group) ([]*Task, error)
	SaveTaskListHistory(ctx context.Context, task *TaskList) error

	// EventTask related methods
	EventTask(ctx context.Context, key, namespace, scope string) (*EventTask, error)
}
