// nolint:revive
package design

import . "goa.design/goa/v3/dsl"

var _ = API("task", func() {
	Title("Task Service")
	Description("The task service is executing tasks created from policies.")
	Server("task", func() {
		Description("Task Server")
		Host("development", func() {
			Description("Local development server")
			URI("http://localhost:8082")
		})
	})
})

var _ = Service("task", func() {
	Description("Task service provides endpoints to work with tasks.")

	Method("Create", func() {
		Description("Create a task and put it in a queue for execution.")
		Payload(CreateTaskRequest)
		Result(CreateTaskResult)
		HTTP(func() {
			POST("/v1/task/{taskName}")

			Header("cacheNamespace:x-cache-namespace", String, "Cache key namespace", func() {
				Example("login")
			})
			Header("cacheScope:x-cache-scope", String, "Cache key scope", func() {
				Example("user")
			})

			Body("data")

			Response(StatusOK)
		})
	})

	Method("TaskResult", func() {
		Description("TaskResult retrieves task result from the Cache service.")
		Payload(TaskResultRequest)
		Result(Any)
		HTTP(func() {
			GET("/v1/taskResult/{taskID}")
			Response(StatusOK)
		})
	})
})

var _ = Service("taskList", func() {
	Description("TaskList service provides endpoints to work with task lists.")

	Method("Create", func() {
		Description("Create a task list and corresponding tasks and put them in respective queues for execution.")
		Payload(CreateTaskListRequest)
		Result(CreateTaskListResult)
		HTTP(func() {
			POST("/v1/taskList/{taskListName}")

			Header("cacheNamespace:x-cache-namespace", String, "Cache key namespace", func() {
				Example("login")
			})
			Header("cacheScope:x-cache-scope", String, "Cache key scope", func() {
				Example("user")
			})

			Body("data")

			Response(StatusOK)
		})
	})

	Method("TaskListStatus", func() {
		Description("TaskListStatus retrieves a taskList status containing all tasks' unique IDs and statuses from the Cache service.")
		Payload(TaskListStatusRequest)
		Result(TaskListStatusResponse)
		HTTP(func() {
			GET("/v1/taskListStatus/{taskListID}")
			Response(StatusOK)
			Response(StatusCreated, func() {
				Tag("status", "created")
			})
			Response(StatusAccepted, func() {
				Tag("status", "pending")
			})
			Response(StatusMultiStatus, func() {
				Tag("status", "failed")
			})
		})
	})
})

var _ = Service("health", func() {
	Description("Health service provides health check endpoints.")

	Method("Liveness", func() {
		Payload(Empty)
		Result(HealthResponse)
		HTTP(func() {
			GET("/liveness")
			Response(StatusOK)
		})
	})

	Method("Readiness", func() {
		Payload(Empty)
		Result(HealthResponse)
		HTTP(func() {
			GET("/readiness")
			Response(StatusOK)
		})
	})
})

var _ = Service("openapi", func() {
	Description("The openapi service serves the OpenAPI(v3) definition.")
	Meta("swagger:generate", "false")
	HTTP(func() {
		Path("/swagger-ui")
	})
	Files("/openapi.json", "./gen/http/openapi3.json", func() {
		Description("JSON document containing the OpenAPI(v3) service definition")
	})
	Files("/{*filepath}", "./swagger/")
})
