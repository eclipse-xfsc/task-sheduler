package service

import "context"

//go:generate counterfeiter . Queue

type Queue interface {
	// Task related methods
	Add(ctx context.Context, task *Task) error
	Poll(ctx context.Context) (*Task, error)
	Ack(ctx context.Context, task *Task) error
	Unack(ctx context.Context, task *Task) error

	// TaskList related methods
	AddTaskList(ctx context.Context, taskList *TaskList, tasks []*Task) error
	PollList(ctx context.Context) (*TaskList, error)
	AckList(ctx context.Context, taskList *TaskList) error
	AckGroupTasks(ctx context.Context, group *Group) error
}
