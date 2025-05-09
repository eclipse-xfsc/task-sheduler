// Code generated by goa v3.20.1, DO NOT EDIT.
//
// taskList HTTP server types
//
// Command:
// $ goa gen github.com/eclipse-xfsc/task-sheduler/design

package server

import (
	tasklist "github.com/eclipse-xfsc/task-sheduler/gen/task_list"
)

// CreateResponseBody is the type of the "taskList" service "Create" endpoint
// HTTP response body.
type CreateResponseBody struct {
	// Unique taskList identifier.
	TaskListID string `form:"taskListID" json:"taskListID" xml:"taskListID"`
}

// TaskListStatusMultiStatusResponseBody is the type of the "taskList" service
// "TaskListStatus" endpoint HTTP response body.
type TaskListStatusMultiStatusResponseBody struct {
	// Unique taskList identifier.
	ID string `form:"id" json:"id" xml:"id"`
	// Current status of the taskList
	Status string `form:"status" json:"status" xml:"status"`
	// Array of GroupStatus
	Groups []*GroupStatusResponseBody `form:"groups,omitempty" json:"groups,omitempty" xml:"groups,omitempty"`
}

// TaskListStatusCreatedResponseBody is the type of the "taskList" service
// "TaskListStatus" endpoint HTTP response body.
type TaskListStatusCreatedResponseBody struct {
	// Unique taskList identifier.
	ID string `form:"id" json:"id" xml:"id"`
	// Current status of the taskList
	Status string `form:"status" json:"status" xml:"status"`
	// Array of GroupStatus
	Groups []*GroupStatusResponseBody `form:"groups,omitempty" json:"groups,omitempty" xml:"groups,omitempty"`
}

// TaskListStatusAcceptedResponseBody is the type of the "taskList" service
// "TaskListStatus" endpoint HTTP response body.
type TaskListStatusAcceptedResponseBody struct {
	// Unique taskList identifier.
	ID string `form:"id" json:"id" xml:"id"`
	// Current status of the taskList
	Status string `form:"status" json:"status" xml:"status"`
	// Array of GroupStatus
	Groups []*GroupStatusResponseBody `form:"groups,omitempty" json:"groups,omitempty" xml:"groups,omitempty"`
}

// TaskListStatusOKResponseBody is the type of the "taskList" service
// "TaskListStatus" endpoint HTTP response body.
type TaskListStatusOKResponseBody struct {
	// Unique taskList identifier.
	ID string `form:"id" json:"id" xml:"id"`
	// Current status of the taskList
	Status string `form:"status" json:"status" xml:"status"`
	// Array of GroupStatus
	Groups []*GroupStatusResponseBody `form:"groups,omitempty" json:"groups,omitempty" xml:"groups,omitempty"`
}

// GroupStatusResponseBody is used to define fields on response body types.
type GroupStatusResponseBody struct {
	// Unique group identifier.
	ID *string `form:"id,omitempty" json:"id,omitempty" xml:"id,omitempty"`
	// Current status of the group
	Status *string `form:"status,omitempty" json:"status,omitempty" xml:"status,omitempty"`
	// Array of TaskStatus
	Tasks []*TaskStatusResponseBody `form:"tasks,omitempty" json:"tasks,omitempty" xml:"tasks,omitempty"`
}

// TaskStatusResponseBody is used to define fields on response body types.
type TaskStatusResponseBody struct {
	// Unique task identifier.
	ID *string `form:"id,omitempty" json:"id,omitempty" xml:"id,omitempty"`
	// Current status of the task
	Status *string `form:"status,omitempty" json:"status,omitempty" xml:"status,omitempty"`
}

// NewCreateResponseBody builds the HTTP response body from the result of the
// "Create" endpoint of the "taskList" service.
func NewCreateResponseBody(res *tasklist.CreateTaskListResult) *CreateResponseBody {
	body := &CreateResponseBody{
		TaskListID: res.TaskListID,
	}
	return body
}

// NewTaskListStatusMultiStatusResponseBody builds the HTTP response body from
// the result of the "TaskListStatus" endpoint of the "taskList" service.
func NewTaskListStatusMultiStatusResponseBody(res *tasklist.TaskListStatusResponse) *TaskListStatusMultiStatusResponseBody {
	body := &TaskListStatusMultiStatusResponseBody{
		ID:     res.ID,
		Status: res.Status,
	}
	if res.Groups != nil {
		body.Groups = make([]*GroupStatusResponseBody, len(res.Groups))
		for i, val := range res.Groups {
			body.Groups[i] = marshalTasklistGroupStatusToGroupStatusResponseBody(val)
		}
	}
	return body
}

// NewTaskListStatusCreatedResponseBody builds the HTTP response body from the
// result of the "TaskListStatus" endpoint of the "taskList" service.
func NewTaskListStatusCreatedResponseBody(res *tasklist.TaskListStatusResponse) *TaskListStatusCreatedResponseBody {
	body := &TaskListStatusCreatedResponseBody{
		ID:     res.ID,
		Status: res.Status,
	}
	if res.Groups != nil {
		body.Groups = make([]*GroupStatusResponseBody, len(res.Groups))
		for i, val := range res.Groups {
			body.Groups[i] = marshalTasklistGroupStatusToGroupStatusResponseBody(val)
		}
	}
	return body
}

// NewTaskListStatusAcceptedResponseBody builds the HTTP response body from the
// result of the "TaskListStatus" endpoint of the "taskList" service.
func NewTaskListStatusAcceptedResponseBody(res *tasklist.TaskListStatusResponse) *TaskListStatusAcceptedResponseBody {
	body := &TaskListStatusAcceptedResponseBody{
		ID:     res.ID,
		Status: res.Status,
	}
	if res.Groups != nil {
		body.Groups = make([]*GroupStatusResponseBody, len(res.Groups))
		for i, val := range res.Groups {
			body.Groups[i] = marshalTasklistGroupStatusToGroupStatusResponseBody(val)
		}
	}
	return body
}

// NewTaskListStatusOKResponseBody builds the HTTP response body from the
// result of the "TaskListStatus" endpoint of the "taskList" service.
func NewTaskListStatusOKResponseBody(res *tasklist.TaskListStatusResponse) *TaskListStatusOKResponseBody {
	body := &TaskListStatusOKResponseBody{
		ID:     res.ID,
		Status: res.Status,
	}
	if res.Groups != nil {
		body.Groups = make([]*GroupStatusResponseBody, len(res.Groups))
		for i, val := range res.Groups {
			body.Groups[i] = marshalTasklistGroupStatusToGroupStatusResponseBody(val)
		}
	}
	return body
}

// NewCreateTaskListRequest builds a taskList service Create endpoint payload.
func NewCreateTaskListRequest(body any, taskListName string, cacheNamespace *string, cacheScope *string) *tasklist.CreateTaskListRequest {
	v := body
	res := &tasklist.CreateTaskListRequest{
		Data: v,
	}
	res.TaskListName = taskListName
	res.CacheNamespace = cacheNamespace
	res.CacheScope = cacheScope

	return res
}

// NewTaskListStatusRequest builds a taskList service TaskListStatus endpoint
// payload.
func NewTaskListStatusRequest(taskListID string) *tasklist.TaskListStatusRequest {
	v := &tasklist.TaskListStatusRequest{}
	v.TaskListID = taskListID

	return v
}
