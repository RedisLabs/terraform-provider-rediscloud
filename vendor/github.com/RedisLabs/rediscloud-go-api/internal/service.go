package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/avast/retry-go"
)

type Log interface {
	Println(v ...interface{})
}

type api struct {
	client *HttpClient
	logger Log
}

func NewAPI(client *HttpClient, logger Log) *api {
	return &api{client: client, logger: logger}
}

// WaitForResourceId will poll the task, waiting for the task to finish processing, where it will then return.
// An error will be returned if the task couldn't be retrieved or the task was not processed successfully.
//
// The task will be continuously polled until the task either fails or succeeds - cancellation can be achieved
// by cancelling the context.
func (a *api) WaitForResourceId(ctx context.Context, id string) (int, error) {
	task, err := a.waitForTaskToComplete(ctx, id)
	if err != nil {
		return 0, err
	}

	return redis.IntValue(task.Response.ID), nil
}

// Wait will poll the task, waiting for the task to finish processing, where it will then return.
// An error will be returned if the task couldn't be retrieved or the task was not processed successfully.
//
// The task will be continuously polled until the task either fails or succeeds - cancellation can be achieved
// by cancelling the context.
func (a *api) Wait(ctx context.Context, id string) error {
	_, err := a.waitForTaskToComplete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// WaitForResource will poll the task, waiting for the task to finish processing, where it will then marshal the
// returned resource into the value pointed to be `resource`.
//
// The task will be continuously polled until the task either fails or succeeds - cancellation can be achieved
// by cancelling the context.
func (a *api) WaitForResource(ctx context.Context, id string, resource interface{}) error {
	task, err := a.waitForTaskToComplete(ctx, id)
	if err != nil {
		return err
	}

	err = json.Unmarshal(*task.Response.Resource, resource)
	if err != nil {
		return err
	}

	return nil
}

func (a *api) waitForTaskToComplete(ctx context.Context, id string) (*task, error) {
	var task *task
	err := retry.Do(func() error {
		var err error
		task, err = a.get(ctx, id)
		if err != nil {
			return retry.Unrecoverable(err)
		}

		status := redis.StringValue(task.Status)
		if status == processedState {
			return nil
		}

		if _, ok := processingStates[status]; !ok {
			return retry.Unrecoverable(fmt.Errorf("task %s failed %s - %s", id, status, redis.StringValue(task.Description)))
		}

		return fmt.Errorf("task %s not processed yet: %s", id, status)
	},
		retry.Attempts(math.MaxUint64), retry.Delay(1*time.Second), retry.MaxDelay(30*time.Second),
		retry.LastErrorOnly(true), retry.Context(ctx), retry.OnRetry(func(_ uint, err error) {
			a.logger.Println(err)
		}))
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (a *api) get(ctx context.Context, id string) (*task, error) {
	var task task
	if err := a.client.Get(ctx, fmt.Sprintf("retrieve task %s", id), "/tasks/"+url.PathEscape(id), &task); err != nil {
		return nil, err
	}

	if task.Response != nil && task.Response.Error != nil {
		return nil, task.Response.Error
	}

	return &task, nil
}

var processingStates = map[string]bool{
	"initialized":            true,
	"received":               true,
	"processing-in-progress": true,
}

const processedState = "processing-completed"
