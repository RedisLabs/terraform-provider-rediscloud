package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceRedisCloudActiveActiveSubscriptionRegions() *schema.Resource {
	return &schema.Resource{
		Description: "The Active Active Subscription Regions data source allows access to a list of supported cloud provider regions. These regions can be used with the active active subscription resource.",
		ReadContext: dataSourceRedisCloudActiveActiveRegionsRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "The name of the cloud provider to filter returned regions, (accepted values are `AWS` or `GCP`).",
				Optional:         true,
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
			},
			"provider_name": {
				Description:      "The name of the cloud provider to filter returned regions, (accepted values are `AWS` or `GCP`).",
				Optional:         true,
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
			},
			"regions": {
				Description: "A list of regions from either a single or multiple cloud providers",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The identifier assigned by the cloud provider, (for example `eu-west-1` for `AWS`)",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"provider_name": {
							Description: "The identifier of the owning cloud provider, (either `AWS` or `GCP`)",
							Computed:    true,
							Type:        schema.TypeString,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudActiveActiveRegionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subs, err := api.client.Subscription.List(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *subscriptions.Subscription) bool

	// Filter to active-active subscriptions only (pro subs come from the same endpoint)
	filters = append(filters, func(sub *subscriptions.Subscription) bool {
		return redis.StringValue(sub.DeploymentType) == "active-active"
	})

	if name, ok := d.GetOk("name"); ok {
		filters = append(filters, func(sub *subscriptions.Subscription) bool {
			return redis.StringValue(sub.Name) == name
		})
	}

	subs = filterSubscriptions(subs, filters)

	if len(subs) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(subs) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	sub := subs[0]

	regions, err := api.client.Subscription.ListActiveActiveRegions(ctx, *sub.ID)

	if len(regions) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	return diags

}
