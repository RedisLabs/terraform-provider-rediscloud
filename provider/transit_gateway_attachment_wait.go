package provider

import (
	"fmt"
	"slices"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func transitGatewayAttachmentFilterMatcher(
	filter func(tgw *attachments.TransitGatewayAttachment) bool,
) utils.TransitGatewayAttachmentMatcher {
	return func(tgw *attachments.TransitGatewayAttachment) (bool, string, error) {
		if !filter(tgw) {
			return false, "", nil
		}

		return true, utils.TransitGatewayAttachmentWaitStateFound, nil
	}
}

func tgwAttachmentFilterMatcher(filters []func(tgwa *attachments.TransitGatewayAttachment) bool) utils.TransitGatewayAttachmentMatcher {
	return transitGatewayAttachmentFilterMatcher(func(tgwa *attachments.TransitGatewayAttachment) bool {
		return filterTgwAttachment(tgwa, filters)
	})
}

func transitGatewayAttachmentIDMatcher(tgwID int) utils.TransitGatewayAttachmentMatcher {
	return transitGatewayAttachmentFilterMatcher(func(tgw *attachments.TransitGatewayAttachment) bool {
		return redis.IntValue(tgw.Id) == tgwID
	})
}

func transitGatewayAttachmentAvailableMatcher(
	tgwID int,
	gatewayName string,
) utils.TransitGatewayAttachmentMatcher {
	return func(tgw *attachments.TransitGatewayAttachment) (bool, string, error) {
		if redis.IntValue(tgw.Id) != tgwID {
			return false, "", nil
		}

		status := redis.StringValue(tgw.Status)
		if status == utils.TransitGatewayAttachmentStatusAvailable {
			return true, status, nil
		}
		if slices.Contains(utils.TransitGatewayAttachmentPendingStates, status) {
			return false, status, nil
		}
		if slices.Contains(utils.TransitGatewayAttachmentTerminalStates, status) {
			return false, status, fmt.Errorf("%s attachment reached terminal state: %s", gatewayName, status)
		}

		return false, status, fmt.Errorf("%s attachment has unexpected state: %s", gatewayName, status)
	}
}
