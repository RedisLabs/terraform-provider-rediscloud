package aclrule

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AclRuleResourceModel describes the resource data model.
type AclRuleResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	Rule     types.String   `tfsdk:"rule"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}
