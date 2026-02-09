package datapersistence

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dataPersistenceOptionAttrTypes defines the attribute types for DataPersistenceOptionModel.
// This must match the tfsdk tags in DataPersistenceOptionModel.
var dataPersistenceOptionAttrTypes = map[string]attr.Type{
	"name":        types.StringType,
	"description": types.StringType,
}

// Read refreshes the Terraform state with the latest data.
func (d *dataPersistenceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataPersistenceDataSourceModel

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	// Fetch data persistence options from the API
	dataPersistence, err := d.client.Client.Account.ListDataPersistence(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Data Persistence Options",
			err.Error(),
		)
		return
	}

	// Set the ID
	state.ID = types.StringValue("ALL")

	// Convert API response to state
	dataPersistenceSet, diags := flattenDataPersistence(ctx, dataPersistence)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.DataPersistence = dataPersistenceSet

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// flattenDataPersistence converts the API response to Plugin Framework types.
func flattenDataPersistence(ctx context.Context, dataPersistenceList []*account.DataPersistence) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	elemType := types.ObjectType{AttrTypes: dataPersistenceOptionAttrTypes}

	if len(dataPersistenceList) == 0 {
		return types.SetNull(elemType), diags
	}

	var elements []attr.Value
	for _, dp := range dataPersistenceList {
		if dp == nil {
			continue
		}

		// Create model instance from API response
		model := DataPersistenceOptionModel{
			Name:        types.StringValue(redis.StringValue(dp.Name)),
			Description: types.StringValue(redis.StringValue(dp.Description)),
		}

		// Convert model to types.Object using reflection on tfsdk tags
		obj, objDiags := types.ObjectValueFrom(ctx, dataPersistenceOptionAttrTypes, model)
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
