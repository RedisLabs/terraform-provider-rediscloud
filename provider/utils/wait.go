package utils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"sync/atomic"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
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

// WaitForActiveActiveTransitGatewayResourceToBeAvailable waits for Active-Active Transit Gateway API resources
// to become available. This handles the case where Response.Resource is nil during initial subscription provisioning.
func WaitForActiveActiveTransitGatewayResourceToBeAvailable(ctx context.Context, subId int, regionId int, api *client.ApiClient, timeout time.Duration) (*attachments.GetAttachmentsTask, error) {
	wait := &retry.StateChangeConf{
		Pending:      []string{"provisioning"},
		Target:       []string{"available"},
		Timeout:      timeout,
		Delay:        api.WaitDelay(10 * time.Second),
		PollInterval: api.WaitPollInterval(30 * time.Second),

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

	task, ok := result.(*attachments.GetAttachmentsTask)
	if !ok {
		return nil, fmt.Errorf("internal error: unexpected result type from wait operation for subscription %d, region %d", subId, regionId)
	}

	return task, nil
}

// TransitGatewayAttachmentStatusAvailable is the status indicating an attachment is ready for use.
const TransitGatewayAttachmentStatusAvailable = "available"

// transitGatewayAttachmentPendingStates are states that indicate the attachment is still being
// provisioned or modified. The waiter will continue polling while in these states.
var transitGatewayAttachmentPendingStates = []string{
	"initiating-request",
	"pending",
	"pendingAcceptance",
	"modifying",
	"deleting",
	"rejecting",
	"failing",
	"rollingBack",
}

var transitGatewayAttachmentTerminalStates = map[string]bool{
	"failed":   true,
	"rejected": true,
	"deleted":  true,
}

// WaitForTransitGatewayAttachmentToBeAvailable waits for a Pro Transit Gateway attachment to reach
// the "available" status. This handles the delay between creating an attachment and AWS completing
// the VPC attachment process.
func WaitForTransitGatewayAttachmentToBeAvailable(
	ctx context.Context,
	subId int,
	tgwId int,
	api *client.ApiClient,
	timeout time.Duration,
) (*attachments.TransitGatewayAttachment, error) {
	wait := &retry.StateChangeConf{
		Pending:      transitGatewayAttachmentPendingStates,
		Target:       []string{TransitGatewayAttachmentStatusAvailable},
		Timeout:      timeout,
		Delay:        api.WaitDelay(10 * time.Second),
		PollInterval: api.WaitPollInterval(30 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for Transit Gateway attachment to be available for subscription %d, tgw %d", subId, tgwId)

			tgwTask, err := api.Client.TransitGatewayAttachments.Get(ctx, subId)
			if err != nil {
				return nil, "", err
			}

			// Check for nil response structure
			if tgwTask == nil || tgwTask.Response == nil || tgwTask.Response.Resource == nil {
				return nil, "pending", nil
			}

			// Find the specific attachment by tgwId
			for _, tgw := range tgwTask.Response.Resource.TransitGatewayAttachment {
				if redis.IntValue(tgw.Id) == tgwId {
					status := redis.StringValue(tgw.Status)
					log.Printf("[DEBUG] Transit Gateway attachment status: %s", status)

					if transitGatewayAttachmentTerminalStates[status] {
						return nil, status, fmt.Errorf("transit gateway attachment reached terminal state: %s", status)
					}

					return tgw, status, nil
				}
			}

			// Attachment not found yet
			return nil, "pending", nil
		},
	}

	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("timeout waiting for Transit Gateway attachment to become available for subscription %d, tgw %d. "+
			"Original error: %w", subId, tgwId, err)
	}

	tgw, ok := result.(*attachments.TransitGatewayAttachment)
	if !ok {
		return nil, fmt.Errorf("internal error: unexpected result type from wait operation for subscription %d, tgw %d", subId, tgwId)
	}

	return tgw, nil
}

// WaitForActiveActiveTransitGatewayAttachmentToBeAvailable waits for an Active-Active Transit Gateway
// attachment to reach the "available" status. This handles the delay between creating an attachment
// and AWS completing the VPC attachment process.
func WaitForActiveActiveTransitGatewayAttachmentToBeAvailable(
	ctx context.Context,
	subId int,
	regionId int,
	tgwId int,
	api *client.ApiClient,
	timeout time.Duration,
) (*attachments.TransitGatewayAttachment, error) {
	wait := &retry.StateChangeConf{
		Pending:      transitGatewayAttachmentPendingStates,
		Target:       []string{TransitGatewayAttachmentStatusAvailable},
		Timeout:      timeout,
		Delay:        api.WaitDelay(10 * time.Second),
		PollInterval: api.WaitPollInterval(30 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for Active-Active Transit Gateway attachment to be available for subscription %d, region %d, tgw %d", subId, regionId, tgwId)

			tgwTask, err := api.Client.TransitGatewayAttachments.GetActiveActive(ctx, subId, regionId)
			if err != nil {
				return nil, "", err
			}

			// Check for nil response structure
			if tgwTask == nil || tgwTask.Response == nil || tgwTask.Response.Resource == nil {
				return nil, "pending", nil
			}

			// Find the specific attachment by tgwId
			for _, tgw := range tgwTask.Response.Resource.TransitGatewayAttachment {
				if redis.IntValue(tgw.Id) == tgwId {
					status := redis.StringValue(tgw.Status)
					log.Printf("[DEBUG] Active-Active Transit Gateway attachment status: %s", status)

					if transitGatewayAttachmentTerminalStates[status] {
						return nil, status, fmt.Errorf("Active-Active Transit Gateway attachment reached terminal state: %s", status)
					}

					return tgw, status, nil
				}
			}

			// Attachment not found yet
			return nil, "pending", nil
		},
	}

	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("timeout waiting for Active-Active Transit Gateway attachment to become available for subscription %d, region %d, tgw %d. "+
			"Original error: %w", subId, regionId, tgwId, err)
	}

	tgw, ok := result.(*attachments.TransitGatewayAttachment)
	if !ok {
		return nil, fmt.Errorf("internal error: unexpected result type from wait operation for subscription %d, region %d, tgw %d", subId, regionId, tgwId)
	}

	return tgw, nil
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

// Use WaitForPrivateLinkPrincipals to verify exact membership once all expected
// principals are associated or associating. Retry temporary mismatches to allow
// API reads to catch up with writes.
// The region ID is only used for Active-Active subscriptions.
// Pass a context bounded by the calling operation's timeout.
func WaitForPrivateLinkPrincipals(
	ctx context.Context,
	subId int,
	api *client.ApiClient,
	expected []pl.PrivateLinkPrincipal,
	timeout time.Duration,
	subscriptionType string,
	regionId int,
) error {
	// Refresh and timeout handling can run concurrently, so share the last
	// observation atomically for error reporting.
	var lastObservation atomic.Value
	lastObservation.Store("no principal list returned")
	wait := &retry.StateChangeConf{
		Pending:      []string{"pending"},
		Target:       []string{"matched"},
		Timeout:      timeout,
		PollInterval: 10 * time.Second,
		Refresh: func() (interface{}, string, error) {
			var link *pl.PrivateLink
			var err error
			if subscriptionType == subscriptions.SubscriptionDeploymentTypeActiveActive {
				link, err = api.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subId, regionId)
			} else {
				link, err = api.Client.PrivateLink.GetPrivateLink(ctx, subId)
			}
			if err != nil {
				return nil, "", err
			}
			matched, err := privateLinkPrincipalsMatch(link.Principals, expected)
			if err != nil {
				return nil, "", err
			}
			if matched {
				return link, "matched", nil
			}
			lastObservation.Store(formatPrincipalLists(expected, link.Principals))
			return link, "pending", nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("verifying PrivateLink principals: %w (last observed: %s)", err, lastObservation.Load().(string))
	}
	return nil
}

// Aliases label principals without changing access, so match membership by
// principal value and type while checking association status.
func privateLinkPrincipalsMatch(actual []*pl.PrivateLinkPrincipal, expected []pl.PrivateLinkPrincipal) (bool, error) {
	for _, principal := range expected {
		found := false
		for _, candidate := range actual {
			if candidate == nil || redis.StringValue(candidate.Principal) != redis.StringValue(principal.Principal) || redis.StringValue(candidate.Type) != redis.StringValue(principal.Type) {
				continue
			}
			status := redis.StringValue(candidate.Status)
			if status == pl.PrivateLinkPrincipalStatusFailed {
				return false, fmt.Errorf("PrivateLink principal %q association failed", redis.StringValue(principal.Principal))
			}
			// An associating principal can be waiting for the consumer to accept
			// its invitation. Acceptance happens separately from this operation.
			found = status == pl.PrivateLinkPrincipalStatusAssociated || status == pl.PrivateLinkPrincipalStatusAssociating
			break
		}
		if !found {
			return false, nil
		}
	}
	// The resource owns the complete principal set, so an extra principal is
	// also a mismatch.
	return len(actual) == len(expected), nil
}

func formatPrincipalLists(expected []pl.PrivateLinkPrincipal, actual []*pl.PrivateLinkPrincipal) string {
	expectedPrincipals := make([]string, len(expected))
	for i, principal := range expected {
		expectedPrincipals[i] = redis.StringValue(principal.Principal)
	}
	actualPrincipals := make([]string, len(actual))
	for i, principal := range actual {
		if principal != nil {
			actualPrincipals[i] = redis.StringValue(principal.Principal)
		}
	}
	return fmt.Sprintf("expected %v, got %v", expectedPrincipals, actualPrincipals)
}
