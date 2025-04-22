package service

import (
	"strings"
	"time"
)

type State string

const (
	// Created is the initial task state.
	Created = "created"

	// Pending state is when a worker has marked the task for processing
	// indicating to other workers that they should not process the task.
	Pending = "pending"

	// Done state is when the task is completed.
	// TODO(penkovski): do we need this state if task is deleted after it is done?
	Done = "done"

	// Failed state is when the task execution failed but the task is part
	// of a group and later execution is not possible, so it could not be "Unack"-ed
	Failed = "failed"
)

type Task struct {
	ID             string    `json:"id"`             // ID is unique task identifier.
	GroupID        string    `json:"groupID"`        // GroupID is set when the task is part of `tasklist.Group`.
	Name           string    `json:"name"`           // Name is used by external callers use to create tasks.
	State          State     `json:"state"`          // State of the task.
	URL            string    `json:"url"`            // URL against which the task request will be executed (optional).
	Method         string    `json:"method"`         // HTTP method of the task request (optional).
	Request        []byte    `json:"request"`        // Request body which will be sent in the task request.
	Response       []byte    `json:"response"`       // Response received after the task request is executed.
	ResponseCode   int       `json:"responseCode"`   // ResponseCode received after task request is executed.
	RequestPolicy  string    `json:"requestPolicy"`  // RequestPolicy to be executed before task request execution.
	ResponsePolicy string    `json:"responsePolicy"` // ResponsePolicy to be executed on the task response.
	FinalPolicy    string    `json:"finalPolicy"`    // FinalPolicy to be executed on the task response.
	CacheNamespace string    `json:"cacheNamespace"` // CacheNamespace if set, is used for constructing cache key.
	CacheScope     string    `json:"cacheScope"`     // CacheScope if set, is used for constructing cache key.
	Retries        int       `json:"retries"`        // Retries is the number of failed attempts to execute this task
	CreatedAt      time.Time `json:"createdAt"`      // CreatedAt specifies task creation time.
	StartedAt      time.Time `json:"startedAt"`      // StartedAt specifies task execution start time.
	FinishedAt     time.Time `json:"finishedAt"`     // FinishedAt specifies the time when the task is done.
}

type EventTask struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Scope     string `json:"scope"`
	TaskName  string
}

// CacheKey constructs the key for storing task result in the cache.
func (t *Task) CacheKey() string {
	key := t.ID
	namespace := strings.TrimSpace(t.CacheNamespace)
	scope := strings.TrimSpace(t.CacheScope)
	if namespace != "" {
		key += "," + namespace
	}
	if scope != "" {
		key += "," + scope
	}
	return key
}
