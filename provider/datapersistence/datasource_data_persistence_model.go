package datapersistence

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataPersistenceDataSourceModel describes the data source data model.
type DataPersistenceDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	DataPersistence types.Set    `tfsdk:"data_persistence"`
}

// DataPersistenceOptionModel describes each item in the data_persistence set.
type DataPersistenceOptionModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}
