package pro

import (
	"context"
	"fmt"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"regexp"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func DataSourceRedisCloudProDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "The Pro Database data source allows access to the details of an existing database within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudProDatabaseRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "ID of the subscription that the database belongs to",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				Required:         true,
			},
			"db_id": {
				Description: "The id of the database to filter returned databases",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
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
			"dataset_size_in_gb": {
				Description: "Maximum amount of data in the dataset for this specific database in GB",
				Type:        schema.TypeFloat,
				Computed:    true,
			},
			"redis_version": {
				Description: "The redis version of the database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"query_performance_factor": {
				Description: "Query performance factor for this specific database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"support_oss_cluster_api": {
				Description: "Supports the Redis open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"resp_version": {
				Description: "The database's RESP version",
				Type:        schema.TypeString,
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
			"enable_default_user": {
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"latest_backup_status": {
				Description: "Details about the last backup that took place for this database",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"response": {
							Computed: true,
							Type:     schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"status": {
										Description: "The status of the last backup operation",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"last_backup_time": {
										Description: "When the last backup operation occurred",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"failure_reason": {
										Description: "If a failure, why the backup operation failed",
										Computed:    true,
										Type:        schema.TypeString,
									},
								},
							},
						},
						"error": {
							Computed: true,
							Type:     schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Description: "The type of error encountered while looking up the status of the last backup",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"description": {
										Description: "A description of the error encountered while looking up the status of the last backup",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"status": {
										Description: "Any particular HTTP status code associated with the erroneous status check",
										Computed:    true,
										Type:        schema.TypeString,
									},
								},
							},
						},
					},
				},
			},
			"latest_import_status": {
				Description: "Details about the last import that took place for this active-active database",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"response": {
							Computed: true,
							Type:     schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"status": {
										Description: "The status of the last import operation",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"last_import_time": {
										Description: "When the last import operation occurred",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"failure_reason": {
										Description: "If a failure, why the import operation failed",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"failure_reason_params": {
										Description: "Parameters of the failure, if appropriate",
										Computed:    true,
										Type:        schema.TypeList,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Description: "",
													Computed:    true,
													Type:        schema.TypeString,
												},
												"value": {
													Description: "",
													Computed:    true,
													Type:        schema.TypeString,
												},
											},
										},
									},
								},
							},
						},
						"error": {
							Computed: true,
							Type:     schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Description: "The type of error encountered while looking up the status of the last import",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"description": {
										Description: "A description of the error encountered while looking up the status of the last import",
										Computed:    true,
										Type:        schema.TypeString,
									},
									"status": {
										Description: "Any particular HTTP status code associated with the erroneous status check",
										Computed:    true,
										Type:        schema.TypeString,
									},
								},
							},
						},
					},
				},
			},
			"tags": {
				Description: "Tags for database management",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
	}
}

func dataSourceRedisCloudProDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(db *databases.Database) bool

	// Filter to pro databases only (active-active dbs come from the same endpoint)
	filters = append(filters, func(db *databases.Database) bool {
		return !redis.BoolValue(db.ActiveActiveRedis)
	})

	if v, ok := d.GetOk("db_id"); ok {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.IntValue(db.ID) == v.(int)
		})
	}
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

	list := api.Client.Database.List(ctx, subId)
	dbs, err := filterProDatabases(list, filters)
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
	dbId := redis.IntValue(dbs[0].ID)
	db, err := api.Client.Database.Get(ctx, subId, dbId)
	if err != nil {
		return diag.FromErr(list.Err())
	}

	d.SetId(fmt.Sprintf("%d/%d", subId, dbId))

	if err := d.Set("db_id", dbId); err != nil {
		return diag.FromErr(err)
	}
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
	if err := d.Set("dataset_size_in_gb", redis.Float64Value(db.DatasetSizeInGB)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resp_version", redis.StringValue(db.RespVersion)); err != nil {
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

	if db.ThroughputMeasurement != nil {
		if err := d.Set("throughput_measurement_by", redis.StringValue(db.ThroughputMeasurement.By)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("throughput_measurement_value", redis.IntValue(db.ThroughputMeasurement.Value)); err != nil {
			return diag.FromErr(err)
		}
	}

	if db.Security != nil {
		if v := redis.StringValue(db.Security.Password); v != "" {
			if err := d.Set("password", v); err != nil {
				return diag.FromErr(err)
			}
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
	if err := d.Set("alert", FlattenAlerts(db.Alerts)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("module", FlattenModules(db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	if db.Clustering != nil {
		if err := d.Set("hashing_policy", FlattenRegexRules(db.Clustering.RegexRules)); err != nil {
			return diag.FromErr(err)
		}
	}
	if db.Security != nil {
		if err := d.Set("enable_default_user", redis.BoolValue(db.Security.EnableDefaultUser)); err != nil {
			return diag.FromErr(err)
		}
	}

	var parsedLatestBackupStatus []map[string]interface{}
	latestBackupStatus, err := api.Client.LatestBackup.Get(ctx, subId, dbId)
	if err != nil {
		// Forgive errors here, sometimes we just can't get a latest status
	} else {
		parsedLatestBackupStatus, err = utils.ParseLatestBackupStatus(latestBackupStatus)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("latest_backup_status", parsedLatestBackupStatus); err != nil {
		return diag.FromErr(err)
	}

	var parsedLatestImportStatus []map[string]interface{}
	latestImportStatus, err := api.Client.LatestImport.Get(ctx, subId, dbId)
	if err != nil {
		// Forgive errors here, sometimes we just can't get a latest status
	} else {
		parsedLatestImportStatus, err = utils.ParseLatestImportStatus(latestImportStatus)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("latest_import_status", parsedLatestImportStatus); err != nil {
		return diag.FromErr(err)
	}

	if err := readTags(ctx, api, subId, dbId, d); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("query_performance_factor", redis.String(*db.QueryPerformanceFactor)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("redis_version", redis.String(*db.RedisVersion)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterProDatabases(list *databases.ListDatabase, filters []func(db *databases.Database) bool) ([]*databases.Database, error) {
	var filtered []*databases.Database
	for list.Next() {
		if filterProDatabase(list.Value(), filters) {
			filtered = append(filtered, list.Value())
		}
	}
	if list.Err() != nil {
		return nil, list.Err()
	}

	return filtered, nil
}

func filterProDatabase(db *databases.Database, filters []func(db *databases.Database) bool) bool {
	for _, filter := range filters {
		if !filter(db) {
			return false
		}
	}
	return true
}
