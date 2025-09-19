package utils

import (
	"context"
	"log"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

func WaitForSubscriptionToBeActive(ctx context.Context, id int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusPending},
		Target:       []string{subscriptions.SubscriptionStatusActive},
		Timeout:      SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be %s", id, subscriptions.SubscriptionStatusActive)

			subscription, err := api.Client.Subscription.Get(ctx, id)
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
