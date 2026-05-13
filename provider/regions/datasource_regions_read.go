package regions

import (
	"context"
	"fmt"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// regionAttrTypes defines the attribute types for a single region object.
var regionAttrTypes = map[string]attr.Type{
	"region_id":     types.Int64Type,
	"name":          types.StringType,
	"provider_name": types.StringType,
}

// Read refreshes the Terraform state with the latest data.
func (d *regionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Defensive nil check for client
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var config RegionsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all regions from the API
	regions, err := d.client.Client.Account.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Regions",
			fmt.Sprintf("An error occurred while reading regions: %s", err.Error()),
		)
		return
	}

	// Build filters based on configuration
	var filters []func(region *account.Region) bool

	// Build the synthetic ID — either the provider filter or all providers joined
	id := strings.Join(cloud_accounts.ProviderValues(), "-")
	if !config.ProviderName.IsNull() && config.ProviderName.ValueString() != "" {
		providerName := config.ProviderName.ValueString()
		filters = append(filters, func(region *account.Region) bool {
			if region == nil {
				return false
			}
			return redis.StringValue(region.Provider) == providerName
		})
		id = providerName
	}

	// Apply filters
	regions = filterRegions(ctx, regions, filters)

	if len(regions) == 0 {
		resp.Diagnostics.AddError(
			"No Regions Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	// Set synthetic ID
	config.ID = types.StringValue(id)

	// Flatten regions into the state model
	regionsSet, flattenDiags := flattenRegions(ctx, regions)
	resp.Diagnostics.Append(flattenDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Regions = regionsSet

	// Set state
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

// flattenRegions converts API region objects into a types.Set for the state model.
func flattenRegions(ctx context.Context, regions []*account.Region) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	elemType := types.ObjectType{
		AttrTypes: regionAttrTypes,
	}

	if len(regions) == 0 {
		return types.SetNull(elemType), diags
	}

	var elements []attr.Value
	for _, region := range regions {
		if region == nil {
			tflog.Warn(ctx, "Skipping nil region entry in API response — this may indicate an upstream API issue")
			continue
		}

		model := RegionModel{
			RegionID:     types.Int64Value(int64(redis.IntValue(region.ID))),
			Name:         types.StringValue(redis.StringValue(region.Name)),
			ProviderName: types.StringValue(redis.StringValue(region.Provider)),
		}

		obj, objDiags := types.ObjectValueFrom(ctx, regionAttrTypes, model)
		diags.Append(objDiags...)
		if diags.HasError() {
			return types.SetNull(elemType), diags
		}
		elements = append(elements, obj)
	}

	setValue, setDiags := types.SetValue(elemType, elements)
	diags.Append(setDiags...)
	return setValue, diags
}

// filterRegions applies all filters to the list of regions.
func filterRegions(ctx context.Context, regions []*account.Region, filters []func(region *account.Region) bool) []*account.Region {
	var filtered []*account.Region
	for _, region := range regions {
		if region == nil {
			tflog.Warn(ctx, "Skipping nil region entry in API response — this may indicate an upstream API issue")
			continue
		}
		if filterRegion(region, filters) {
			filtered = append(filtered, region)
		}
	}
	return filtered
}

// filterRegion checks if a single region passes all filters.
func filterRegion(region *account.Region, filters []func(region *account.Region) bool) bool {
	for _, f := range filters {
		if !f(region) {
			return false
		}
	}
	return true
}
