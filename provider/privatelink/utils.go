package privatelink

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type privateLinkId struct {
	subscriptionId int
	principalId    string
}

func toPrivateLinkId(id string) (*privateLinkId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid id: %s", id)
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	privateLinkIdStr := parts[1]

	return &privateLinkId{
		subscriptionId: subId,
		principalId:    privateLinkIdStr,
	}, nil
}

func waitForPrivateLinkToBeActive(ctx context.Context, id int, client *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			pl.PrivateLinkStatusInitializing},
		Target:       []string{pl.PrivateLinkStatusActive},
		Timeout:      utils.SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 10 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for private link %d to be active", id)

			privatelink, err := client.Client.PrivateLink.GetPrivateLink(ctx, id)
			if err != nil {
				return "", "", err
			}

			return *privatelink.ShareName, *privatelink.Status, nil
		}}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForPrincipalToBeAssociated(ctx context.Context, client *client.ApiClient, id int, principal *string) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			pl.PrivateLinkPrincipalStatusInitializing, pl.PrivateLinkPrincipalStatusAssociating},
		Target:       []string{pl.PrivateLinkPrincipalStatusAssociated},
		Timeout:      utils.SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 10 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for private link principal %d to be associated", id)

			privateLink, err := client.Client.PrivateLink.GetPrivateLink(ctx, id)
			if err != nil {
				return "", "", err
			}

			for _, p := range privateLink.Principals {
				if *p.Principal == *principal {
					return *p.Principal, *p.Status, nil
				}
			}

			return nil, "", fmt.Errorf("principal %s not found", *principal)
		}}

	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
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

func flattenPrincipals(principals []*pl.PrivateLinkPrincipal) []map[string]interface{} {
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

func flattenDatabases(databases []*pl.PrivateLinkDatabase) []map[string]interface{} {
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

func flattenConnections(connections []*pl.PrivateLinkConnection) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, connection := range connections {
		tf := map[string]interface{}{
			"association_id":   redis.StringValue(connection.AssociationId),
			"connection_id":    redis.IntValue(connection.ConnectionId),
			"type":             redis.StringValue(connection.Type),
			"owner_id":         redis.IntValue(connection.OwnerId),
			"association_date": redis.StringValue(connection.AssociationDate),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}
