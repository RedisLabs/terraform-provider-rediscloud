package testhelpers

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
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

// PreCheckNoEssentialsSubscription fails fast if an Essentials subscription already exists,
// because Redis Cloud allows only one Essentials subscription per account.
func PreCheckNoEssentialsSubscription(t *testing.T) bool {
	t.Helper()
	if !envchecks.RedisCloudCheck(t) {
		return false
	}

	apiClient, err := client.GetTestClient()
	if err != nil {
		t.Errorf("Failed to create API client: %v", err)
		return false
	}

	subs, err := apiClient.Client.FixedSubscriptions.List(context.TODO())
	if err != nil {
		t.Errorf("Failed to list essentials subscriptions: %v", err)
		return false
	}

	if len(subs) > 0 {
		t.Errorf("Essentials subscription already exists (ID: %d). Redis Cloud allows only 1 essentials subscription per account. Please delete the existing subscription before running this test.", redis.IntValue(subs[0].ID))
		return false
	}
	return true
}

// CheckEssentialsSubscriptionDestroy verifies that every Essentials subscription recorded in the
// Terraform state has been removed from Redis Cloud.
func CheckEssentialsSubscriptionDestroy(s *terraform.State) error {
	apiClient, err := client.GetTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_essentials_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := apiClient.Client.FixedSubscriptions.List(context.TODO())
		if err != nil {
			return err
		}

		for _, sub := range subs {
			if redis.IntValue(sub.ID) == subId {
				return fmt.Errorf("fixed subscription %d still exists", subId)
			}
		}
	}

	return nil
}
