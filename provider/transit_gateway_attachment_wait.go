package provider

import (
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func transitGatewayAttachmentIDMatcher(tgwID int) utils.TransitGatewayAttachmentMatcher {
	return func(tgw *attachments.TransitGatewayAttachment) (bool, error) {
		return redis.IntValue(tgw.Id) == tgwID, nil
	}
}

func transitGatewayAttachmentAvailableMatcher(
	tgwID int,
	gatewayName string,
) utils.TransitGatewayAttachmentMatcher {
	return func(tgw *attachments.TransitGatewayAttachment) (bool, error) {
		if redis.IntValue(tgw.Id) != tgwID {
			return false, nil
		}

		status := redis.StringValue(tgw.Status)
		switch status {
		case "available":
			return true, nil
		case "provisioning", "initiating-request", "pending", "pendingAcceptance", "modifying", "deleting", "rejecting", "failing", "rollingBack":
			return false, nil
		case "failed", "rejected", "deleted":
			return false, fmt.Errorf("%s attachment reached terminal state: %s", gatewayName, status)
		default:
			return false, fmt.Errorf("%s attachment has unexpected state: %s", gatewayName, status)
		}
	}
}
