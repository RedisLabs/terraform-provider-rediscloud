package paymentmethod

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PaymentMethodDataSourceModel describes the data source data model.
type PaymentMethodDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	CardType        types.String `tfsdk:"card_type"`
	ExcludeExpired  types.Bool   `tfsdk:"exclude_expired"`
	LastFourNumbers types.String `tfsdk:"last_four_numbers"`
}
