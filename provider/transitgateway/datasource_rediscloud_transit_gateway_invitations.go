package transitgateway

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceRedisCloudTransitGatewayInvitations() *schema.Resource {
	return &schema.Resource{
		Description: "Lists AWS Transit Gateway resource share invitations for a Pro subscription. Invitations are created when you share an AWS Transit Gateway with Redis Cloud via AWS Resource Manager.",
		ReadContext: dataSourceRedisCloudTransitGatewayInvitationsRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of the Pro subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"invitations": {
				Description: "List of Transit Gateway invitations",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The invitation ID",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"tgw_id": {
							Description: "The Transit Gateway ID",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"aws_tgw_uid": {
							Description: "The AWS Transit Gateway UID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"status": {
							Description: "The invitation status",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"aws_account_id": {
							Description: "The AWS account ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudTransitGatewayInvitationsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	invitations, err := api.Client.TransitGatewayAttachments.ListInvitations(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("invitations", flattenTransitGatewayInvitations(invitations)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	return diags
}

func flattenTransitGatewayInvitations(invitations []*attachments.TransitGatewayInvitation) []map[string]interface{} {
	var result []map[string]interface{}
	for _, invitation := range invitations {
		invitationMap := map[string]interface{}{
			"id":             redis.IntValue(invitation.Id),
			"tgw_id":         redis.IntValue(invitation.TgwId),
			"aws_tgw_uid":    redis.StringValue(invitation.AwsTgwUid),
			"status":         redis.StringValue(invitation.Status),
			"aws_account_id": redis.StringValue(invitation.AwsAccountId),
		}
		result = append(result, invitationMap)
	}
	return result
}
