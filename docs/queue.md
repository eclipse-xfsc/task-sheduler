# Task service - Queue

### Queue Interface

The Task Queue is an interface and can be reviewed [here](../internal/service/queue.go).
Current [implementation](../internal/storage/storage.go) uses persistent database.

### Why the current implementation of the Queue is Database

Why we decided to use a database as queue instead of a universal message queue product
like Kafka, so that the executor won't need to poll for new tasks, but will instead
receive them as they come?

1. The TSA requirements document describes a functionality for task groups.
   These are groups of tasks which may be executed sequentially with later tasks in the
   group depending on the results of the previous tasks in the group. This means that we
   can't just execute a task when an event for its creation has arrived. We have to keep
   persistent execution state through multiple task executions and multiple service instances
   (e.g. pods/executors) for which a database seems like more natural choice.

2. Tasks are not just simple messages, but contain state. The state may change during
   the lifecycle and execution progress of the task. The state must also be persistent,
   auditable and should be available for querying by clients. A database seems more suitable
   to us for implementing these features than a simple delivery message queue.

The downside of our current approach is that the database is constantly polled by the
executor for new tasks. In practice this should not be a problem, because the task collection
containing *pending* tasks for execution should contain very small number of records.
Even at peak load, it should contain a few hundred tasks at most, because after tasks
are executed they are removed from the queue collection. We expect that the queue collection
will be empty most of the time and even when it isn't, there won't be many records inside.
Practically speaking, the query to retrieve tasks for execution should be very fast and
light in terms of database load.

The benefits of this approach are that it's simple to implement and reason about.

> If you have better ideas, or these arguments sound strange, please get in touch with us
> and we'll consider other options and improvements to the current model.
