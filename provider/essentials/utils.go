package essentials

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	fixedSubscriptions "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"log"
	"time"
)

func filterFixedDatabases(list *fixedDatabases.ListFixedDatabase, filters []func(db *fixedDatabases.FixedDatabase) bool) ([]*fixedDatabases.FixedDatabase, error) {
	var filtered []*fixedDatabases.FixedDatabase
	for list.Next() {
		if filterFixedDatabase(list.Value(), filters) {
			filtered = append(filtered, list.Value())
		}
	}
	if list.Err() != nil {
		return nil, list.Err()
	}

	return filtered, nil
}

func filterFixedDatabase(db *fixedDatabases.FixedDatabase, filters []func(db *fixedDatabases.FixedDatabase) bool) bool {
	for _, filter := range filters {
		if !filter(db) {
			return false
		}
	}
	return true
}

func waitForEssentialsSubscriptionToBeActive(ctx context.Context, id int, api *utils.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusPending},
		Target:  []string{subscriptions.SubscriptionStatusActive},
		Timeout: utils.SafetyTimeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed subscription %d to be active", id)

			subscription, err := api.Client.FixedSubscriptions.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForEssentialsSubscriptionToBeDeleted(ctx context.Context, id int, api *utils.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusDeleting},
		Target:  []string{"deleted"},
		Timeout: utils.SafetyTimeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed subscription %d to be deleted", id)

			subscription, err := api.Client.FixedSubscriptions.Get(ctx, id)
			if err != nil {
				if _, ok := err.(*fixedSubscriptions.NotFound); ok {
					return "deleted", "deleted", nil
				}
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
