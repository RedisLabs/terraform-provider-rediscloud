package cloudaccount

import "github.com/hashicorp/terraform-plugin-framework/types"

// CloudAccountDataSourceModel describes the data source data model.
type CloudAccountDataSourceModel struct {
	ID                     types.Int64  `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	AccessKeyID            types.String `tfsdk:"access_key_id"`
	ExcludeInternalAccount types.Bool   `tfsdk:"exclude_internal_account"`
	ProviderType           types.String `tfsdk:"provider_type"`
}
