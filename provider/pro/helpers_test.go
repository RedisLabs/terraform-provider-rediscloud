package pro_test

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// checkProSubscriptionDestroy verifies that all rediscloud_subscription resources
// have been destroyed after a test completes.
func checkProSubscriptionDestroy(s *terraform.State) error {
	return checkSubscriptionDestroy(s, "rediscloud_subscription")
}

func checkAASubscriptionDestroy(s *terraform.State) error {
	return checkSubscriptionDestroy(s, "rediscloud_active_active_subscription")
}

func checkSubscriptionDestroy(s *terraform.State, resourceType string) error {
	apiClient, err := client.GetTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != resourceType {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := apiClient.Client.Subscription.List(context.TODO())
		if err != nil {
			return err
		}

		for _, sub := range subs {
			if redis.IntValue(sub.ID) == subId {
				return fmt.Errorf("subscription %d still exists", subId)
			}
		}
	}

	return nil
}
