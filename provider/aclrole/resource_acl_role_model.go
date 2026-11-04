package aclrole

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AclRoleResourceModel describes the resource data model.
type AclRoleResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	Rule     RuleSetValue   `tfsdk:"rule"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}
