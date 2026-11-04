package aclrole

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AclRoleDataSourceModel describes the data source data model.
type AclRoleDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Rule RuleSetValue `tfsdk:"rule"`
}
