package transitgateway

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
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
			"wait_for_invitations_timeout": {
				Description: "When set, retry fetching invitations until at least one is found or the specified timeout (in seconds) is reached. " +
					"Useful when creating AWS RAM share and Redis Cloud resources in the same Terraform run.",
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
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
						"name": {
							Description: "The name of the resource share",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"resource_share_uid": {
							Description: "The AWS Resource Share ARN",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"aws_account_id": {
							Description: "The AWS account ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"status": {
							Description: "The invitation status",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"shared_date": {
							Description: "The date the resource was shared",
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

	waitTimeoutSeconds := d.Get("wait_for_invitations_timeout").(int)

	var invitations []*attachments.TransitGatewayInvitation

	if waitTimeoutSeconds > 0 {
		// Wait for invitations to appear
		wait := &retry.StateChangeConf{
			Pending:      []string{"waiting"},
			Target:       []string{"found"},
			Timeout:      time.Duration(waitTimeoutSeconds) * time.Second,
			Delay:        5 * time.Second,
			PollInterval: 10 * time.Second,

			Refresh: func() (result interface{}, state string, err error) {
				log.Printf("[DEBUG] Waiting for Transit Gateway invitations to appear for subscription %d", subId)

				inv, err := api.Client.TransitGatewayAttachments.ListInvitations(ctx, subId)
				if err != nil {
					return nil, "", err
				}

				if len(inv) == 0 {
					return nil, "waiting", nil
				}

				return inv, "found", nil
			},
		}

		result, err := wait.WaitForStateContext(ctx)
		if err != nil {
			return diag.Errorf("Timeout waiting for Transit Gateway invitations to appear for subscription %d: %s", subId, err)
		}
		var ok bool
		invitations, ok = result.([]*attachments.TransitGatewayInvitation)
		if !ok {
			return diag.Errorf("Internal error: unexpected result type from wait operation for subscription %d", subId)
		}
	} else {
		// No waiting - fetch once
		invitations, err = api.Client.TransitGatewayAttachments.ListInvitations(ctx, subId)
		if err != nil {
			return diag.FromErr(err)
		}
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
			"id":                 redis.IntValue(invitation.Id),
			"name":               redis.StringValue(invitation.Name),
			"resource_share_uid": redis.StringValue(invitation.ResourceShareUid),
			"aws_account_id":     redis.StringValue(invitation.AwsAccountId),
			"status":             redis.StringValue(invitation.Status),
			"shared_date":        redis.StringValue(invitation.SharedDate),
		}
		result = append(result, invitationMap)
	}
	return result
}
