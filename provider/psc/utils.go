package psc

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"log"
	"strconv"
	"strings"
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

	endpoint := findPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return utils.PlaceholderStatusDisappear, utils.PlaceholderStatusDisappear, nil
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}

func findPrivateServiceConnectEndpoints(id int, endpoints []*psc.PrivateServiceConnectEndpoint) *psc.PrivateServiceConnectEndpoint {
	for _, endpoint := range endpoints {
		if redis.IntValue(endpoint.ID) == id {
			return endpoint
		}
	}
	return nil
}

func flattenPrivateServiceConnectEndpointServiceAttachments(serviceAttachments []psc.TerraformGCPServiceAttachment) []map[string]interface{} {
	var rl []map[string]interface{}
	for _, serviceAttachment := range serviceAttachments {

		serviceAttachmentMapString := map[string]interface{}{
			"name":                 serviceAttachment.Name,
			"dns_record":           serviceAttachment.DNSRecord,
			"ip_address_name":      serviceAttachment.IPAddressName,
			"forwarding_rule_name": serviceAttachment.ForwardingRuleName,
		}

		rl = append(rl, serviceAttachmentMapString)
	}

	return rl
}

func refreshPrivateServiceConnectServiceEndpointActiveActiveStatus(ctx context.Context, subscriptionId int, regionId int,
	pscServiceId int, endpointId int, targetStatus string, api *utils.ApiClient) (result interface{}, state string, err error) {
	log.Printf("[DEBUG] Waiting for private service connect service endpoint status %d/%d/%d/%d to be %s",
		subscriptionId, regionId, pscServiceId, endpointId, targetStatus)

	endpoints, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpoints(ctx, subscriptionId, regionId, pscServiceId)
	if err != nil {
		return nil, "", err
	}

	endpoint := findPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return nil, "", fmt.Errorf("endpoint with id %d not found", endpointId)
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}
