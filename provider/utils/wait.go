package utils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func WaitForSubscriptionToBeDeleted(ctx context.Context, id int, api *client.ApiClient, timeout time.Duration) error {
	_, err := WaitForSubscriptionToBeStatus(ctx, id, api, timeout, []string{subscriptions.SubscriptionStatusDeleted}, subscriptions.SubscriptionStatusDeleting)
	return err
}

func WaitForSubscriptionToBeActive(ctx context.Context, id int, api *client.ApiClient, timeout time.Duration) error {
	_, err := WaitForSubscriptionToBeStatus(ctx, id, api, timeout, []string{subscriptions.SubscriptionStatusActive})
	return err
}

func WaitForSubscriptionToBeStatus(ctx context.Context, id int, api *client.ApiClient, timeout time.Duration, targetStatuses []string, pendingStatuses ...string) (string, error) {
	if len(pendingStatuses) == 0 {
		pendingStatuses = []string{subscriptions.SubscriptionStatusPending}
	}

	wait := &retry.StateChangeConf{
		Pending:      pendingStatuses,
		Target:       targetStatuses,
		Timeout:      timeout,
		Delay:        api.WaitDelay(10 * time.Second),
		PollInterval: api.WaitPollInterval(30 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to reach one of %v", id, targetStatuses)

			subscription, err := api.Client.Subscription.Get(ctx, id)
			if err != nil {
				// Once explicit or automatic deletion removes the subscription, polling returns
				// NotFound. Treat that as success only when deleted is an accepted target;
				// waiters for other statuses must still return the API error.
				var notFound *subscriptions.NotFound
				if slices.Contains(targetStatuses, subscriptions.SubscriptionStatusDeleted) && errors.As(err, &notFound) {
					return subscriptions.SubscriptionStatusDeleted, subscriptions.SubscriptionStatusDeleted, nil
				}
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// WaitForSubscriptionPublicEndpointAccess waits for the subscription's public_endpoint_access
// property to match the expected value. This handles API eventual consistency after updates.
func WaitForSubscriptionPublicEndpointAccess(ctx context.Context, id int, api *client.ApiClient, expected bool, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{"waiting"},
		Target:       []string{"matched"},
		Timeout:      timeout,
		Delay:        api.WaitDelay(2 * time.Second),
		PollInterval: api.WaitPollInterval(5 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d public_endpoint_access to be %t", id, expected)

			subscription, err := api.Client.Subscription.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			// Default to true if nil (API behaviour)
			actual := subscription.PublicEndpointAccess == nil || *subscription.PublicEndpointAccess
			if actual == expected {
				return subscription, "matched", nil
			}
			return subscription, "waiting", nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func WaitForDatabaseToBeActive(ctx context.Context, subId, id int, api *client.ApiClient, timeout time.Duration) error {
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
		Timeout:      timeout,
		Delay:        api.WaitDelay(10 * time.Second),
		PollInterval: api.WaitPollInterval(30 * time.Second),

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

const (
	waitStateWaiting = "waiting"
	waitStateFound   = "found"
)

// TransitGatewayAttachmentMatcher decides whether an attachment satisfies the caller's wait condition. Return false
// when the attachment does not satisfy the condition. Return an error when observing the attachment means the wait
// cannot succeed and polling needs to stop.
type TransitGatewayAttachmentMatcher func(tgw *attachments.TransitGatewayAttachment) (bool, error)

// WaitForTransitGatewayAttachmentsToMatch waits until the selected Transit Gateway API response contains one or more
// attachments accepted by matcher. The region ID is only used for Active-Active subscriptions.
func WaitForTransitGatewayAttachmentsToMatch(
	ctx context.Context,
	subId int,
	api *client.ApiClient,
	matcher TransitGatewayAttachmentMatcher,
	timeout time.Duration,
	subscriptionType string,
	regionId int,
) ([]*attachments.TransitGatewayAttachment, error) {
	gatewayName := "Transit Gateway"
	location := fmt.Sprintf("subscription %d", subId)
	if subscriptionType == subscriptions.SubscriptionDeploymentTypeActiveActive {
		gatewayName = "Active-Active Transit Gateway"
		location = fmt.Sprintf("subscription %d, region %d", subId, regionId)
	}

	wait := &retry.StateChangeConf{
		Pending:      []string{waitStateWaiting},
		Target:       []string{waitStateFound},
		Timeout:      timeout,
		Delay:        api.WaitDelay(5 * time.Second),
		PollInterval: api.WaitPollInterval(10 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for matching %s attachments for %s", gatewayName, location)

			var tgwTask *attachments.GetAttachmentsTask
			var requestErr error
			// Call the correct API endpoint, based on the TGW type
			if subscriptionType == subscriptions.SubscriptionDeploymentTypeActiveActive {
				// Active-Active subscriptions use the regional API endpoint.
				tgwTask, requestErr = api.Client.TransitGatewayAttachments.GetActiveActive(ctx, subId, regionId)
			} else {
				// Single-region subscriptions use the subscription endpoint.
				tgwTask, requestErr = api.Client.TransitGatewayAttachments.Get(ctx, subId)
			}
			if requestErr != nil {
				return nil, "", requestErr
			}

			if tgwTask == nil || tgwTask.Response == nil || tgwTask.Response.Resource == nil {
				return struct{}{}, waitStateWaiting, nil
			}

			matchingTgws, matchErr := matchTransitGatewayAttachments(tgwTask, matcher)
			if matchErr != nil {
				return nil, "", matchErr
			}
			if len(matchingTgws) == 0 {
				return struct{}{}, waitStateWaiting, nil
			}

			return matchingTgws, waitStateFound, nil
		},
	}

	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for %s attachments for %s: %w", gatewayName, location, err)
	}

	matchingTgws, ok := result.([]*attachments.TransitGatewayAttachment)
	if !ok {
		return nil, fmt.Errorf("internal error: unexpected result type from wait operation for %s", location)
	}

	return matchingTgws, nil
}

func matchTransitGatewayAttachments(
	tgwTask *attachments.GetAttachmentsTask,
	matcher TransitGatewayAttachmentMatcher,
) ([]*attachments.TransitGatewayAttachment, error) {
	var matchingTgws []*attachments.TransitGatewayAttachment
	for _, tgw := range tgwTask.Response.Resource.TransitGatewayAttachment {
		matches, err := matcher(tgw)
		if err != nil {
			return nil, err
		}
		if matches {
			matchingTgws = append(matchingTgws, tgw)
		}
	}

	return matchingTgws, nil
}

func WaitForSubscriptionToBeEncryptionKeyPending(ctx context.Context, id int, api *client.ApiClient, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusPending},
		Target:       []string{subscriptions.SubscriptionStatusEncryptionKeyPending, subscriptions.SubscriptionStatusActive},
		Timeout:      timeout,
		Delay:        api.WaitDelay(10 * time.Second),
		PollInterval: api.WaitPollInterval(30 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be %s", id, subscriptions.SubscriptionStatusEncryptionKeyPending)

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
