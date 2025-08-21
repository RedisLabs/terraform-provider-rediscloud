package private_service_connect

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

func buildPrivateServiceConnectEndpointId(subId int, pscId int, endpointId int) string {
	return privateServiceConnectEndpointId{
		subscriptionId: subId,
		pscServiceId:   pscId,
		endpointId:     endpointId}.String()
}

type privateServiceConnectEndpointId struct {
	subscriptionId int
	pscServiceId   int
	endpointId     int
}

func (p privateServiceConnectEndpointId) String() string {
	return fmt.Sprintf("%d/%d/%d", p.subscriptionId, p.pscServiceId, p.endpointId)
}

func toPscEndpointId(id string) (*privateServiceConnectEndpointId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id: %s", id)
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	pscId, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	endpointId, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}

	return &privateServiceConnectEndpointId{
		subscriptionId: subId,
		pscServiceId:   pscId,
		endpointId:     endpointId,
	}, nil
}

func refreshPrivateServiceConnectServiceEndpointDisappear(ctx context.Context, subscriptionId int,
	pscServiceId int, endpointId int, api *utils.ApiClient) (result interface{}, state string, err error) {

	log.Printf("[DEBUG] Waiting for private service connect service endpoint %d/%d/%d to be deleted",
		subscriptionId, pscServiceId, endpointId)

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, subscriptionId, pscServiceId)
	if err != nil {
		return nil, "", err
	}

	endpoint := FindPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return utils.PlaceholderStatusDisappear, utils.PlaceholderStatusDisappear, nil
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}

func FindPrivateServiceConnectEndpoints(id int, endpoints []*psc.PrivateServiceConnectEndpoint) *psc.PrivateServiceConnectEndpoint {
	for _, endpoint := range endpoints {
		if redis.IntValue(endpoint.ID) == id {
			return endpoint
		}
	}
	return nil
}

func refreshPrivateServiceConnectServiceEndpointActiveActiveStatus(ctx context.Context, subscriptionId int, regionId int,
	pscServiceId int, endpointId int, targetStatus string, api *utils.ApiClient) (result interface{}, state string, err error) {
	log.Printf("[DEBUG] Waiting for private service connect service endpoint status %d/%d/%d/%d to be %s",
		subscriptionId, regionId, pscServiceId, endpointId, targetStatus)

	endpoints, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpoints(ctx, subscriptionId, regionId, pscServiceId)
	if err != nil {
		return nil, "", err
	}

	endpoint := FindPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return nil, "", fmt.Errorf("endpoint with id %d not found", endpointId)
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}

func toVpcPeeringId(id string) (int, int, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid id: %s", id)
	}

	sub, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	peering, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return sub, peering, nil
}

func findVpcPeering(id int, peerings []*subscriptions.VPCPeering) *subscriptions.VPCPeering {
	for _, peering := range peerings {
		if redis.IntValue(peering.ID) == id {
			return peering
		}
	}
	return nil
}

func waitForPeeringToBeInitiated(ctx context.Context, subId, id int, api *utils.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay: 10 * time.Second,
		Pending: []string{
			subscriptions.VPCPeeringStatusInitiatingRequest,
		},
		Target: []string{
			subscriptions.VPCPeeringStatusActive,
			subscriptions.VPCPeeringStatusInactive,
			subscriptions.VPCPeeringStatusPendingAcceptance,
		},
		Timeout: 10 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for vpc peering %d to be initiated", id)

			list, err := api.Client.Subscription.ListVPCPeering(ctx, subId)
			if err != nil {
				return nil, "", err
			}

			peering := findVpcPeering(id, list)
			if peering == nil {
				log.Printf("Peering %d/%d not present yet", subId, id)
				return nil, "", nil
			}

			return redis.StringValue(peering.Status), redis.StringValue(peering.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
