package event

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
)

const eventDataKey = "key"

type Client struct {
	storage  service.Storage
	queue    service.Queue
	consumer *nats.Consumer
	events   cloudevents.Client
}

func New(s service.Storage, q service.Queue, addr, subject string) (*Client, error) {
	// create cloudevents NATS consumer
	// other protocol implementations: https://github.com/cloudevents/sdk-go/tree/main/protocol
	c, err := nats.NewConsumer(addr, subject, nats.NatsOptions())
	if err != nil {
		return nil, err
	}

	e, err := cloudevents.NewClient(c)
	if err != nil {
		return nil, err
	}

	return &Client{
		storage:  s,
		queue:    q,
		consumer: c,
		events:   e,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	return c.events.StartReceiver(ctx, c.handler)
}

func (c *Client) Close(ctx context.Context) error {
	return c.consumer.Close(ctx)
}

// handler is registered as a callback function when the client is started.
// It creates a task for execution when an event is received from the cache.
func (c *Client) handler(ctx context.Context, event cloudevents.Event) error {
	if event.DataContentType() != "application/json" {
		return errors.New("event data has invalid content type, must be application/json")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(event.Data(), &data); err != nil {
		return err
	}

	cKey, ok := data[eventDataKey]
	if !ok {
		return errors.New("invalid event data key")
	}
	cacheKey, _ := cKey.(string)

	sCacheKey := strings.Split(cacheKey, ",")
	if len(sCacheKey) == 0 {
		return errors.New("cache key cannot be empty")
	}

	key := sCacheKey[0]

	var namespace, scope string
	if len(sCacheKey) > 1 {
		namespace = sCacheKey[1]
	}
	if len(sCacheKey) > 2 {
		scope = sCacheKey[2]
	}

	// get event task template from storage
	eventTask, err := c.storage.EventTask(ctx, key, namespace, scope)
	if err != nil {
		return err
	}

	// add task to task queue
	if err := c.enqueueTask(ctx, eventTask); err != nil {
		return err
	}

	return nil
}

func (c *Client) enqueueTask(ctx context.Context, eventTask *service.EventTask) error {
	// get predefined task definition from storage
	task, err := c.storage.TaskTemplate(ctx, eventTask.TaskName)
	if err != nil {
		return err
	}

	if task.RequestPolicy == "" {
		return errors.New("event task must execute a policy")
	}

	input, err := json.Marshal(eventTask)
	if err != nil {
		return errors.New("error marshaling input to JSON", err)
	}

	task.ID = uuid.NewString()
	task.State = service.Created
	task.CreatedAt = time.Now()
	task.Request = input

	if err := c.queue.Add(ctx, task); err != nil {
		return errors.New("failed to create task", err)
	}

	return nil
}
