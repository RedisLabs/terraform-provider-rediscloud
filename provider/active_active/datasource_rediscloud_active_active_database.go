package active_active

import (
	"context"
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceRedisCloudActiveActiveDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "The Active Active Database data source allows access to the details of an existing AA database within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudActiveActiveDatabaseRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "Identifier of the subscription",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"db_id": {
				Description: "Identifier of the database created",
				Type:        schema.TypeInt,
				Computed:    true,
				Optional:    true,
			},
			"name": {
				Description: "A meaningful name to identify the database",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"memory_limit_in_gb": {
				Description: "(Deprecated) Maximum memory usage for this specific database",
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
			"support_oss_cluster_api": {
				Description: "Support Redis open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"external_endpoint_for_oss_cluster_api": {
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"enable_tls": {
				Description: "Use TLS for authentication.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"tls_certificate": {
				Description: "TLS certificate used for authentication.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_eviction": {
				Description: "Data eviction items policy",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"global_modules": {
				Description: "List of modules to enable on the database. This information is only used when creating a new database and any changes will be ignored after this.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"public_endpoint": {
				Description: "Region public and private endpoints to access the database",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"private_endpoint": {
				Description: "Region public and private endpoints to access the database",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"latest_backup_statuses": {
				Description: "Details about the last backups that took place across each region for this active-active database",
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"response": {
							Type:     schema.TypeSet,
							Computed: true,
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

func dataSourceRedisCloudActiveActiveDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	subId := d.Get("subscription_id").(int)

	var filters []func(db *databases.ActiveActiveDatabase) bool

	// Filter to active-active databases only (pro dbs come from the same endpoint)
	filters = append(filters, func(db *databases.ActiveActiveDatabase) bool {
		return redis.BoolValue(db.ActiveActiveRedis)
	})

	if v, ok := d.GetOk("db_id"); ok {
		filters = append(filters, func(db *databases.ActiveActiveDatabase) bool {
			return redis.IntValue(db.ID) == v.(int)
		})
	}

	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(db *databases.ActiveActiveDatabase) bool {
			return redis.StringValue(db.Name) == v.(string)
		})
	}

	list := api.Client.Database.ListActiveActive(ctx, subId)

	dbs, err := filterAADatabases(list, filters)
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
	db, err := api.Client.Database.GetActiveActive(ctx, subId, redis.IntValue(dbs[0].ID))
	if err != nil {
		return diag.FromErr(list.Err())
	}

	dbId := redis.IntValue(db.ID)
	d.SetId(fmt.Sprintf("%d/%d", subId, dbId))

	if err := d.Set("db_id", dbId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", redis.StringValue(db.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_limit_in_gb", redis.Float64(*db.CrdbDatabases[0].MemoryLimitInGB)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("dataset_size_in_gb", redis.Float64(*db.CrdbDatabases[0].DatasetSizeInGB)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("redis_version", redis.String(*db.RedisVersion)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("external_endpoint_for_oss_cluster_api", redis.BoolValue(db.UseExternalEndpointForOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("enable_tls", redis.BoolValue(db.CrdbDatabases[0].Security.EnableTls)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("data_eviction", redis.StringValue(db.DataEvictionPolicy)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("global_modules", flattenModulesToNames(db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	publicEndpointConfig := make(map[string]interface{})
	privateEndpointConfig := make(map[string]interface{})
	for _, regionDb := range db.CrdbDatabases {
		// Set the endpoints for the region
		publicEndpointConfig[redis.StringValue(regionDb.Region)] = redis.StringValue(regionDb.PublicEndpoint)
		privateEndpointConfig[redis.StringValue(regionDb.Region)] = redis.StringValue(regionDb.PrivateEndpoint)
	}

	if err := d.Set("public_endpoint", publicEndpointConfig); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("private_endpoint", privateEndpointConfig); err != nil {
		return diag.FromErr(err)
	}

	var parsedLatestBackupStatuses []map[string]interface{}
	for _, regionDb := range db.CrdbDatabases {
		region := redis.StringValue(regionDb.Region)
		latestBackupStatus, err := api.Client.LatestBackup.GetActiveActive(ctx, subId, dbId, region)
		if err != nil {
			// Forgive errors here, sometimes we just can't get a latest status
		} else {
			parsedLatestBackupStatus, err := utils.ParseLatestBackupStatus(latestBackupStatus)
			if err != nil {
				return diag.FromErr(err)
			}
			soloParsedLatestBackupStatus := parsedLatestBackupStatus[0]
			soloParsedLatestBackupStatus["region"] = region
			parsedLatestBackupStatuses = append(parsedLatestBackupStatuses, parsedLatestBackupStatus[0])
		}
	}
	if err := d.Set("latest_backup_statuses", parsedLatestBackupStatuses); err != nil {
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

	if err := utils.ReadTags(ctx, api, subId, dbId, d); err != nil {
		return diag.FromErr(err)
	}

	if dbTlsCertificate, err := getCertificateData(ctx, api, subId, dbId); err != nil {
		return diag.FromErr(err)
	} else if dbTlsCertificate != nil {
		if err := d.Set("tls_certificate", dbTlsCertificate.PublicCertificatePEMString); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func getCertificateData(ctx context.Context, api *utils.ApiClient, subId int, dbId int) (*databases.DatabaseCertificate, error) {
	dbTlsCertificate, err := api.Client.Database.GetCertificate(ctx, subId, dbId)

	if err != nil {
		return nil, err
	}

	if dbTlsCertificate == nil {
		return nil, fmt.Errorf("no certificate found for database %d", dbId)
	}
	return dbTlsCertificate, nil
}

func filterAADatabases(list *databases.ListActiveActiveDatabase, filters []func(db *databases.ActiveActiveDatabase) bool) ([]*databases.ActiveActiveDatabase, error) {
	var filtered []*databases.ActiveActiveDatabase
	for list.Next() {
		if filterAADatabase(list.Value(), filters) {
			filtered = append(filtered, list.Value())
		}
	}
	if list.Err() != nil {
		return nil, list.Err()
	}

	return filtered, nil
}

func filterAADatabase(db *databases.ActiveActiveDatabase, filters []func(db *databases.ActiveActiveDatabase) bool) bool {
	for _, filter := range filters {
		if !filter(db) {
			return false
		}
	}
	return true
}
