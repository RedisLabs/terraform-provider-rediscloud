package privatelink

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func waitForPrivateLinkToBeActive(ctx context.Context, client *client.ApiClient, subscriptionId int, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			pl.PrivateLinkStatusInitializing},
		Target:       []string{pl.PrivateLinkStatusActive},
		Timeout:      timeout,
		Delay:        client.WaitDelay(10 * time.Second),
		PollInterval: client.WaitPollInterval(10 * time.Second),
		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for private link %d to be active", subscriptionId)

			privateLink, err := client.Client.PrivateLink.GetPrivateLink(ctx, subscriptionId)
			if err != nil {
				// NotFound during wait means the privatelink is still being provisioned
				// (API returns empty response while initialising). Treat as retryable.
				var notFound *pl.NotFound
				if errors.As(err, &notFound) {
					return nil, pl.PrivateLinkStatusInitializing, nil
				}
				return nil, "", err
			}

			return *privateLink.ShareName, *privateLink.Status, nil
		}}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForActiveActivePrivateLinkToBeActive(ctx context.Context, client *client.ApiClient, subscriptionId int, regionId int, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			pl.PrivateLinkStatusInitializing},
		Target:       []string{pl.PrivateLinkStatusActive},
		Timeout:      timeout,
		Delay:        client.WaitDelay(10 * time.Second),
		PollInterval: client.WaitPollInterval(10 * time.Second),
		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for private link %d to be active", subscriptionId)

			privateLink, err := client.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subscriptionId, regionId)
			if err != nil {
				// NotFound during wait means the privatelink is still being provisioned
				// (API returns empty response while initialising). Treat as retryable.
				var notFound *pl.NotFoundActiveActive
				if errors.As(err, &notFound) {
					return nil, pl.PrivateLinkStatusInitializing, nil
				}
				return nil, "", err
			}

			return *privateLink.ShareName, *privateLink.Status, nil
		}}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

// PrivateLinkGetter lets a shared waiter use a subscription or regional PrivateLink read.
type PrivateLinkGetter func(context.Context) (*pl.PrivateLink, error)

// Use WaitForPrivateLinkPrincipals when an operation requires exact principal
// membership. Retry temporary mismatches to allow API reads to catch up with writes.
func WaitForPrivateLinkPrincipals(ctx context.Context, getPrincipals PrivateLinkGetter, expected []pl.PrivateLinkPrincipal, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
			link, err := getPrincipals(ctx)
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
			// An external consumer accepts the invitation separately, so associating is sufficient here.
			found = status != pl.PrivateLinkPrincipalStatusDisassociating && status != pl.PrivateLinkPrincipalStatusDisassociated
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

func principalsFromSet(principals *schema.Set) []pl.PrivateLinkPrincipal {
	var createPrincipals []pl.PrivateLinkPrincipal
	for _, principal := range principals.List() {
		principalMap := principal.(map[string]interface{})

		createPrincipal := pl.PrivateLinkPrincipal{
			Principal: redis.String(principalMap["principal"].(string)),
			Alias:     redis.String(principalMap["principal_alias"].(string)),
			Type:      redis.String(principalMap["principal_type"].(string)),
		}

		createPrincipals = append(createPrincipals, createPrincipal)
	}

	sort.Slice(createPrincipals, func(i, j int) bool {
		return *createPrincipals[i].Principal < *createPrincipals[j].Principal
	})

	return createPrincipals
}

// FlattenPrincipals converts private link principals to Terraform state shape.
func FlattenPrincipals(principals []*pl.PrivateLinkPrincipal) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, principal := range principals {
		tf := map[string]interface{}{
			"principal":       redis.StringValue(principal.Principal),
			"principal_type":  redis.StringValue(principal.Type),
			"principal_alias": redis.StringValue(principal.Alias),
		}
		tfs = append(tfs, tf)
	}

	sort.Slice(tfs, func(i, j int) bool {
		return tfs[i]["principal"].(string) < tfs[j]["principal"].(string)
	})

	return tfs
}

// FlattenDatabases converts private link databases to Terraform state shape.
func FlattenDatabases(databases []*pl.PrivateLinkDatabase) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, db := range databases {
		tf := map[string]interface{}{
			"database_id":            redis.IntValue(db.DatabaseId),
			"port":                   redis.IntValue(db.Port),
			"resource_link_endpoint": redis.StringValue(db.ResourceLinkEndpoint),
		}
		tfs = append(tfs, tf)
	}
	return tfs
}

// FlattenConnections converts private link connections to Terraform state shape.
func FlattenConnections(connections []*pl.PrivateLinkConnection) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, connection := range connections {
		tf := map[string]interface{}{
			"association_id":   redis.StringValue(connection.AssociationId),
			"connection_id":    redis.StringValue(connection.ConnectionId),
			"connection_type":  redis.StringValue(connection.Type),
			"owner_id":         redis.StringValue(connection.OwnerId),
			"association_date": redis.StringValue(connection.AssociationDate),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func makeActiveActivePrivateLinkId(subId int, regionId int) string {
	return strconv.Itoa(subId) + "/" + strconv.Itoa(regionId)
}
