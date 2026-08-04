package cloudaccount

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CloudAccountResourceModel describes the resource data model.
type CloudAccountResourceModel struct {
	ID              types.String   `tfsdk:"id"`
	AccessKeyID     types.String   `tfsdk:"access_key_id"`
	AccessSecretKey types.String   `tfsdk:"access_secret_key"`
	ConsolePassword types.String   `tfsdk:"console_password"`
	ConsoleUsername types.String   `tfsdk:"console_username"`
	Name            types.String   `tfsdk:"name"`
	ProviderType    types.String   `tfsdk:"provider_type"`
	SignInLoginURL  types.String   `tfsdk:"sign_in_login_url"`
	Status          types.String   `tfsdk:"status"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}
