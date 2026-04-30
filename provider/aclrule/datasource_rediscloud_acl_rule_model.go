package aclrule

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AclRuleModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Rule types.String `tfsdk:"rule"`
}
