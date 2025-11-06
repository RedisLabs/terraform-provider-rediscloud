package utils

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
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

func WaitForDatabaseToBeActive(ctx context.Context, subId, id int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			databases.StatusDraft,
			databases.StatusPending,
			databases.StatusActiveChangePending,
			databases.StatusRCPActiveChangeDraft,
			databases.StatusActiveChangeDraft,
			databases.StatusRCPDraft,
			databases.StatusRCPChangePending,
			databases.StatusProxyPolicyChangePending,
			databases.StatusProxyPolicyChangeDraft,
			databases.StatusDynamicEndpointsCreationPending,
			databases.StatusActiveUpgradePending,
			"bdb-update-pending", // Database update in progress.
			// TODO replace with api model string in next release
		},
		Target:       []string{databases.StatusActive},
		Timeout:      SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for database %d to be active", id)

			database, err := api.Client.Database.Get(ctx, subId, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(database.Status), redis.StringValue(database.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

// WaitForActiveActiveTransitGatewayResourceToBeAvailable waits for Active-Active Transit Gateway API resources
// to become available. This handles the case where Response.Resource is nil during initial subscription provisioning.
func WaitForActiveActiveTransitGatewayResourceToBeAvailable(ctx context.Context, subId int, regionId int, api *client.ApiClient) (*attachments.GetAttachmentsTask, error) {
	wait := &retry.StateChangeConf{
		Pending:      []string{"provisioning"},
		Target:       []string{"available"},
		Timeout:      TransitGatewayProvisioningTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for Active-Active Transit Gateway resource to be available for subscription %d, region %d", subId, regionId)

			tgwTask, err := api.Client.TransitGatewayAttachments.GetActiveActive(ctx, subId, regionId)
			if err != nil {
				return nil, "", err
			}

			// Check for nil response structure during provisioning
			if tgwTask == nil || tgwTask.Response == nil || tgwTask.Response.Resource == nil {
				return nil, "provisioning", nil
			}

			return tgwTask, "available", nil
		},
	}

	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("timeout waiting for Active-Active Transit Gateway resource to become available for subscription %d, region %d. "+
			"This may indicate the subscription is still provisioning or there's an issue with the subscription setup. "+
			"Original error: %w", subId, regionId, err)
	}

	return result.(*attachments.GetAttachmentsTask), nil
}
