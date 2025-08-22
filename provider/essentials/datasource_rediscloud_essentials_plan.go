package essentials

import (
	"context"
	"strconv"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/fixed/plans"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceRedisCloudEssentialsPlan() *schema.Resource {
	return &schema.Resource{
		Description: "An Essentials subscription plan",
		ReadContext: dataSourceRedisCloudEssentialsPlanRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The plan's unique identifier",
				Type:        schema.TypeInt,
				Computed:    true,
				Optional:    true,
			},
			"name": {
				Description: "A convenient name for the plan. Not guaranteed to be unique, especially across provider/region",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"size": {
				Description: "The capacity of databases created in this plan",
				Type:        schema.TypeFloat,
				Computed:    true,
				Optional:    true,
			},
			"size_measurement_unit": {
				Description: "The units of 'size', usually 'MB' or 'GB'",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"subscription_id": {
				Description: "Filter plans by what is available for a given subscription",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"cloud_provider": {
				Description: "The cloud provider: 'AWS', 'GCP' or 'Azure'",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"region": {
				Description: "The region to place databases in, format and availability dependent on cloud_provider",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"region_id": {
				Description: "An internal, unique-across-cloud-providers id for database region",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"price": {
				Description: "The plan's cost",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"price_currency": {
				Description: "Self-explanatory",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"price_period": {
				Description: "Self-explanatory, usually 'Month'",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"maximum_databases": {
				Description: "Self-explanatory",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"maximum_throughput": {
				Description: "Self-explanatory",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"maximum_bandwidth_in_gb": {
				Description: "Self-explanatory",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"availability": {
				Description: "'No replication', 'Single-zone' or 'Multi-zone'",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"connections": {
				Description: "Self-explanatory",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cidr_allow_rules": {
				Description: "Self-explanatory",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"support_data_persistence": {
				Description: "Self-explanatory",
				Type:        schema.TypeBool,
				Computed:    true,
				Optional:    true,
			},
			"support_instant_and_daily_backups": {
				Description: "Self-explanatory",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"support_replication": {
				Description: "Self-explanatory",
				Type:        schema.TypeBool,
				Computed:    true,
				Optional:    true,
			},
			"support_clustering": {
				Description: "Self-explanatory",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"supported_alerts": {
				Description: "List of the type of alerts supported by databases in this plan",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"customer_support": {
				Description: "Level of customer support available e.g. 'Basic', 'Standard'",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceRedisCloudEssentialsPlanRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	api := meta.(*utils.ApiClient)

	list, err := getResourceList(ctx, d, api)

	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(plan *plans.GetPlanResponse) bool

	if v, ok := d.GetOk("id"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.IntValue(plan.ID) == v.(int)
		})
	}

	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Name) == v.(string)
		})
	}

	if v, ok := d.GetOk("size"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.Float64Value(plan.Size) == v.(float64)
		})
	}

	if v, ok := d.GetOk("size_measurement_unit"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.SizeMeasurementUnit) == v.(string)
		})
	}

	if v, ok := d.GetOk("cloud_provider"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Provider) == v.(string)
		})
	}

	if v, ok := d.GetOk("region"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Region) == v.(string)
		})
	}

	if v, ok := d.GetOk("availability"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Availability) == v.(string)
		})
	}

	if v, ok := d.GetOk("support_data_persistence"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.BoolValue(plan.SupportDataPersistence) == v.(bool)
		})
	}

	if v, ok := d.GetOk("support_replication"); ok {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.BoolValue(plan.SupportReplication) == v.(bool)
		})
	}

	list = filterPlans(list, filters)

	if len(list) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(list) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	plan := list[0]

	d.SetId(strconv.Itoa(redis.IntValue(plan.ID)))
	if err := d.Set("id", redis.IntValue(plan.ID)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", redis.StringValue(plan.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("size", redis.Float64Value(plan.Size)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("size_measurement_unit", redis.StringValue(plan.SizeMeasurementUnit)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cloud_provider", redis.StringValue(plan.Provider)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", redis.StringValue(plan.Region)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region_id", redis.IntValue(plan.RegionID)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("price", redis.IntValue(plan.Price)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("price_currency", redis.StringValue(plan.PriceCurrency)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("price_period", redis.StringValue(plan.PricePeriod)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("maximum_databases", redis.IntValue(plan.MaximumDatabases)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("maximum_throughput", redis.IntValue(plan.MaximumThroughput)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("maximum_bandwidth_in_gb", redis.IntValue(plan.MaximumBandwidthGB)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("availability", redis.StringValue(plan.Availability)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("connections", redis.StringValue(plan.Connections)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cidr_allow_rules", redis.IntValue(plan.CidrAllowRules)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_data_persistence", redis.BoolValue(plan.SupportDataPersistence)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_instant_and_daily_backups", redis.BoolValue(plan.SupportInstantAndDailyBackups)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_replication", redis.BoolValue(plan.SupportReplication)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_clustering", redis.BoolValue(plan.SupportClustering)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("supported_alerts", redis.StringSliceValue(plan.SupportedAlerts...)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("customer_support", redis.StringValue(plan.CustomerSupport)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func getResourceList(ctx context.Context, d *schema.ResourceData, api *utils.ApiClient) ([]*plans.GetPlanResponse, error) {
	var list []*plans.GetPlanResponse
	var err error

	if id, ok := d.GetOk("subscription_id"); ok {
		list, err = api.Client.FixedPlanSubscriptions.List(ctx, id.(int))
	} else if provider, ok := d.GetOk("cloud_provider"); ok {
		list, err = api.Client.FixedPlans.ListWithProvider(ctx, strings.ToUpper(provider.(string)))
	} else {
		list, err = api.Client.FixedPlans.List(ctx)
	}

	return list, err
}

func filterPlans(allPlans []*plans.GetPlanResponse, filters []func(plan *plans.GetPlanResponse) bool) []*plans.GetPlanResponse {
	var filtered []*plans.GetPlanResponse
	for _, candidatePlan := range allPlans {
		if filterPlan(candidatePlan, filters) {
			filtered = append(filtered, candidatePlan)
		}
	}

	return filtered
}

func filterPlan(plan *plans.GetPlanResponse, filters []func(plan *plans.GetPlanResponse) bool) bool {
	for _, f := range filters {
		if !f(plan) {
			return false
		}
	}
	return true
}
