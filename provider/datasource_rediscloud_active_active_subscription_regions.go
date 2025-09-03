package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRedisCloudActiveActiveSubscriptionRegions() *schema.Resource {
	return &schema.Resource{
		Description: "Gets a list of regions in the specified Active-Active subscription.",
		ReadContext: dataSourceRedisCloudActiveActiveRegionsRead,

		Schema: map[string]*schema.Schema{
			"subscription_name": {
				Description: "The name of the Active-Active subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regions": {
				Description: "A list of regions associated with an Active-Active subscription",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Description: "Deployment region as defined by the cloud provider",
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
										Computed:    true,
									},
									"database_name": {
										Description: "The name of the database",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"write_operations_per_second": {
										Description: "Write operations per second for the database",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"read_operations_per_second": {
										Description: "Read operations per second for the database",
										Type:        schema.TypeInt,
										Computed:    true,
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
	api := meta.(*client.ApiClient)

	subs, err := api.Client.Subscription.List(ctx)

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

	regions, err := api.Client.Subscription.ListActiveActiveRegions(ctx, *sub.ID)

	if err != nil {
		return diag.FromErr(err)
	}

	if len(regions) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	var genericRegions = flattenActiveActiveRegions(regions)

	id := fmt.Sprintf("%d-active-active-regions", *sub.ID)
	d.SetId(id)

	if err := d.Set("regions", genericRegions); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

// generifies the region/db data so it can be put into the terraform schema
func flattenActiveActiveRegions(regionList []*subscriptions.ActiveActiveRegion) []map[string]interface{} {
	var rl []map[string]interface{}
	for _, currentRegion := range regionList {

		var dbs []map[string]interface{}
		for _, db := range currentRegion.Databases {
			dbMap := map[string]interface{}{
				"database_id":                 db.DatabaseId,
				"database_name":               db.DatabaseName,
				"write_operations_per_second": db.WriteOperationsPerSecond,
				"read_operations_per_second":  db.ReadOperationsPerSecond,
			}
			dbs = append(dbs, dbMap)
		}

		regionMap := map[string]interface{}{
			"region":                     currentRegion.Region,
			"networking_deployment_cidr": currentRegion.DeploymentCIDR,
			"vpc_id":                     currentRegion.VpcId,
			"databases":                  dbs,
		}
		rl = append(rl, regionMap)
	}
	return rl
}
