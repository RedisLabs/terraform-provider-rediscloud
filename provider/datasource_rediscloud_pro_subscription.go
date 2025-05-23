package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceRedisCloudProSubscription() *schema.Resource {
	return &schema.Resource{
		Description: "The Pro Subscription data source allows access to the details of an existing pro subscription within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudProSubscriptionRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the subscription to filter returned subscriptions",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"payment_method": {
				Description: "Payment method for the requested subscription. Either 'credit-card' or 'marketplace'",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"payment_method_id": {
				Description: "A valid payment method pre-defined in the current account",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"memory_storage": {
				Description: "Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’",
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
				Description: "A cloud provider object",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Description: "The cloud provider to use with the subscription, (either `AWS` or `GCP`)",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"cloud_account_id": {
							Description: "Cloud account identifier, (A Cloud Account Id = 1 implies using Redis Labs internal cloud account)",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"region": {
							Description: "Cloud networking details, per region",
							Type:        schema.TypeSet,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Description: "Deployment region as defined by cloud provider",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"multiple_availability_zones": {
										Description: "Support deployment on multiple availability zones within the selected region",
										Type:        schema.TypeBool,
										Computed:    true,
									},
									"preferred_availability_zones": {
										Description: "List of availability zones used",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"networking_vpc_id": {
										Description: "The ID of the VPC where the Redis Cloud subscription is deployed",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"networks": {
										Description: "List of networks used",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"networking_subnet_id": {
													Description: "The subnet that the subscription deploys into",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"networking_deployment_cidr": {
													Description: "Deployment CIDR mask",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"networking_vpc_id": {
													Description: "Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified)",
													Type:        schema.TypeString,
													Computed:    true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
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
										Description: "What hour in the day (0-23) may maintenance start",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"duration_in_hours": {
										Description: "How long maintenance may take",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"days": {
										Description: "A list of days on which the window is open ('Monday', 'Tuesday' etc)",
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

func dataSourceRedisCloudProSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subs, err := api.client.Subscription.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *subscriptions.Subscription) bool

	// Filter to pro subscriptions only (active-active subs come from the same endpoint)
	filters = append(filters, func(sub *subscriptions.Subscription) bool {
		return redis.StringValue(sub.DeploymentType) != "active-active"
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
	if err := d.Set("memory_storage", redis.StringValue(sub.MemoryStorage)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("number_of_databases", redis.IntValue(sub.NumberOfDatabases)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cloud_provider", flattenCloudDetails(sub.CloudDetails, false)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(sub.Status)); err != nil {
		return diag.FromErr(err)
	}

	subId := redis.IntValue(sub.ID)

	m, err := api.client.Maintenance.Get(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("maintenance_windows", flattenMaintenance(m)); err != nil {
		return diag.FromErr(err)
	}

	pricingList, err := api.client.Pricing.List(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("pricing", flattenPricing(pricingList)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	return diags
}

func filterSubscriptions(subs []*subscriptions.Subscription, filters []func(sub *subscriptions.Subscription) bool) []*subscriptions.Subscription {
	var filteredSubs []*subscriptions.Subscription
	for _, sub := range subs {
		if filterSub(sub, filters) {
			filteredSubs = append(filteredSubs, sub)
		}
	}

	return filteredSubs
}

func filterSub(method *subscriptions.Subscription, filters []func(method *subscriptions.Subscription) bool) bool {
	for _, f := range filters {
		if !f(method) {
			return false
		}
	}
	return true
}
