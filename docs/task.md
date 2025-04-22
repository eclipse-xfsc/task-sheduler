# Task Service - Task Documentation

### Task Definition

A tasks is described in JSON format and base task (template) definition *must* be available
before a task can be created. Task JSON templates are stored in Mongo database in a collection
named `taskTemplates`. Some task properties are statically predefined in the template,
others are populated dynamically by the input request and the output response. Below is
an example of task template definition:

```json
{
    "name":"exampleTask",
    "url":"https://jsonplaceholder.typicode.com/todos/1",
    "method":"GET",
    "requestPolicy":"policies/example/example/1.0",
    "responsePolicy":"",
    "finalPolicy":"",
    "cacheNamespace":"login",
    "cacheScope":"user"
}
```

Tasks are created by using their `name` attribute. If a task template with the given
`name` does not exist, a task will not be created and an error is returned.

### Task Execution

Below is an example of creating a task with the template definition given above:
```shell
curl -v -X POST http://localhost:8082/v1/task/exampleTask -d '{"exampleInput":{"test":123}}'
```

The HTTP request will create a task for asynchronous execution and the JSON object
given as input will be used as the body of the task request. The caller will receive
immediately the `taskID` as response, and the result of the asynchronous task
execution will be stored in the TSA Cache after the task is completed.

The actual _Task execution_ is strictly bound to the _Task definition_. In order a _task_
to be executed successfully, its _definition_ **must** contain either a `requestPolicy` OR
`url` and `method`. When a `requestPolicy` is set in the _Task definition_, the task will
evaluate it and will ignore the `url` and the `method`. If a `requestPolicy` is missing in
the _Task definition_, the task will execute an HTTP request to the given `url` with the
given `method`. If both `requestPolicy` AND `url` and `method` are missing in the _Task definition_,
the task cannot be executed. Reference table:

_Task definition_ contains: | `requestPolicy` only | `url` and `method` only | Both `requestPolicy` AND `url` and `method` | Neither
--- | --- | --- | --- |---
**_Task_ will execute** | `requestPolicy` | `url` and `method` | `requestPolicy` | None

### Task Executor Configuration

There are three environment variables that control the behavior of the task executor.

```shell
EXECUTOR_WORKERS="5"
EXECUTOR_POLL_INTERVAL="1s"
EXECUTOR_MAX_TASK_RETRIES="10"
```

Poll interval specifies how often the executor will try to fetch a *pending* task
from the queue. After a task is fetched, it is given to one of the workers for execution.
With the given example of 1 second (1s), the executor will retrieve 1 task per second at most.
If there are multiple instances (pods) of the service, multiply by their number
(e.g. 5 pods = 5 tasks per second).

If this is not enough, the poll interval can be decreased, or we can slightly modify
the polling function to fetch many tasks at once (and also increase the number of workers).

Maximum task retries specifies how many failed attempts to execute a single task are going
to be made by workers before the task is removed from the queue. In the example above workers are going to
execute a task 10 times and fail before the task is removed.

To learn more about the queue and why current implementation uses database as queue see [queue](queue.md).
