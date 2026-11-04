package acluser

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AclUserResourceModel describes the resource data model.
type AclUserResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	Role     types.String   `tfsdk:"role"`
	Password types.String   `tfsdk:"password"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}
