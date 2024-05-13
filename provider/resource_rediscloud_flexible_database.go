package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_backups"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_imports"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudFlexibleDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates database resource within a flexible subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudFlexibleDatabaseCreate,
		ReadContext:   resourceRedisCloudFlexibleDatabaseRead,
		UpdateContext: resourceRedisCloudFlexibleDatabaseUpdate,
		DeleteContext: resourceRedisCloudFlexibleDatabaseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				subId, dbId, err := toDatabaseId(d.Id())
				if err != nil {
					return nil, err
				}
				if err := d.Set("subscription_id", subId); err != nil {
					return nil, err
				}
				if err := d.Set("db_id", dbId); err != nil {
					return nil, err
				}
				d.SetId(buildResourceId(subId, dbId))
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		CustomizeDiff: remoteBackupIntervalSetCorrectly("remote_backup"),

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "Identifier of the flexible subscription",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"db_id": {
				Description: "Identifier of the database created",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"name": {
				Description:      "A meaningful name to identify the database",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 40)),
			},
			"protocol": {
				Description:      "The protocol that will be used to access the database, (either ‘redis’ or 'memcached’) ",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.ProtocolValues(), false)),
				Optional:         true,
				ForceNew:         true,
				Default:          "redis",
			},
			"memory_limit_in_gb": {
				Description: "Maximum memory usage for this specific database",
				Type:        schema.TypeFloat,
				Required:    true,
			},
			"support_oss_cluster_api": {
				Description: "Support Redis open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"resp_version": {
				Description: "The database's RESP version",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"external_endpoint_for_oss_cluster_api": {
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"data_persistence": {
				Description: "Rate of database data persistence (in persistent storage)",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "none",
			},
			"data_eviction": {
				Description: "(Optional) The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "volatile-lru",
			},
			"replication": {
				Description: "Databases replication",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"throughput_measurement_by": {
				Description:      "Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"number-of-shards", "operations-per-second"}, false)),
			},
			"throughput_measurement_value": {
				Description: "Throughput value (as applies to selected measurement method)",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"average_item_size_in_bytes": {
				Description: "Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database",
				Type:        schema.TypeInt,
				Optional:    true,
				// Setting default to 0 so that the hash func produces the same hash when this field is not
				// specified. SDK's catch-all issue around this: https://github.com/hashicorp/terraform-plugin-sdk/issues/261
				Default: 0,
			},
			"password": {
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
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
			"client_ssl_certificate": {
				Description: "SSL certificate to authenticate user connections",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"periodic_backup_path": {
				Description:   "Path that will be used to store database backup files",
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"remote_backup"},
				Deprecated:    "Use `remote_backup` block instead",
			},
			"replica_of": {
				Description: "Set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsURLWithScheme([]string{"redis"})),
				},
			},
			"alert": {
				Description: "Set of alerts to enable on the database",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description:      "Alert name",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.AlertNameValues(), false)),
						},
						"value": {
							Description: "Alert value",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
			"modules": {
				Description: "Modules to be provisioned in the database",
				Type:        schema.TypeSet,
				// In TF <0.12 List of objects is not supported, so we need to opt-in to use this old behaviour.
				ConfigMode: schema.SchemaConfigModeAttr,
				Optional:   true,
				// The API doesn't allow to update/delete modules. Unless we recreate the database.
				ForceNew: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the module to enable",
							Type:        schema.TypeString,
							ForceNew:    true,
							Required:    true,
						},
					},
				},
			},
			"source_ips": {
				Description: "Set of CIDR addresses to allow access to the database",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
				},
			},
			"hashing_policy": {
				Description: "List of regular expression rules to shard the database by. See the documentation on clustering for more information on the hashing policy - https://docs.redislabs.com/latest/rc/concepts/clustering/",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					// Can't check that these are valid regex rules as the service wants something like `(?<tag>.*)`
					// which isn't a valid Go regex
				},
			},
			"enable_tls": {
				Description: "Use TLS for authentication",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"enable_default_user": {
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"port": {
				Description:      "TCP port on which the database is available",
				Type:             schema.TypeInt,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(10000, 19999)),
				Optional:         true,
				ForceNew:         true,
			},
			"remote_backup": {
				Description:   "An object that specifies the backup options for the database",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"periodic_backup_path"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": {
							Description:      "Defines the frequency of the automatic backup",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.BackupIntervals(), false)),
						},
						"time_utc": {
							Description:      "Defines the hour automatic backups are made - only applicable when interval is `every-12-hours` or `every-24-hours`",
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: isTime(),
							DiffSuppressFunc: skipDiffIfIntervalIs12And12HourTimeDiff,
						},
						"storage_type": {
							Description:      "Defines the provider of the storage location",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.BackupStorageTypes(), false)),
						},
						"storage_path": {
							Description: "Defines a URI representing the backup storage location",
							Type:        schema.TypeString,
							Required:    true,
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

func resourceRedisCloudFlexibleDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId := d.Get("subscription_id").(int)

	subscriptionMutex.Lock(subId)

	name := d.Get("name").(string)
	protocol := d.Get("protocol").(string)
	memoryLimitInGB := d.Get("memory_limit_in_gb").(float64)
	supportOSSClusterAPI := d.Get("support_oss_cluster_api").(bool)
	respVersion := d.Get("resp_version").(string)
	dataPersistence := d.Get("data_persistence").(string)
	dataEviction := d.Get("data_eviction").(string)
	password := d.Get("password").(string)
	replication := d.Get("replication").(bool)
	throughputMeasurementBy := d.Get("throughput_measurement_by").(string)
	throughputMeasurementValue := d.Get("throughput_measurement_value").(int)
	averageItemSizeInBytes := d.Get("average_item_size_in_bytes").(int)

	createModules := make([]*databases.Module, 0)
	modules := d.Get("modules").(*schema.Set)
	for _, module := range modules.List() {
		moduleMap := module.(map[string]interface{})

		modName := moduleMap["name"].(string)

		createModule := &databases.Module{
			Name: redis.String(modName),
		}

		createModules = append(createModules, createModule)
	}

	createAlerts := make([]*databases.Alert, 0)
	alerts := d.Get("alert").(*schema.Set)
	for _, alert := range alerts.List() {
		alertMap := alert.(map[string]interface{})

		alertName := alertMap["name"].(string)
		alertValue := alertMap["value"].(int)

		createAlert := &databases.Alert{
			Name:  redis.String(alertName),
			Value: redis.Int(alertValue),
		}

		createAlerts = append(createAlerts, createAlert)
	}

	createDatabase := databases.CreateDatabase{
		Name:                 redis.String(name),
		Protocol:             redis.String(protocol),
		MemoryLimitInGB:      redis.Float64(memoryLimitInGB),
		SupportOSSClusterAPI: redis.Bool(supportOSSClusterAPI),
		DataPersistence:      redis.String(dataPersistence),
		DataEvictionPolicy:   redis.String(dataEviction),
		Replication:          redis.Bool(replication),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    redis.String(throughputMeasurementBy),
			Value: redis.Int(throughputMeasurementValue),
		},
		Modules:      createModules,
		Alerts:       createAlerts,
		RemoteBackup: buildBackupPlan(d.Get("remote_backup").([]interface{}), d.Get("periodic_backup_path")),
	}

	if password != "" {
		createDatabase.Password = redis.String(password)
	}

	if averageItemSizeInBytes > 0 {
		createDatabase.AverageItemSizeInBytes = &averageItemSizeInBytes
	}

	if v, ok := d.GetOk("port"); ok {
		createDatabase.PortNumber = redis.Int(v.(int))
	}

	if respVersion != "" {
		createDatabase.RespVersion = redis.String(respVersion)
	}

	dbId, err := api.client.Database.Create(ctx, subId, createDatabase)
	if err != nil {
		subscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	d.SetId(buildResourceId(subId, dbId))

	// Confirm Subscription Active status
	err = waitForDatabaseToBeActive(ctx, subId, dbId, api)
	if err != nil {
		subscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	// Locate Databases to confirm Active status

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password, enableDefaultUser and so on.
	subscriptionMutex.Unlock(subId)
	return resourceRedisCloudSubscriptionDatabaseUpdate(ctx, d, meta)
}

func resourceRedisCloudFlexibleDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, dbId, err := toDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// We are not import this resource, so we can read the subscription_id defined in this resource.
	if subId == 0 {
		subId = d.Get("subscription_id").(int)
	}

	db, err := api.client.Database.Get(ctx, subId, dbId)
	if err != nil {
		if _, ok := err.(*databases.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("db_id", redis.IntValue(db.ID)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(db.Name)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("protocol", redis.StringValue(db.Protocol)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("memory_limit_in_gb", redis.Float64Value(db.MemoryLimitInGB)); err != nil {
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

	if err := d.Set("throughput_measurement_by", redis.StringValue(db.ThroughputMeasurement.By)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("throughput_measurement_value", redis.IntValue(db.ThroughputMeasurement.Value)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("public_endpoint", redis.StringValue(db.PublicEndpoint)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("private_endpoint", redis.StringValue(db.PrivateEndpoint)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("modules", flattenModules(db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("alert", flattenAlerts(db.Alerts)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("average_item_size_in_bytes", d.Get("average_item_size_in_bytes").(int)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("external_endpoint_for_oss_cluster_api",
		d.Get("external_endpoint_for_oss_cluster_api").(bool)); err != nil {
		return diag.FromErr(err)
	}

	password := d.Get("password").(string)
	if redis.StringValue(db.Protocol) == "redis" {
		// Only db with the "redis" protocol returns the password.
		password = redis.StringValue(db.Security.Password)
	}

	if err := d.Set("password", password); err != nil {
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

	if err := d.Set("hashing_policy", flattenRegexRules(db.Clustering.RegexRules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("enable_tls", redis.Bool(*db.Security.EnableTls)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("enable_default_user", redis.Bool(*db.Security.EnableDefaultUser)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("remote_backup", flattenBackupPlan(db.Backup, d.Get("remote_backup").([]interface{}), d.Get("periodic_backup_path").(string))); err != nil {
		return diag.FromErr(err)
	}

	var parsedLatestBackupStatus []map[string]interface{}
	latestBackupStatus, err := api.client.LatestBackup.Get(ctx, subId, dbId)
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
	latestImportStatus, err := api.client.LatestImport.Get(ctx, subId, dbId)
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

func resourceRedisCloudFlexibleDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	var diags diag.Diagnostics
	subId := d.Get("subscription_id").(int)

	_, dbId, err := toDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		return diag.FromErr(err)
	}

	dbErr := api.client.Database.Delete(ctx, subId, dbId)
	if dbErr != nil {
		diag.FromErr(dbErr)
	}
	return diags
}

func resourceRedisCloudFlexibleDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	_, dbId, err := toDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subId := d.Get("subscription_id").(int)
	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	// If the recommended approach is taken and there are 0 alerts, a nil-slice value is sent to the UpdateDatabase
	// constructor. We instead want a non-nil (but zero length) slice to be passed forward.
	//goland:noinspection GoPreferNilSlice
	alerts := []*databases.Alert{}

	for _, alert := range d.Get("alert").(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		alerts = append(alerts, &databases.Alert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	update := databases.UpdateDatabase{
		Name:                 redis.String(d.Get("name").(string)),
		MemoryLimitInGB:      redis.Float64(d.Get("memory_limit_in_gb").(float64)),
		SupportOSSClusterAPI: redis.Bool(d.Get("support_oss_cluster_api").(bool)),
		Replication:          redis.Bool(d.Get("replication").(bool)),
		ThroughputMeasurement: &databases.UpdateThroughputMeasurement{
			By:    redis.String(d.Get("throughput_measurement_by").(string)),
			Value: redis.Int(d.Get("throughput_measurement_value").(int)),
		},
		DataPersistence:    redis.String(d.Get("data_persistence").(string)),
		DataEvictionPolicy: redis.String(d.Get("data_eviction").(string)),
		SourceIP:           setToStringSlice(d.Get("source_ips").(*schema.Set)),
		Alerts:             &alerts,
		RemoteBackup:       buildBackupPlan(d.Get("remote_backup").([]interface{}), d.Get("periodic_backup_path")),
		EnableDefaultUser:  redis.Bool(d.Get("enable_default_user").(bool)),
	}
	if len(setToStringSlice(d.Get("source_ips").(*schema.Set))) == 0 {
		update.SourceIP = []*string{redis.String("0.0.0.0/0")}
	}

	if d.Get("password").(string) != "" {
		update.Password = redis.String(d.Get("password").(string))
	}

	update.ReplicaOf = setToStringSlice(d.Get("replica_of").(*schema.Set))
	if update.ReplicaOf == nil {
		update.ReplicaOf = make([]*string, 0)
	}

	// The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	enableTLS := d.Get("enable_tls").(bool)
	if enableTLS {
		// TLS only: enable_tls=true, client_ssl_certificate="".
		update.EnableTls = redis.Bool(enableTLS)
		// mTLS: enableTls=true, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		}
	} else {
		// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		} else {
			// Default: enable_tls=false, client_ssl_certificate=""
			update.EnableTls = redis.Bool(enableTLS)
		}
	}

	regex := d.Get("hashing_policy").([]interface{})
	if len(regex) != 0 {
		update.RegexRules = interfaceToStringSlice(regex)
	}

	backupPath := d.Get("periodic_backup_path").(string)
	if backupPath != "" {
		update.PeriodicBackupPath = redis.String(backupPath)
	}

	update.UseExternalEndpointForOSSClusterAPI = redis.Bool(d.Get("external_endpoint_for_oss_cluster_api").(bool))

	respVersion := d.Get("resp_version").(string)
	if respVersion != "" {
		update.RespVersion = redis.String(respVersion)
	}

	err = api.client.Database.Update(ctx, subId, dbId, update)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		return diag.FromErr(err)
	}

	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudSubscriptionDatabaseRead(ctx, d, meta)
}

func buildBackupPlan(data interface{}, periodicBackupPath interface{}) *databases.DatabaseBackupConfig {
	var d map[string]interface{}

	switch v := data.(type) {
	case []interface{}:
		if len(v) != 1 {
			if periodicBackupPath == nil {
				return &databases.DatabaseBackupConfig{Active: redis.Bool(false)}
			} else {
				return nil
			}
		}
		d = v[0].(map[string]interface{})
	default:
		d = v.(map[string]interface{})
	}

	config := databases.DatabaseBackupConfig{
		Active:      redis.Bool(true),
		Interval:    redis.String(d["interval"].(string)),
		StorageType: redis.String(d["storage_type"].(string)),
		StoragePath: redis.String(d["storage_path"].(string)),
	}

	if v := d["time_utc"].(string); v != "" {
		config.TimeUTC = redis.String(v)
	}

	return &config
}

func flattenBackupPlan(backup *databases.Backup, existing []interface{}, periodicBackupPath string) []map[string]interface{} {
	if backup == nil || !redis.BoolValue(backup.Enabled) || periodicBackupPath != "" {
		return nil
	}

	storageType := ""
	if len(existing) == 1 {
		d := existing[0].(map[string]interface{})
		storageType = d["storage_type"].(string)
	}

	return []map[string]interface{}{
		{
			"interval":     redis.StringValue(backup.Interval),
			"time_utc":     redis.StringValue(backup.TimeUTC),
			"storage_type": storageType,
			"storage_path": redis.StringValue(backup.Destination),
		},
	}
}

func toDatabaseId(id string) (int, int, error) {
	parts := strings.Split(id, "/")

	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid id: %s", id)
	}

	if len(parts) == 1 {
		dbId, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		return 0, dbId, nil
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	dbId, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return subId, dbId, nil
}

func skipDiffIfIntervalIs12And12HourTimeDiff(k, oldValue, newValue string, d *schema.ResourceData) bool {
	// If interval is set to every 12 hours and the `time_utc` is in the afternoon,
	// then the API will return the _morning_ time when queried.
	// `interval` is assumed to be an attribute within the same block as the attribute being diffed.

	parts := strings.Split(k, ".")
	parts[len(parts)-1] = "interval"

	var interval = d.Get(strings.Join(parts, "."))

	if interval != databases.BackupIntervalEvery12Hours {
		return false
	}

	oldTime, err := time.Parse("15:04", oldValue)
	if err != nil {
		return false
	}
	newTime, err := time.Parse("15:04", newValue)
	if err != nil {
		return false
	}

	return oldTime.Minute() == newTime.Minute() && oldTime.Add(12*time.Hour).Hour() == newTime.Hour()
}

func remoteBackupIntervalSetCorrectly(key string) schema.CustomizeDiffFunc {
	// Validate multiple attributes - https://github.com/hashicorp/terraform-plugin-sdk/issues/233

	return func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
		if v, ok := diff.GetOk(key); ok {
			backups := v.([]interface{})
			if len(backups) == 1 {
				v := backups[0].(map[string]interface{})

				interval := v["interval"].(string)
				timeUtc := v["time_utc"].(string)

				if interval != databases.BackupIntervalEvery12Hours && interval != databases.BackupIntervalEvery24Hours && timeUtc != "" {
					return fmt.Errorf("unexpected value at %s.0.time_utc - time_utc can only be set when interval is either %s or %s", key, databases.BackupIntervalEvery24Hours, databases.BackupIntervalEvery12Hours)
				}
			}
		}
		return nil
	}

}

func parseLatestBackupStatus(latestBackupStatus *latest_backups.LatestBackupStatus) ([]map[string]interface{}, error) {
	lbs := map[string]interface{}{
		"response": nil,
		"error":    nil,
	}

	if latestBackupStatus.Response.Resource != nil {
		j, err := json.Marshal(latestBackupStatus.Response.Resource)
		if err != nil {
			return nil, err
		}
		lbs["response"] = string(j)
	}

	if latestBackupStatus.Response.Error != nil {
		err := map[string]interface{}{
			"type":        redis.StringValue(latestBackupStatus.Response.Error.Type),
			"description": redis.StringValue(latestBackupStatus.Response.Error.Description),
			"status":      redis.StringValue(latestBackupStatus.Response.Error.Status),
		}
		lbs["error"] = []map[string]interface{}{err}
	}

	return []map[string]interface{}{lbs}, nil
}

func parseLatestImportStatus(latestImportStatus *latest_imports.LatestImportStatus) ([]map[string]interface{}, error) {
	lis := map[string]interface{}{
		"response": nil,
		"error":    nil,
	}

	if latestImportStatus.Response.Resource != nil {
		j, err := json.Marshal(latestImportStatus.Response.Resource)
		if err != nil {
			return nil, err
		}
		lis["response"] = string(j)
	}

	if latestImportStatus.Response.Error != nil {
		err := map[string]interface{}{
			"type":        redis.StringValue(latestImportStatus.Response.Error.Type),
			"description": redis.StringValue(latestImportStatus.Response.Error.Description),
			"status":      redis.StringValue(latestImportStatus.Response.Error.Status),
		}
		lis["error"] = []map[string]interface{}{err}
	}

	return []map[string]interface{}{lis}, nil
}
