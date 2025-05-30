package provider

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fs "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRedisCloudEssentialsSubscription() *schema.Resource {
	return &schema.Resource{
		Description: "Watches an Essentials Subscription within your Redis Enterprise Cloud Account.",
		ReadContext: dataSourceRedisCloudEssentialsSubscriptionRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The subscription's id",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"name": {
				Description: "A meaningful name to identify the subscription",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"status": {
				Description: "The status of this subscription",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"plan_id": {
				Description: "The identifier of the plan to template the subscription",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"payment_method_id": {
				Description: "The identifier of the method which will be charged for this subscription. Not required for free plans",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"creation_date": {
				Description: "The date/time this subscription was created",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceRedisCloudEssentialsSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subs, err := api.client.FixedSubscriptions.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(method *fs.FixedSubscriptionResponse) bool

	if id, ok := d.GetOk("id"); ok {
		filters = append(filters, func(sub *fs.FixedSubscriptionResponse) bool {
			return redis.IntValue(sub.ID) == id
		})
	}

	if name, ok := d.GetOk("name"); ok {
		filters = append(filters, func(sub *fs.FixedSubscriptionResponse) bool {
			return redis.StringValue(sub.Name) == name
		})
	}

	subs = filterFixedSubscriptions(subs, filters)

	if len(subs) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(subs) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	sub := subs[0]

	d.SetId(strconv.Itoa(redis.IntValue(sub.ID)))
	if err := d.Set("id", redis.IntValue(sub.ID)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", redis.StringValue(sub.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(sub.Status)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("plan_id", redis.IntValue(sub.PlanId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method_id", redis.IntValue(sub.PaymentMethodID)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("creation_date", redis.TimeValue(sub.CreationDate).String()); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterFixedSubscriptions(subs []*fs.FixedSubscriptionResponse, filters []func(sub *fs.FixedSubscriptionResponse) bool) []*fs.FixedSubscriptionResponse {
	var filteredSubs []*fs.FixedSubscriptionResponse
	for _, sub := range subs {
		if filterFixedSub(sub, filters) {
			filteredSubs = append(filteredSubs, sub)
		}
	}

	return filteredSubs
}

func filterFixedSub(method *fs.FixedSubscriptionResponse, filters []func(method *fs.FixedSubscriptionResponse) bool) bool {
	for _, f := range filters {
		if !f(method) {
			return false
		}
	}
	return true
}
