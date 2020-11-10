package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"time"
)

func dataSourceRedisCloudRegions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRedisCloudRegionsRead,

		Schema: map[string]*schema.Schema{
			"provider_name": {
				Optional: true,
				Type:     schema.TypeString,
				ValidateDiagFunc: toDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
			},
			"regions": {
				Type: schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Computed: true,
							Type: schema.TypeString,
						},
						"provider_name": {
							Computed: true,
							Type: schema.TypeString,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudRegionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	regions, err := api.client.Account.ListRegions(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *account.Region) bool

	if provider, ok := d.GetOk("provider_name"); ok {
		filters = append(filters, func(region *account.Region) bool {
			return formattedProvider(region) == provider
		})
	}

	regions = filterRegions(regions, filters)

	if len(regions) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	d.SetId(time.Now().UTC().String())
	d.Set("regions", flattenRegions(regions))

	return diags
}

func formattedProvider(region *account.Region) string {
	return redis.StringValue(region.Provider)
}

func filterRegions(regions []*account.Region, filters []func(region *account.Region) bool) []*account.Region {
	var filteredRegions []*account.Region
	for _, region := range regions {
		if filterRegion(region, filters) {
			filteredRegions = append(filteredRegions, region)
		}
	}

	return filteredRegions
}

func filterRegion(region *account.Region, filters []func(region *account.Region) bool) bool {
	for _, f := range filters {
		if !f(region) {
			return false
		}
	}
	return true
}

func flattenRegions(regionList []*account.Region) []map[string]interface{} {

	var rl []map[string]interface{}
	for _, currentRegion := range regionList {

		regionMapString := map[string]interface{}{
			"name":          currentRegion.Name,
			"provider_name": currentRegion.Provider,
		}

		rl = append(rl, regionMapString)
	}

	return rl
}
