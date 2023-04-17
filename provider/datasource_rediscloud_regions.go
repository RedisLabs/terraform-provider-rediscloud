package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strings"
)

func dataSourceRedisCloudRegions() *schema.Resource {
	return &schema.Resource{
		Description: "The Regions data source allows access to a list of supported cloud provider regions. These regions can be used with the subscription resource.",
		ReadContext: dataSourceRedisCloudRegionsRead,

		Schema: map[string]*schema.Schema{
			"provider_name": {
				Description:      "The name of the cloud provider to filter returned regions, (accepted values are `AWS` or `GCP`).",
				Optional:         true,
				Type:             schema.TypeString,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
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

func dataSourceRedisCloudRegionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	regions, err := api.client.Account.ListRegions(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *account.Region) bool

	var id = strings.Join(cloud_accounts.ProviderValues(), "-")
	if provider, ok := d.GetOk("provider_name"); ok {
		filters = append(filters, func(region *account.Region) bool {
			return formattedProvider(region) == provider
		})
		id = provider.(string)
	}

	regions = filterRegions(regions, filters)

	if len(regions) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	d.SetId(id)
	if err := d.Set("regions", flattenRegions(regions)); err != nil {
		return diag.FromErr(err)
	}

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
