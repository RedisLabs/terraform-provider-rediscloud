package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
	"strconv"
)

func dataSourceRedisCloudPaymentMethod() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRedisCloudPaymentMethodRead,

		Schema: map[string]*schema.Schema{
			"card_type": {
				Optional: true,
				Computed: true,
				Type:     schema.TypeString,
			},
			"last_four_numbers": {
				Optional: true,
				Computed: true,
				Type:     schema.TypeString,

				ValidateDiagFunc: toDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d{4}$"), "")),
			},
		},
	}
}

func dataSourceRedisCloudPaymentMethodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*apiClient)

	methods, err := client.client.Account.ListPaymentMethods(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method account.PaymentMethod) bool

	if card, ok := d.GetOk("card_type"); ok {
		filters = append(filters, func(method account.PaymentMethod) bool {
			return method.Type == card
		})
	}
	if fourNumbers, ok := d.GetOk("last_four_numbers"); ok {
		filters = append(filters, func(method account.PaymentMethod) bool {
			return formattedCardNumber(method) == fourNumbers
		})
	}

	methods = filterPaymentMethods(methods, filters)

	if len(methods) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(methods) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	method := methods[0]

	d.SetId(strconv.Itoa(method.Id))
	d.Set("card_type", method.Type)
	d.Set("last_four_numbers", formattedCardNumber(method))

	return diags
}

func formattedCardNumber(method account.PaymentMethod) string {
	return fmt.Sprintf("%04d", method.CreditCardEndsWith)
}

func filterPaymentMethods(methods []account.PaymentMethod, filters []func(method account.PaymentMethod) bool) []account.PaymentMethod {
	var filteredMethods []account.PaymentMethod
	for _, method := range methods {
		if filterPaymentMethod(method, filters) {
			filteredMethods = append(filteredMethods, method)
		}
	}

	return filteredMethods
}

func filterPaymentMethod(method account.PaymentMethod, filters []func(method account.PaymentMethod) bool) bool {
	for _, f := range filters {
		if !f(method) {
			return false
		}
	}
	return true
}
