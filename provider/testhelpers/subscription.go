package testhelpers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// CheckProSubscriptionDestroy verifies that every Pro subscription recorded in the
// Terraform state has been removed from Redis Cloud.
func CheckProSubscriptionDestroy(state *terraform.State) error {
	return checkSubscriptionDestroy(state, "rediscloud_subscription")
}

// CheckActiveActiveSubscriptionDestroy verifies that every Active-Active subscription
// recorded in the Terraform state has been removed from Redis Cloud.
func CheckActiveActiveSubscriptionDestroy(state *terraform.State) error {
	return checkSubscriptionDestroy(state, "rediscloud_active_active_subscription")
}

func checkSubscriptionDestroy(state *terraform.State, resourceType string) error {
	subscriptionIDs := make(map[int]struct{})
	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != resourceType {
			continue
		}

		subscriptionID, err := strconv.Atoi(resourceState.Primary.ID)
		if err != nil {
			return err
		}
		subscriptionIDs[subscriptionID] = struct{}{}
	}

	if len(subscriptionIDs) == 0 {
		return nil
	}

	apiClient, err := client.GetTestClient()
	if err != nil {
		return err
	}

	subscriptions, err := apiClient.Client.Subscription.List(context.TODO())
	if err != nil {
		return err
	}

	for _, subscription := range subscriptions {
		subscriptionID := redis.IntValue(subscription.ID)
		if _, exists := subscriptionIDs[subscriptionID]; exists {
			return fmt.Errorf("subscription %d still exists", subscriptionID)
		}
	}

	return nil
}
