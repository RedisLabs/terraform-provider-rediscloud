package regions

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RegionsDataSourceModel describes the data source data model.
type RegionsDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProviderName types.String `tfsdk:"provider_name"`
	Regions      types.Set    `tfsdk:"regions"`
}

// RegionModel describes a single region within the regions set.
type RegionModel struct {
	RegionID     types.Int64  `tfsdk:"region_id"`
	Name         types.String `tfsdk:"name"`
	ProviderName types.String `tfsdk:"provider_name"`
}
