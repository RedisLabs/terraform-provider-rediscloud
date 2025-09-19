package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceRedisCloudActiveActiveSubscription() *schema.Resource {
	return &schema.Resource{
		Description: "The Active Active Subscription data source allows access to the details of an existing AA subscription within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudActiveActiveSubscriptionRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"payment_method": {
				Description: "Payment method for the requested subscription. If credit card is specified, the payment method id must be defined. This information is only used when creating a new subscription and any changes will be ignored after this.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"payment_method_id": {
				Description: "A valid payment method pre-defined in the current account",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"number_of_databases": {
				Description: "The number of databases that are linked to this subscription",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"status": {
				Description: "Current status of the subscription",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cloud_provider": {
				Description: "A cloud provider string either GCP or AWS",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"maintenance_windows": {
				Description: "Details about the subscription's maintenance window specification",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Description: "Either automatic (Redis specified) or manual (User specified)",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"window": {
							Description: "A list of maintenance windows for manual-mode",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"start_hour": {
										Description: "What hour in the day (0-23) the window opens",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"duration_in_hours": {
										Description: "How long the window is open",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"days": {
										Description: "A list of weekdays on which the window is open ('Monday', 'Tuesday' etc)",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
			"pricing": {
				Description: "Pricing details totalled over this Subscription",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database_name": {
							Description: "The database this pricing entry applies to",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"type": {
							Description: "The type of cost e.g. 'Shards'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"type_details": {
							Description: "Further detail e.g. 'micro'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"quantity": {
							Description: "Self-explanatory",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"quantity_measurement": {
							Description: "Self-explanatory",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"price_per_unit": {
							Description: "Self-explanatory",
							Type:        schema.TypeFloat,
							Computed:    true,
						},
						"price_currency": {
							Description: "Self-explanatory e.g. 'USD'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"price_period": {
							Description: "Self-explanatory e.g. 'hour'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"region": {
							Description: "Self-explanatory, if the cost is associated with a particular region",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudActiveActiveSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subs, err := api.Client.Subscription.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *subscriptions.Subscription) bool

	// Filter to AA subscriptions only (pro subs come from the same endpoint)
	filters = append(filters, func(sub *subscriptions.Subscription) bool {
		return redis.StringValue(sub.DeploymentType) == "active-active"
	})

	if name, ok := d.GetOk("name"); ok {
		filters = append(filters, func(sub *subscriptions.Subscription) bool {
			return redis.StringValue(sub.Name) == name
		})
	}

	subs = filterSubscriptions(subs, filters)

	if len(subs) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(subs) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	sub := subs[0]

	paymentMethodID := ""
	if sub.PaymentMethodID != nil {
		paymentMethodID = strconv.Itoa(redis.IntValue(sub.PaymentMethodID))
	}

	if err := d.Set("name", redis.StringValue(sub.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method_id", paymentMethodID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method", sub.PaymentMethod); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("number_of_databases", redis.IntValue(sub.NumberOfDatabases)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(sub.Status)); err != nil {
		return diag.FromErr(err)
	}

	cloudDetails := sub.CloudDetails
	if len(cloudDetails) == 0 {
		// Clearing the value - a subscription with 0 databases will have no CloudDetail blocks
		if err := d.Set("cloud_provider", nil); err != nil {
			return diag.FromErr(err)
		}
	} else {
		cloudProvider := cloudDetails[0].Provider
		if err := d.Set("cloud_provider", cloudProvider); err != nil {
			return diag.FromErr(err)
		}
	}

	subId := redis.IntValue(sub.ID)

	m, err := api.Client.Maintenance.Get(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("maintenance_windows", flattenMaintenance(m)); err != nil {
		return diag.FromErr(err)
	}

	pricingList, err := api.Client.Pricing.List(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("pricing", flattenPricing(pricingList)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	return diags
}
