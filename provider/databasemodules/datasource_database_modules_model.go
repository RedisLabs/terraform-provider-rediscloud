package databasemodules

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// DatabaseModulesDataSourceModel describes the data source data model.
type DatabaseModulesDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Modules types.Set    `tfsdk:"modules"`
}

// ModuleModel describes each element of the modules set.
type ModuleModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// ModuleElemType is the element type of the computed modules set.
var ModuleElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ModuleModel{})}
