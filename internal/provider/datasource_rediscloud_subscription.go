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
		Description: "The Subscription data source allows access to the details of an existing subscription within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudSubscriptionRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the subscription to filter returned subscriptions",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
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
			"persistent_storage_encryption": {
				Description: "Encrypt data stored in persistent storage. Required for a GCP subscription",
				Type:        schema.TypeBool,
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
							Description: "Cloud networking details, per region (single region or multiple regions for Active-Active cluster only)",
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
													Description:      "Deployment CIDR mask",
													Type:             schema.TypeString,
													Computed:         true,
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
	if err := d.Set("cloud_provider", flattenCloudDetails(sub.CloudDetails, false)); err != nil {
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
