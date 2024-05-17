package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceRedisCloudEssentialsDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "The Essentials Database data source allows access to the details of an existing database within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudEssentialsDatabaseRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "Identifier of the essentials subscription",
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
			"protocol": {
				Description: "The protocol that will be used to access the database, (either ‘redis’, 'memcached’ or 'stack')",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cloud_provider": {
				Description: "The Cloud Provider hosting this database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"region": {
				Description: "The region within the Cloud Provider where this database is hosted",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"redis_version_compliance": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"resp_version": {
				Description: "RESP version must be compatible with Redis version.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_persistence": {
				Description: "Rate of database data persistence (in persistent storage).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_eviction": {
				Description: "The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"replication": {
				Description: "Database's replication",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"activated_on": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"periodic_backup_path": {
				Description: "Path that will be used to store database backup files",
				Type:        schema.TypeString,
				Computed:    true,
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
			"source_ips": {
				Description: "Set of CIDR addresses to allow access to the database",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
				},
			},
			"replica": {
				Description: "Details of database replication",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sync_source": {
							Description: "A source database to replicate here",
							Type:        schema.TypeSet,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"endpoint": {
										Description: "A Redis URI (sample format: 'redis://user:password@host:port)'. If the URI provided is Redis Cloud instance, only host and port should be provided (using the format: ['redis://endpoint1:6379', 'redis://endpoint2:6380'])",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"encryption": {
										Description: "Defines if encryption should be used to connect to the sync source. If not provided and if the source is a Redis Cloud instance, it will automatically detect if the source uses encryption",
										Type:        schema.TypeBool,
										Computed:    true,
									},
									"server_cert": {
										Description: "TLS/SSL certificate chain of the sync source. If left null and if the source is a Redis Cloud instance, it will automatically detect the certificate to use",
										Type:        schema.TypeString,
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
			"client_tls_certificates": {
				Description: "",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"password": {
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
			},
			"enable_default_user": {
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"alert": {
				Description: "Set of alerts to enable on the database",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Alert name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"value": {
							Description: "Alert value",
							Type:        schema.TypeInt,
							Computed:    true,
						},
					},
				},
			},
			"modules": {
				Description: "Modules to be provisioned in the database",
				Type:        schema.TypeSet,
				// In TF <0.12 List of objects is not supported, so we need to opt-in to use this old behaviour.
				ConfigMode: schema.SchemaConfigModeAttr,
				Computed:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the module to enable",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"latest_backup_status": {
				Description: "Details about the last backup that took place for this database",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"response": {
							Description: "JSON-style details about the last backup",
							Computed:    true,
							Type:        schema.TypeString,
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
				Description: "Details about the last import that took place for this database",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"response": {
							Description: "JSON-style details about the last import",
							Computed:    true,
							Type:        schema.TypeString,
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
		},
	}
}

func dataSourceRedisCloudEssentialsDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subId := d.Get("subscription_id").(int)

	var filters []func(db *fixedDatabases.FixedDatabase) bool

	if v, ok := d.GetOk("db_id"); ok {
		filters = append(filters, func(db *fixedDatabases.FixedDatabase) bool {
			return redis.IntValue(db.DatabaseId) == v.(int)
		})
	}

	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(db *fixedDatabases.FixedDatabase) bool {
			return redis.StringValue(db.Name) == v.(string)
		})
	}

	list := api.client.FixedDatabases.List(ctx, subId)
	dbs, err := filterFixedDatabases(list, filters)
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
	db, err := api.client.FixedDatabases.Get(ctx, subId, redis.IntValue(dbs[0].DatabaseId))
	if err != nil {
		return diag.FromErr(list.Err())
	}

	databaseId := redis.IntValue(db.DatabaseId)

	d.SetId(buildResourceId(subId, databaseId))

	if err := d.Set("db_id", redis.IntValue(db.DatabaseId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", redis.StringValue(db.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("protocol", redis.StringValue(db.Protocol)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cloud_provider", redis.StringValue(db.Provider)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", redis.StringValue(db.Region)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("redis_version_compliance", redis.StringValue(db.RedisVersionCompliance)); err != nil {
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
	if err := d.Set("activated_on", db.ActivatedOn.String()); err != nil {
		return diag.FromErr(err)
	}
	// Periodic Backup Path is not returned by the API directly, it might be in the backup object
	if db.Backup != nil && redis.BoolValue(db.Backup.Enabled) {
		if err := d.Set("periodic_backup_path", redis.StringValue(db.Backup.Destination)); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("public_endpoint", redis.StringValue(db.PublicEndpoint)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("private_endpoint", redis.StringValue(db.PrivateEndpoint)); err != nil {
		return diag.FromErr(err)
	}

	var sourceIPs []string
	if !(len(db.Security.SourceIPs) == 1 && redis.StringValue(db.Security.SourceIPs[0]) == "0.0.0.0/0") {
		// The API handles an empty list as ["0.0.0.0/0"] but need to be careful to match the input to avoid Terraform detecting drift
		sourceIPs = redis.StringSliceValue(db.Security.SourceIPs...)
	}
	if err := d.Set("source_ips", sourceIPs); err != nil {
		return diag.FromErr(err)
	}

	if db.Replica != nil {
		if err := d.Set("replica", writeReplica(*db.Replica)); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("replica", nil); err != nil {
			return diag.FromErr(err)
		}
	}

	// Client TLS Certificates are not returned

	password := d.Get("password").(string)
	if redis.StringValue(db.Protocol) == "redis" {
		// Only db with the "redis" protocol returns the password.
		password = redis.StringValue(db.Security.Password)
	}
	if err := d.Set("password", password); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("enable_default_user", redis.Bool(*db.Security.EnableDefaultUser)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("alert", flattenAlerts(*db.Alerts)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("modules", flattenModules(*db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	var parsedLatestBackupStatus []map[string]interface{}
	latestBackupStatus, err := api.client.LatestBackup.GetFixed(ctx, subId, databaseId)
	if err != nil {
		// Forgive errors here, sometimes we just can't get a latest status
	} else {
		parsedLatestBackupStatus, err = parseLatestBackupStatus(latestBackupStatus)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("latest_backup_status", parsedLatestBackupStatus); err != nil {
		return diag.FromErr(err)
	}

	var parsedLatestImportStatus []map[string]interface{}
	latestImportStatus, err := api.client.LatestImport.GetFixed(ctx, subId, databaseId)
	if err != nil {
		// Forgive errors here, sometimes we just can't get a latest status
	} else {
		parsedLatestImportStatus, err = parseLatestImportStatus(latestImportStatus)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("latest_import_status", parsedLatestImportStatus); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterFixedDatabases(list *fixedDatabases.ListFixedDatabase, filters []func(db *fixedDatabases.FixedDatabase) bool) ([]*fixedDatabases.FixedDatabase, error) {
	var filtered []*fixedDatabases.FixedDatabase
	for list.Next() {
		if filterFixedDatabase(list.Value(), filters) {
			filtered = append(filtered, list.Value())
		}
	}
	if list.Err() != nil {
		return nil, list.Err()
	}

	return filtered, nil
}

func filterFixedDatabase(db *fixedDatabases.FixedDatabase, filters []func(db *fixedDatabases.FixedDatabase) bool) bool {
	for _, filter := range filters {
		if !filter(db) {
			return false
		}
	}
	return true
}
