package agentmemory

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Task completion can precede readiness. Return the last observed store even on
// failure so the caller can preserve its identity and status without another GET.
func waitForAgentMemoryToBeReady(ctx context.Context, id string, c *client.ApiClient, timeout time.Duration) (*api.Store, error) {
	var mu sync.Mutex
	var lastStore *api.Store
	wait := &retry.StateChangeConf{
		Pending: []string{api.StatusProvisioning},
		Target:  []string{api.StatusReady},
		Timeout: timeout,
		// Check immediately: the API task has already completed.
		PollInterval: c.WaitPollInterval(5 * time.Second),
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Waiting for Agent Memory store %s to be ready", id)
			store, err := c.Client.AgentMemory.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}
			if store == nil || redis.StringValue(store.StoreID) != id {
				return nil, "", fmt.Errorf("agent memory readiness response did not match store %s", id)
			}
			// Cancellation can return before an in-flight refresh finishes.
			mu.Lock()
			lastStore = store
			mu.Unlock()
			return store, redis.StringValue(store.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	mu.Lock()
	defer mu.Unlock()
	return lastStore, err
}
