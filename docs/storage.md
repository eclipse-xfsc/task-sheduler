# Task service - Storage

### Storage Interface

The Task Storage is an interface and can be reviewed [here](../internal/service/storage.go). 

### Storage implementation

Current [implementation](../internal/storage/storage.go) uses MongoDB database.
Adding other implementations is easy - just implement the Storage interface.

### Task Storage

In current implementation there are three Mongo collections with different purpose.

1. **taskTemplates**

    The collection contains predefined task definitions in JSON format. Here are defined
what tasks can be created and executed by the service.

2. **tasks**

    The collection contains newly created tasks *pending* for execution. It acts like a 
FIFO queue and is used by the task executor to retrieve tasks for workers to execute.

3. **tasksHistory**

    The collection contains successfully completed tasks for results querying,
audit, reporting and debugging purposes.

### Task List Storage

In current implementation there are four Mongo collections with different purpose.

1. **taskListTemplates**

    The collection contains predefined task list definitions in JSON format. Each definition contains
groups of tasks which must be instantiated and later executed as part of the task list.

2. **taskLists**

    The collection contains newly created task lists *pending* for execution. It acts like a
FIFO queue and is used by the task list executor to retrieve task lists for workers to execute.

3. **tasks**

    The collection contains the tasks belonging to a group which is part of a task list. When a task list
is fetched for execution, all tasks are fetched and executed for that particular task list.

4. **tasksListHistory**

    The collection contains completed task lists for results querying,
audit, reporting and debugging purposes.

### Event Task definition Storage

In current implementation there is one Mongo collection for storing Event Task definitions.

1. **eventTasks**

    The collection contains predefined Event Task definitions in JSON format. Each definition 
contains event metadata fields and a valid Task name. See: [cache event tasks](cache-event-task.md)
