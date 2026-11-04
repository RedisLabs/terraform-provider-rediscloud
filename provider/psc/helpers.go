package psc

import (
	"context"
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pscapi "github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// ServiceAttachmentModel represents a GCP service attachment created for a Private Service Connect endpoint.
type ServiceAttachmentModel struct {
	Name               types.String `tfsdk:"name"`
	DNSRecord          types.String `tfsdk:"dns_record"`
	IPAddressName      types.String `tfsdk:"ip_address_name"`
	ForwardingRuleName types.String `tfsdk:"forwarding_rule_name"`
}

var serviceAttachmentElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ServiceAttachmentModel{})}

// ServiceAttachmentsFromAPI maps service attachments as the SDKv2 implementation did, writing "" for missing fields.
func ServiceAttachmentsFromAPI(ctx context.Context, attachments []pscapi.TerraformGCPServiceAttachment) (types.List, diag.Diagnostics) {
	models := make([]ServiceAttachmentModel, 0, len(attachments))
	for _, attachment := range attachments {
		//TODO(TF3.0) make service attachment fields nullable
		models = append(models, ServiceAttachmentModel{
			Name:               types.StringValue(redis.StringValue(attachment.Name)),
			DNSRecord:          types.StringValue(redis.StringValue(attachment.DNSRecord)),
			IPAddressName:      types.StringValue(redis.StringValue(attachment.IPAddressName)),
			ForwardingRuleName: types.StringValue(redis.StringValue(attachment.ForwardingRuleName)),
		})
	}
	return types.ListValueFrom(ctx, serviceAttachmentElemType, models)
}

func buildPrivateServiceConnectId(subId int, pscServiceId int) string {
	return fmt.Sprintf("%d/%d", subId, pscServiceId)
}
