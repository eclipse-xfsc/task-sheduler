// nolint:revive
package design

import . "goa.design/goa/v3/dsl"

var CreateTaskRequest = Type("CreateTaskRequest", func() {
	Field(1, "taskName", String, "Task name.")
	Field(2, "data", Any, "Data contains JSON payload that will be used for task execution.")
	Field(3, "cacheNamespace", String, "Cache key namespace.")
	Field(4, "cacheScope", String, "Cache key scope.")
	Required("taskName", "data")
})

var CreateTaskResult = Type("CreateTaskResult", func() {
	Field(1, "taskID", String, "Unique task identifier.")
	Required("taskID")
})

var TaskResultRequest = Type("TaskResultRequest", func() {
	Field(1, "taskID", String, "Unique task identifier.")
	Required("taskID")
})

var CreateTaskListRequest = Type("CreateTaskListRequest", func() {
	Field(1, "taskListName", String, "TaskList name.")
	Field(2, "data", Any, "Data contains JSON payload that will be used for taskList execution.")
	Field(3, "cacheNamespace", String, "Cache key namespace.")
	Field(4, "cacheScope", String, "Cache key scope.")
	Required("taskListName", "data")
})

var CreateTaskListResult = Type("CreateTaskListResult", func() {
	Field(1, "taskListID", String, "Unique taskList identifier.")
	Required("taskListID")
})

var TaskListStatusRequest = Type("TaskListStatusRequest", func() {
	Field(1, "taskListID", String, "Unique taskList identifier.")
	Required("taskListID")
})

var TaskListStatusResponse = Type("TaskListStatusResponse", func() {
	Field(1, "id", String, "Unique taskList identifier.", func() {
		Example("9cc9f504-2b7f-4e24-ac59-653e9533840a")
	})
	Field(2, "status", String, "Current status of the taskList", func() {
		Example("done")
	})
	Field(3, "groups", ArrayOf(GroupStatus), "Array of GroupStatus")
	Required("id", "status")
})

var GroupStatus = Type("GroupStatus", func() {
	Field(1, "id", String, "Unique group identifier.", func() {
		Example("a7d1349d-34b5-4c65-b671-d1aa362fc446")
	})
	Field(2, "status", String, "Current status of the group", func() {
		Example("done")
	})
	Field(3, "tasks", ArrayOf(TaskStatus), "Array of TaskStatus")
})

var TaskStatus = Type("TaskStatus", func() {
	Field(1, "id", String, "Unique task identifier.", func() {
		Example("d16996cd-1977-42a9-90b2-b4548a35c1b4")
	})
	Field(2, "status", String, "Current status of the task", func() {
		Example("done")
	})
})

var HealthResponse = Type("HealthResponse", func() {
	Field(1, "service", String, "Service name.")
	Field(2, "status", String, "Status message.")
	Field(3, "version", String, "Service runtime version.")
	Required("service", "status", "version")
})
