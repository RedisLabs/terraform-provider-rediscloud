package transitgateway

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func DataSourceRedisCloudActiveActiveTransitGatewayInvitations() *schema.Resource {
	return &schema.Resource{
		Description: "Lists AWS Transit Gateway resource share invitations for a specific region in an Active-Active subscription. Invitations are created when you share an AWS Transit Gateway with Redis Cloud via AWS Resource Manager.",
		ReadContext: dataSourceRedisCloudActiveActiveTransitGatewayInvitationsRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of the Active-Active subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"region_id": {
				Description: "The region ID",
				Type:        schema.TypeInt,
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

func dataSourceRedisCloudActiveActiveTransitGatewayInvitationsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	regionId := d.Get("region_id").(int)
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
				log.Printf("[DEBUG] Waiting for Active-Active Transit Gateway invitations to appear for subscription %d, region %d", subId, regionId)

				inv, err := api.Client.TransitGatewayAttachments.ListInvitationsActiveActive(ctx, subId, regionId)
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
			return diag.Errorf("Timeout waiting for Active-Active Transit Gateway invitations to appear for subscription %d, region %d: %s", subId, regionId, err)
		}
		invitations = result.([]*attachments.TransitGatewayInvitation)
	} else {
		// No waiting - fetch once
		invitations, err = api.Client.TransitGatewayAttachments.ListInvitationsActiveActive(ctx, subId, regionId)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("invitations", flattenTransitGatewayInvitations(invitations)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", subId, regionId))

	return diags
}
