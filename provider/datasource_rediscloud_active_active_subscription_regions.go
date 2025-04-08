package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRedisCloudActiveActiveSubscriptionRegions() *schema.Resource {
	return &schema.Resource{
		Description: "Gets a list of regions in the specified Active-Active subscription.",
		ReadContext: dataSourceRedisCloudActiveActiveRegionsRead,

		Schema: map[string]*schema.Schema{
			"subscription_name": {
				Description: "The name of the subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regions": {
				Description: "A list of regions from an active active subscription",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Description: "Deployment region as defined by cloud provider",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"networking_deployment_cidr": {
							Description: "Deployment CIDR mask",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"vpc_id": {
							Description: "VPC ID for the region",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"databases": {
							Description: "A list of databases found in the region",
							Computed:    true,
							Type:        schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"database_id": {
										Description: "A numeric id for the database",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"database_name": {
										Description: "A meaningful name to identify the database",
										Type:        schema.TypeString,
										Required:    true,
									},
									"write_operations_per_second": {
										Description: "Write operations per second for the database",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"read_operations_per_second": {
										Description: "Read operations per second for the database",
										Type:        schema.TypeInt,
										Required:    true,
									},
								},
							},
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

	// Filter down to requested subscription by name
	if name, ok := d.GetOk("subscription_name"); ok {
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

	if err != nil {
		return diag.FromErr(err)
	}

	if len(regions) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	// TODO: may have to manipulate regions to be output in a friendly way here

	if err := d.Set("regions", regions); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
