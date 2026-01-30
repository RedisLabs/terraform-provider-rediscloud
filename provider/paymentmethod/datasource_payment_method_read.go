package paymentmethod

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Read refreshes the Terraform state with the latest data.
func (d *paymentMethodDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Defensive nil check for client
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var state PaymentMethodDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all payment methods from the API
	methods, err := d.client.Client.Account.ListPaymentMethods(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Payment Methods",
			fmt.Sprintf("An error occurred while reading payment methods: %s", err.Error()),
		)
		return
	}

	// Build filters based on configuration
	var filters []func(method *account.PaymentMethod) bool

	// Handle exclude_expired filter - default to true if not set
	excludeExpired := true
	if !state.ExcludeExpired.IsNull() {
		excludeExpired = state.ExcludeExpired.ValueBool()
	}

	if excludeExpired {
		now := time.Now()
		filters = append(filters, func(method *account.PaymentMethod) bool {
			if method == nil {
				return false
			}

			expirationYear := redis.IntValue(method.ExpirationYear)
			expirationMonth := redis.IntValue(method.ExpirationMonth)

			if expirationYear < now.Year() {
				// Expiration year is last year, so it must already have expired
				return false
			}

			if expirationYear > now.Year() {
				// Expiration year is next year, so it cannot have expired
				return true
			}

			// Expiration year is this year, so we do have to check the month
			if expirationMonth < int(now.Month()) {
				return false
			}

			return true
		})
	}

	// Filter by card type if specified
	if !state.CardType.IsNull() && state.CardType.ValueString() != "" {
		cardType := state.CardType.ValueString()
		filters = append(filters, func(method *account.PaymentMethod) bool {
			if method == nil {
				return false
			}
			return redis.StringValue(method.Type) == cardType
		})
	}

	// Filter by last four numbers if specified
	if !state.LastFourNumbers.IsNull() && state.LastFourNumbers.ValueString() != "" {
		lastFour := state.LastFourNumbers.ValueString()
		filters = append(filters, func(method *account.PaymentMethod) bool {
			if method == nil {
				return false
			}
			return formattedCardNumber(method) == lastFour
		})
	}

	// Apply filters
	methods = filterPaymentMethods(methods, filters)

	// Check for exactly one result
	if len(methods) == 0 {
		resp.Diagnostics.AddError(
			"No Payment Methods Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(methods) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Payment Methods Found",
			"Your query returned more than one result. Please try a more specific search criteria and try again.",
		)
		return
	}

	// Map the result to state
	method := methods[0]
	state.ID = types.StringValue(strconv.Itoa(redis.IntValue(method.ID)))
	state.CardType = types.StringValue(redis.StringValue(method.Type))
	state.LastFourNumbers = types.StringValue(formattedCardNumber(method))

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// formattedCardNumber formats the credit card ending digits as a 4-digit string.
func formattedCardNumber(method *account.PaymentMethod) string {
	return fmt.Sprintf("%04d", redis.IntValue(method.CreditCardEndsWith))
}

// filterPaymentMethods applies all filters to the list of payment methods.
func filterPaymentMethods(methods []*account.PaymentMethod, filters []func(method *account.PaymentMethod) bool) []*account.PaymentMethod {
	var filteredMethods []*account.PaymentMethod
	for _, method := range methods {
		if method == nil {
			continue
		}
		if filterPaymentMethod(method, filters) {
			filteredMethods = append(filteredMethods, method)
		}
	}
	return filteredMethods
}

// filterPaymentMethod checks if a single payment method passes all filters.
func filterPaymentMethod(method *account.PaymentMethod, filters []func(method *account.PaymentMethod) bool) bool {
	for _, f := range filters {
		if !f(method) {
			return false
		}
	}
	return true
}
