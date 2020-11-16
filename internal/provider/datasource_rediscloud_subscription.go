package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceRedisCloudSubscription() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRedisCloudSubscriptionRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"payment_method_id": {
				Type:             schema.TypeString,
				Computed: true,
			},
			"memory_storage": {
				Type:             schema.TypeString,
				Computed: true,
			},
			"persistent_storage_encryption": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"number_of_databases": {
				Type: schema.TypeInt,
				Computed: true,
			},
			"status": {
				Type: schema.TypeString,
				Computed: true,
			},
			"cloud_provider": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Type:             schema.TypeString,
							Computed: true,
						},
						"cloud_account_id": {
							Type:             schema.TypeString,
							Computed: true,
						},
						"region": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"multiple_availability_zones": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"preferred_availability_zones": {
										Type: schema.TypeList,
										Computed: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"networking_deployment_cidr": {
										Type: schema.TypeString,
										Computed: true,
									},
									"networking_vpc_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"networking_subnet_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subs, err := api.client.Subscription.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *subscriptions.Subscription) bool

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

	if err := d.Set("name", redis.StringValue(sub.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method_id", strconv.Itoa(redis.IntValue(sub.PaymentMethodID))); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_storage", redis.StringValue(sub.MemoryStorage)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("persistent_storage_encryption", redis.BoolValue(sub.StorageEncryption)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("number_of_databases", redis.IntValue(sub.NumberOfDatabases)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cloud_provider", flattenCloudDetails(sub.CloudDetails)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(sub.Status)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(redis.IntValue(sub.ID)))

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

