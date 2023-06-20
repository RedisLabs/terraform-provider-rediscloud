package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
	"strconv"
)

func dataSourceRedisCloudDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "The Database data source allows access to the details of an existing database within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudDatabaseRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "ID of the subscription that the database belongs to",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				Required:         true,
			},
			"name": {
				Description: "The name of the database to filter returned databases",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"protocol": {
				Description: "The protocol of the database to filter returned databases",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"region": {
				Description: "The region of the database to filter returned databases",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"memory_limit_in_gb": {
				Description: "The maximum memory usage for the database",
				Type:        schema.TypeFloat,
				Computed:    true,
			},
			"support_oss_cluster_api": {
				Description: "Supports the Redis open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"data_persistence": {
				Description: "The rate of database data persistence (in persistent storage)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_eviction": {
				Description: "The data items eviction method",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"replication": {
				Description: "Database replication",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"throughput_measurement_by": {
				Description: "The throughput measurement method",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"throughput_measurement_value": {
				Description: "The throughput value",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"password": {
				Description: "The password used to access the database - not present on `memcached` protocol databases",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			"public_endpoint": {
				Description: "Public endpoint to access the database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"private_endpoint": {
				Description: "Private endpoint to access the database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"replica_of": {
				Description: "The set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"alert": {
				Description: "Set of alerts to enable on the database",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The alert name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"value": {
							Description: "The alert value",
							Type:        schema.TypeInt,
							Computed:    true,
						},
					},
				},
			},
			"module": {
				Description: "Redis modules that have been enabled on the database",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the module",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"hashing_policy": {
				Description: "The list of regular expression rules the database is sharded by",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceRedisCloudDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(db *databases.Database) bool
	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Name) == v.(string)
		})
	}
	if v, ok := d.GetOk("protocol"); ok {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Protocol) == v.(string)
		})
	}
	if v, ok := d.GetOk("region"); ok {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Region) == v.(string)
		})
	}

	list := api.client.Database.List(ctx, subId)
	dbs, err := filterDatabases(list, filters)
	if err != nil {
		return diag.FromErr(list.Err())
	}

	if len(dbs) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(dbs) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	// Some attributes are only returned when retrieving a single database
	db, err := api.client.Database.Get(ctx, subId, redis.IntValue(dbs[0].ID))
	if err != nil {
		return diag.FromErr(list.Err())
	}

	d.SetId(fmt.Sprintf("%d/%d", subId, redis.IntValue(db.ID)))

	if err := d.Set("name", redis.StringValue(db.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("protocol", redis.StringValue(db.Protocol)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", redis.StringValue(db.Region)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_limit_in_gb", redis.Float64Value(db.MemoryLimitInGB)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("data_persistence", redis.StringValue(db.DataPersistence)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("data_eviction", redis.StringValue(db.DataEvictionPolicy)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("replication", redis.BoolValue(db.Replication)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("throughput_measurement_by", redis.StringValue(db.ThroughputMeasurement.By)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("throughput_measurement_value", redis.IntValue(db.ThroughputMeasurement.Value)); err != nil {
		return diag.FromErr(err)
	}
	if v := redis.StringValue(db.Security.Password); v != "" {
		if err := d.Set("password", v); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("public_endpoint", redis.StringValue(db.PublicEndpoint)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("private_endpoint", redis.StringValue(db.PrivateEndpoint)); err != nil {
		return diag.FromErr(err)
	}
	if db.ReplicaOf != nil {
		if err := d.Set("replica_of", redis.StringSliceValue(db.ReplicaOf.Endpoints...)); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("alert", flattenAlerts(db.Alerts)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("module", flattenModules(db.Modules)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("hashing_policy", flattenRegexRules(db.Clustering.RegexRules)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterDatabases(list *databases.ListDatabase, filters []func(db *databases.Database) bool) ([]*databases.Database, error) {
	var filtered []*databases.Database
	for list.Next() {
		if filterDatabase(list.Value(), filters) {
			filtered = append(filtered, list.Value())
		}
	}
	if list.Err() != nil {
		return nil, list.Err()
	}

	return filtered, nil
}

func filterDatabase(db *databases.Database, filters []func(db *databases.Database) bool) bool {
	for _, filter := range filters {
		if !filter(db) {
			return false
		}
	}
	return true
}
