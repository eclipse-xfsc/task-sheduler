# Task Service - Cache Event Task Documentation

### Subscribe for Cache events

Current implementation uses NATS as a messaging system for events.
There are two environment variables that need to be set for subscribing for cache events.

```shell
NATS_ADDR="example.com:4222"
NATS_SUBJECT="subject"
```

### Event Task definition

In order to create a Task upon receiving a Cache event an `event task template` **must**
be available. Event task JSON templates are stored in Storage. Currently, a Mongo database collection
named `eventTask` is used for storing event task templates. Below is an example of event task template definition:

```json
{
  "key": "did:web:did.actor:alice",
  "namespace": "Login",
  "scope": "Administration",
  "taskName": "exampleTask"
}
```

The `taskName` field **must** be a valid `task definition` name. See: [Tasks](task.md)

### Create Task for Cache event

Every Cache event contains the `key`, `namespace`, and `scope` for a created/updated entry in cache.
The task service gets an `event task template` from storage, if available, and adds a Task in task queue
passing the metadata from the event. The added Task **must** execute a policy (rather than call an external URL).
The event metadata can be accessed inside the executed policy by key.
Example:
```
key := input.key
namespace := input.namespace
scope := input.scope
```
