package databasemodules

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *databaseModulesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	modules, err := d.client.Client.Account.ListDatabaseModules(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Database Modules",
			err.Error(),
		)
		return
	}

	moduleSet, diags := ModulesFromAPI(ctx, modules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := DatabaseModulesDataSourceModel{
		ID:      types.StringValue("ALL"),
		Modules: moduleSet,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
