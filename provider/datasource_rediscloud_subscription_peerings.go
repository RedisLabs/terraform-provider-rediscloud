package provider

import (
	"context"
	"regexp"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceRedisCloudSubscriptionPeerings() *schema.Resource {
	return &schema.Resource{
		Description: "The Subscription Peerings data source allows access to a list of VPC peerings configured on the subscription.",
		ReadContext: dataSourceRedisCloudSubscriptionPeeringsRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "A valid subscription predefined in the current account",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
			},
			"status": {
				Description: "Current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`",
				Optional:    true,
				Type:        schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					subscriptions.VPCPeeringStatusInitiatingRequest,
					subscriptions.VPCPeeringStatusActive,
					subscriptions.VPCPeeringStatusInactive,
					subscriptions.VPCPeeringStatusPendingAcceptance,
					subscriptions.VPCPeeringStatusFailed,
				}, false)),
			},
			"peerings": {
				Description: "A list of VPC peerings from either a single or multiple cloud providers",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"peering_id": {
							Description: "The identifier of the peering",
							Computed:    true,
							Type:        schema.TypeInt,
						},
						"provider_name": {
							Description: "The identifier of the owning cloud provider, (either `AWS` or `GCP`)",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"aws_account_id": {
							Description: "AWS account id that the VPC to be peered lives in",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"vpc_id": {
							Description: "Identifier of the VPC to be peered",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"vpc_cidr": {
							Description: "CIDR range of the VPC to be peered",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"gcp_project_id": {
							Description: "GCP project ID that the VPC to be peered lives in",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"gcp_network_name": {
							Description: "The name of the network to be peered",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"status": {
							Description: "Current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"aws_peering_id": {
							Description: "Identifier of the AWS cloud peering",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"region": {
							Description: "The name of the AWS region in which your VPC exists",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"gcp_redis_project_id": {
							Description: "Identifier of the Redis Enterprise Cloud GCP project to be peered",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"gcp_redis_network_name": {
							Description: "The name of the Redis Enterprise Cloud network to be peered",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"gcp_peering_id": {
							Description: "Identifier of the cloud peering",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudSubscriptionPeeringsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	vpcPeering, err := api.client.Subscription.ListVPCPeering(ctx, subId)

	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *subscriptions.VPCPeering) bool

	var id = "ALL"

	if v, ok := d.GetOk("status"); ok {
		filters = append(filters, func(vpcPeering *subscriptions.VPCPeering) bool {
			return redis.StringValue(vpcPeering.Status) == v.(string)
		})
		id = v.(string)
	}

	vpcPeering = filterVPCPeerings(vpcPeering, filters)

	d.SetId(id)
	if err := d.Set("peerings", flattenVPCPeering(vpcPeering)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resolveProviderFromVPCPeering(vpcPeering *subscriptions.VPCPeering) string {
	if vpcPeering.VPCId != nil && redis.StringValue(vpcPeering.VPCId) != "" {
		return "AWS"
	} else if vpcPeering.GCPProjectUID != nil && redis.StringValue(vpcPeering.GCPProjectUID) != "" {
		return "GCP"
	} else {
		return "Unknown"
	}
}

func filterVPCPeerings(vpcPeerings []*subscriptions.VPCPeering, filters []func(*subscriptions.VPCPeering) bool) []*subscriptions.VPCPeering {
	var filteredVPCPeerings []*subscriptions.VPCPeering
	for _, vpcPeering := range vpcPeerings {
		if filterVPCPeering(vpcPeering, filters) {
			filteredVPCPeerings = append(filteredVPCPeerings, vpcPeering)
		}
	}

	return filteredVPCPeerings
}

func filterVPCPeering(vpcPeering *subscriptions.VPCPeering, filters []func(*subscriptions.VPCPeering) bool) bool {
	for _, f := range filters {
		if !f(vpcPeering) {
			return false
		}
	}
	return true
}

func flattenVPCPeering(vpcPeerings []*subscriptions.VPCPeering) []map[string]interface{} {

	var rl []map[string]interface{}
	for _, currentVPCPeering := range vpcPeerings {

		peeringMapString := map[string]interface{}{
			"peering_id":             currentVPCPeering.ID,
			"provider_name":          resolveProviderFromVPCPeering(currentVPCPeering),
			"status":                 currentVPCPeering.Status,
			"aws_account_id":         currentVPCPeering.AWSAccountID,
			"aws_peering_id":         currentVPCPeering.AWSPeeringID,
			"vpc_id":                 currentVPCPeering.VPCId,
			"vpc_cidr":               currentVPCPeering.VPCCidr,
			"gcp_project_id":         currentVPCPeering.GCPProjectUID,
			"gcp_network_name":       currentVPCPeering.NetworkName,
			"gcp_redis_project_id":   currentVPCPeering.RedisProjectUID,
			"gcp_redis_network_name": currentVPCPeering.RedisNetworkName,
			"gcp_peering_id":         currentVPCPeering.CloudPeeringID,
			"region":                 currentVPCPeering.Region,
		}

		rl = append(rl, peeringMapString)
	}

	return rl
}
