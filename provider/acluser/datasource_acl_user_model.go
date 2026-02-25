package acluser

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AclUserDataSourceModel describes the data source data model.
type AclUserDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Role types.String `tfsdk:"role"`
}
