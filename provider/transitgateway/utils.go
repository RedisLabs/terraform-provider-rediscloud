package transitgateway

import "github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"

func filterTgwAttachments(getAttachmentsTask *attachments.GetAttachmentsTask, filters []func(tgwa *attachments.TransitGatewayAttachment) bool) []*attachments.TransitGatewayAttachment {
	var filtered []*attachments.TransitGatewayAttachment
	for _, tgwa := range getAttachmentsTask.Response.Resource.TransitGatewayAttachment {
		if filterTgwAttachment(tgwa, filters) {
			filtered = append(filtered, tgwa)
		}
	}
	return filtered
}

func filterTgwAttachment(tgwa *attachments.TransitGatewayAttachment, filters []func(tgwa *attachments.TransitGatewayAttachment) bool) bool {
	for _, filter := range filters {
		if !filter(tgwa) {
			return false
		}
	}
	return true
}
