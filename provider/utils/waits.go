package utils

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/redis_rules"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"log"
	"time"
)

// This timeout is an absolute maximum used in some of the waitForStatus operations concerning creation and updating
// Subscriptions and Databases. Reads and Deletions have their own, stricter timeouts because they consistently behave
// well. The Terraform operation-level timeout should kick in way before we hit this and kill the task.
// Unfortunately there's no "time-remaining-before-timeout" utility, or we could use that in the wait blocks.
const SafetyTimeout = 6 * time.Hour

func WaitForSubscriptionToBeActive(ctx context.Context, id int, api *ApiClient) error {
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

func WaitForSubscriptionToBeEncryptionKeyPending(ctx context.Context, id int, api *ApiClient) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusPending},
		Target:       []string{subscriptions.SubscriptionStatusEncryptionKeyPending, subscriptions.SubscriptionStatusActive},
		Timeout:      SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

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

func WaitForSubscriptionToBeDeleted(ctx context.Context, id int, api *ApiClient) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusDeleting},
		Target:       []string{"deleted"}, // TODO: update this with deleted field in SDK
		Timeout:      SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be deleted", id)

			subscription, err := api.Client.Subscription.Get(ctx, id)
			if err != nil {
				if _, ok := err.(*subscriptions.NotFound); ok {
					return "deleted", "deleted", nil
				} // TODO: update this with deleted field in SDK
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

func WaitForDatabaseToBeActive(ctx context.Context, subId, id int, api *ApiClient) error {
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

func WaitForPrivateServiceConnectServiceToBeActive(ctx context.Context, refreshFunc func() (result interface{}, state string, err error)) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			psc.ServiceStatusCreateQueued,
			psc.ServiceStatusInitialized,
			psc.ServiceStatusCreatePending},
		Target:       []string{psc.ServiceStatusActive},
		Timeout:      SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: refreshFunc,
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

const PlaceholderStatusDisappear = "disappeared"

func WaitForPrivateServiceConnectServiceEndpointDisappear(ctx context.Context, refreshFunc func() (result interface{}, state string, err error)) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			psc.EndpointStatusProcessing,
			psc.EndpointStatusPending,
			psc.EndpointStatusAcceptPending,
			psc.EndpointStatusActive,
			psc.EndpointStatusDeleted,
			psc.EndpointStatusRejected,
			psc.EndpointStatusRejectPending,
			psc.EndpointStatusFailed,
		},
		Target:       []string{PlaceholderStatusDisappear},
		Timeout:      SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: refreshFunc,
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func WaitForPrivateServiceConnectServiceEndpointToBePending(ctx context.Context, refreshFunc func(targetStatus string) (result interface{}, state string, err error)) error {
	targetStatus := psc.EndpointStatusPending
	return waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx, func() (result interface{}, state string, err error) {
		return refreshFunc(targetStatus)
	}, targetStatus, []string{
		psc.EndpointStatusInitialized,
		psc.EndpointStatusProcessing})
}

func WaitForPrivateServiceConnectServiceEndpointToBeActive(ctx context.Context, refreshFunc func(targetStatus string) (result interface{}, state string, err error)) error {
	targetStatus := psc.EndpointStatusActive
	return waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx, func() (result interface{}, state string, err error) {
		return refreshFunc(targetStatus)
	}, targetStatus, []string{
		psc.EndpointStatusPending,
		psc.EndpointStatusAcceptPending})
}

func WaitForPrivateServiceConnectServiceEndpointToBeRejected(ctx context.Context, refreshFunc func(targetStatus string) (result interface{}, state string, err error)) error {
	targetStatus := psc.EndpointStatusRejected
	return waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx, func() (result interface{}, state string, err error) {
		return refreshFunc(targetStatus)
	}, targetStatus, []string{
		psc.EndpointStatusPending,
		psc.EndpointStatusRejectPending})
}

func waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx context.Context,
	refreshFunc func() (result interface{}, state string, err error), status string, pendingStatus []string) error {
	wait := &retry.StateChangeConf{
		Pending:      pendingStatus,
		Target:       []string{status},
		Timeout:      SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: refreshFunc,
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func WaitForAclRuleToBeActive(ctx context.Context, id int, api *ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   5 * time.Second,
		Pending: []string{redis_rules.StatusPending},
		Target:  []string{redis_rules.StatusActive},
		Timeout: 5 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for rule %d to be active", id)

			rule, err := api.Client.RedisRules.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(rule.Status), redis.StringValue(rule.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
