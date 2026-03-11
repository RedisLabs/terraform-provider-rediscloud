package pro_test

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	sharedClient     *client.ApiClient
	sharedClientOnce sync.Once
	sharedClientErr  error
)

func getTestClient() (*client.ApiClient, error) {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = client.NewClient()
	})
	return sharedClient, sharedClientErr
}

// checkProSubscriptionDestroy verifies that all rediscloud_subscription resources
// have been destroyed after a test completes.
func checkProSubscriptionDestroy(s *terraform.State) error {
	apiClient, err := getTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_subscription" {
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
