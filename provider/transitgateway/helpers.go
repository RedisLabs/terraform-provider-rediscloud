package transitgateway

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseTransitGatewayAttachmentId parses a Pro TGW attachment ID with format "subscription_id/tgw_id".
func ParseTransitGatewayAttachmentId(id string) (subscriptionId int, tgwId int, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected format of ID (%q), expected <subscription_id>/<tgw_id>", id)
	}

	subscriptionId, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse subscription_id: %w", err)
	}
	tgwId, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse tgw_id: %w", err)
	}

	return subscriptionId, tgwId, nil
}

// ParseActiveActiveTransitGatewayAttachmentId parses an AA TGW attachment ID with format "subscription_id/region_id/tgw_id".
func ParseActiveActiveTransitGatewayAttachmentId(id string) (subscriptionId int, regionId int, tgwId int, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("unexpected format of ID (%q), expected <subscription_id>/<region_id>/<tgw_id>", id)
	}

	subscriptionId, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse subscription_id: %w", err)
	}
	regionId, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse region_id: %w", err)
	}
	tgwId, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse tgw_id: %w", err)
	}

	return subscriptionId, regionId, tgwId, nil
}

// BuildActiveActiveTransitGatewayAttachmentId builds an AA TGW attachment ID with format "subscription_id/region_id/tgw_id".
func BuildActiveActiveTransitGatewayAttachmentId(subscriptionId, regionId, tgwId int) string {
	return fmt.Sprintf("%d/%d/%d", subscriptionId, regionId, tgwId)
}
