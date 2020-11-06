package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
	"strconv"
	"time"
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
			"exclude_expired": {
				Optional: true,
				Default:  true,
				Type:     schema.TypeBool,
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

	var filters []func(method *account.PaymentMethod) bool

	if exclude, ok := d.GetOk("exclude_expired"); ok && exclude.(bool) {
		now := time.Now()
		filters = append(filters, func(method *account.PaymentMethod) bool {
			if redis.IntValue(method.ExpirationYear) < now.Year() {
				// Expiration year is last year, so it must already have expired and no point checking the month
				return false
			}

			if redis.IntValue(method.ExpirationYear) > now.Year() {
				// Expiration year is next year, so it cannot have expired and no point checking the month
				return true
			}

			// Expiration year is this year, so we do have to check the month
			if redis.IntValue(method.ExpirationMonth) < int(now.Month()) {
				return false
			}

			return true
		})
	}
	if card, ok := d.GetOk("card_type"); ok {
		filters = append(filters, func(method *account.PaymentMethod) bool {
			return redis.StringValue(method.Type) == card
		})
	}
	if fourNumbers, ok := d.GetOk("last_four_numbers"); ok {
		filters = append(filters, func(method *account.PaymentMethod) bool {
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

	d.SetId(strconv.Itoa(redis.IntValue(method.ID)))
	d.Set("card_type", method.Type)
	d.Set("last_four_numbers", formattedCardNumber(method))

	return diags
}

func formattedCardNumber(method *account.PaymentMethod) string {
	return fmt.Sprintf("%04d", redis.IntValue(method.CreditCardEndsWith))
}

func filterPaymentMethods(methods []*account.PaymentMethod, filters []func(method *account.PaymentMethod) bool) []*account.PaymentMethod {
	var filteredMethods []*account.PaymentMethod
	for _, method := range methods {
		if filterPaymentMethod(method, filters) {
			filteredMethods = append(filteredMethods, method)
		}
	}

	return filteredMethods
}

func filterPaymentMethod(method *account.PaymentMethod, filters []func(method *account.PaymentMethod) bool) bool {
	for _, f := range filters {
		if !f(method) {
			return false
		}
	}
	return true
}
