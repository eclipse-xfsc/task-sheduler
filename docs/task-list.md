# Task service - Task list

### Task list definition

A task list is described in JSON format and base task list (template) definition *must* be available
before a task list can be created. Task list JSON templates are stored in Mongo database in a collection
named `taskListTemplates`. Below is an example of task list template definition:

```json
{
  "name": "example",
  "cacheNamespace": "login",
  "cacheScope": "user",
  "groups": [
    {
      "execution": "sequential",
      "tasks": [
        "taskName1",
        "taskName2"
      ]
    },
    {
      "execution": "parallel",
      "tasks": [
        "taskName3",
        "taskName4",
        "taskName5"
      ]
    }
  ]
}
```

Task lists are created by using their `name` attribute. Tasks for each group in the `groups` field
are also created by using their `name` so task names should match existing task template
definitions (see [tasks](task.md)). If a task list template with the given `name` does not exist OR a task
template definition does not exist, a task list will not be created and an error is returned.

### Task list Execution

Below is an example of creating a task list with the template definition given above:
```shell
curl -v -X POST http://localhost:8082/v1/taskList/example -d '{"input": {"key": "value"}}'
```

The HTTP request will create a task list and corresponding tasks for each of the groups within the task list
for asynchronous execution and the JSON object given as input will be used as the body of the task list request.
The caller will receive immediately the `taskListID` as response.

The executor then takes a task list from the queue and executes the groups within the task list sequentially.
Each group has a field called `execution` which can be one of two options: `parallel` or `sequential`. This
field describes how the tasks within the group *must* be executed.
 - Sequential group execution: tasks within the group are executed sequentially and the result of each task is passed as
an input to the next. If one task fails to execute, all following tasks are marked as failed and the whole group fails.
 - Parallel group execution: tasks within the group are executed in parallel and the results are dependant. If a task
fails to execute, this does not affect the other tasks but the group is marked with failed status.

### Task list State

The state of the task list asynchronous execution is available later on the `result` endpoint:
```shell
curl -v -X GET http://localhost:8082/v1/taskListStatus/{taskListID}
```
The state is returned as an HTTP status code.
- Status code `200` for `Done` state;
- Status code `201` for `Created` state;
- Status code `202` for `Pending` state (the task list is being executed);
- Status code `207` for `Failed` state (at least one task within the task list has failed).

Example responses:

HTTP Response code `200`: Done
```json
{
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
}

```

HTTP Response code `207`: Failed
```json
{
  "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
  "status": "failed",
  "groups": [
    {
      "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
      "type": "sequential",
      "status": "failed",
      "tasks": [
        {
          "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
          "status": "done"
        },
        {
          "id": "ad641603-1ca0-4342-ad73-d70a6b1ec502",
          "status": "failed"
        }
      ]
    }
  ]
}

```

### Task list Executor Configuration

There are two environment variables that control the level of concurrency
of the task list executor.

```shell
LIST_EXECUTOR_WORKERS="5"
LIST_EXECUTOR_POLL_INTERVAL="1s"
```

Poll interval specifies how often the executor will try to fetch a *pending* task list
from the queue. After a task list is fetched, it is executed in parallel with other executions.
Number of workers is the limit of permitted parallel task lists executions.
With the given example of 1 second (1s), the executor will retrieve 1 task list per second at most.
If there are multiple instances (pods) of the service, multiply by their number
(e.g. 5 pods = 5 tasks per second).

If this is not enough, the poll interval can be decreased, or we can slightly modify
the polling function to fetch many task lists at once (and also increase the number of workers).

To learn more about the queue and why we use database as queue see [queue](queue.md).
