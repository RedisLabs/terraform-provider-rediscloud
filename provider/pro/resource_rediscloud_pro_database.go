package pro

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceRedisCloudProDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates database resource within a pro subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudProDatabaseCreate,
		ReadContext:   resourceRedisCloudProDatabaseRead,
		UpdateContext: resourceRedisCloudProDatabaseUpdate,
		DeleteContext: resourceRedisCloudProDatabaseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				subId, dbId, err := ToDatabaseId(d.Id())
				if err != nil {
					return nil, err
				}
				if err := d.Set("subscription_id", subId); err != nil {
					return nil, err
				}
				if err := d.Set("db_id", dbId); err != nil {
					return nil, err
				}
				d.SetId(utils.BuildResourceId(subId, dbId))
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		CustomizeDiff: customizeDiff(),

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "Identifier of the pro subscription",
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
				Description:      "The protocol that will be used to access the database (either ‘redis’ or 'memcached’)",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.ProtocolValues(), false)),
				Optional:         true,
				ForceNew:         true,
				Default:          "redis",
			},
			"memory_limit_in_gb": {
				Description:  "(Deprecated) Maximum memory usage for this specific database",
				Type:         schema.TypeFloat,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"memory_limit_in_gb", "dataset_size_in_gb"},
			},
			"dataset_size_in_gb": {
				Description:  "Maximum amount of data in the dataset for this specific database in GB",
				Type:         schema.TypeFloat,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"memory_limit_in_gb", "dataset_size_in_gb"},
			},
			"redis_version": {
				Description: "Defines the Redis database version. If omitted, the Redis version will be set to the default version",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
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
			"client_tls_certificates": {
				Description: "TLS certificates to authenticate user connections",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"client_ssl_certificate"},
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
			"query_performance_factor": {
				Description: "Query performance factor for this specific database",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					matched, err := regexp.MatchString(`^([2468])x$`, v)
					if err != nil {
						errs = append(errs, fmt.Errorf("regex match failed: %s", err))
						return
					}
					if !matched {
						errs = append(errs, fmt.Errorf("%q must be an even value between 2x and 8x (inclusive), got: %s", key, v))
					}
					return
				},
			},
			"modules": {
				Description: "Modules to be provisioned in the database. Note: NOT supported for Redis 8.0 and higher as modules are bundled by default.",
				Type:        schema.TypeSet,
				// In TF <0.12 List of objects is not supported, so we need to opt in to use this old behaviour.
				ConfigMode: schema.SchemaConfigModeAttr,
				Optional:   true,
				// The API doesn't allow updating/delete modules. Unless we recreate the database.
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
							ValidateDiagFunc: utils.IsTime(),
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
			"tags": {
				Description: "Tags for database management",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:         true,
				ValidateDiagFunc: ValidateTagsfunc,
			},
		},
	}
}

func resourceRedisCloudProDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId := *utils.GetInt(d, "subscription_id")
	utils.SubscriptionMutex.Lock(subId)

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
		Name:                 utils.GetString(d, "name"),
		Protocol:             utils.GetString(d, "protocol"),
		SupportOSSClusterAPI: utils.GetBool(d, "support_oss_cluster_api"),
		DataPersistence:      utils.GetString(d, "data_persistence"),
		DataEvictionPolicy:   utils.GetString(d, "data_eviction"),
		Replication:          utils.GetBool(d, "replication"),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    utils.GetString(d, "throughput_measurement_by"),
			Value: utils.GetInt(d, "throughput_measurement_value"),
		},
		Modules:      createModules,
		Alerts:       createAlerts,
		RemoteBackup: BuildBackupPlan(d.Get("remote_backup").([]interface{}), d.Get("periodic_backup_path")),
	}

	utils.SetStringIfNotEmpty(d, "query_performance_factor", func(s *string) {
		createDatabase.QueryPerformanceFactor = s
	})

	utils.SetStringIfNotEmpty(d, "redis_version", func(s *string) {
		createDatabase.RedisVersion = s
	})

	utils.SetStringIfNotEmpty(d, "password", func(s *string) {
		createDatabase.Password = s
	})

	utils.SetIntIfPositive(d, "average_item_size_in_bytes", func(i *int) {
		createDatabase.AverageItemSizeInBytes = i
	})

	utils.SetFloat64(d, "dataset_size_in_gb", func(f *float64) {
		createDatabase.DatasetSizeInGB = f
	})

	utils.SetFloat64(d, "memory_limit_in_gb", func(f *float64) {
		createDatabase.MemoryLimitInGB = f
	})

	utils.SetInt(d, "port", func(i *int) {
		createDatabase.PortNumber = i
	})

	utils.SetStringIfNotEmpty(d, "resp_version", func(s *string) {
		createDatabase.RespVersion = s
	})

	// Confirm sub is ready to accept a db request
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	dbId, err := api.Client.Database.Create(ctx, subId, createDatabase)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	d.SetId(utils.BuildResourceId(subId, dbId))

	// Confirm db + sub active status
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password, enableDefaultUser and so on.
	utils.SubscriptionMutex.Unlock(subId)
	return resourceRedisCloudProDatabaseUpdate(ctx, d, meta)
}

func resourceRedisCloudProDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	var diags diag.Diagnostics

	subId, dbId, err := ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// We are not import this resource, so we can read the subscription_id defined in this resource.
	if subId == 0 {
		subId = d.Get("subscription_id").(int)
	}

	db, err := api.Client.Database.Get(ctx, subId, dbId)
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

	if err := d.Set("query_performance_factor", redis.StringValue(db.QueryPerformanceFactor)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("redis_version", redis.StringValue(db.RedisVersion)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("modules", FlattenModules(db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("alert", FlattenAlerts(db.Alerts)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("average_item_size_in_bytes", d.Get("average_item_size_in_bytes").(int)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("external_endpoint_for_oss_cluster_api",
		d.Get("external_endpoint_for_oss_cluster_api").(bool)); err != nil {
		return diag.FromErr(err)
	}

	// To prevent both fields being included in API requests, only one of these two fields should be set in the state
	// Only add `dataset_size_in_gb` to the state if `memory_limit_in_gb` is not already in the state
	if _, inState := d.GetOk("memory_limit_in_gb"); !inState {
		if err := d.Set("dataset_size_in_gb", redis.Float64Value(db.DatasetSizeInGB)); err != nil {
			return diag.FromErr(err)
		}
	}

	// Likewise, only add `memory_limit_in_gb` to the state if `dataset_size_in_gb` is not already in the state
	if _, inState := d.GetOk("dataset_size_in_gb"); !inState {
		if err := d.Set("memory_limit_in_gb", redis.Float64Value(db.MemoryLimitInGB)); err != nil {
			return diag.FromErr(err)
		}
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

	if err := d.Set("hashing_policy", FlattenRegexRules(db.Clustering.RegexRules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("enable_tls", redis.Bool(*db.Security.EnableTls)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("enable_default_user", redis.Bool(*db.Security.EnableDefaultUser)); err != nil {
		return diag.FromErr(err)
	}

	tlsAuthEnabled := *db.Security.TLSClientAuthentication
	if err := utils.ApplyCertificateHints(tlsAuthEnabled, d); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("remote_backup", FlattenBackupPlan(db.Backup, d.Get("remote_backup").([]interface{}), d.Get("periodic_backup_path").(string))); err != nil {
		return diag.FromErr(err)
	}

	if db.QueryPerformanceFactor != nil {
		if err := d.Set("query_performance_factor", redis.String(*db.QueryPerformanceFactor)); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := ReadTags(ctx, api, subId, dbId, d); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudProDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*client.ApiClient)

	var diags diag.Diagnostics
	subId := d.Get("subscription_id").(int)

	_, dbId, err := ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	// Confirm sub + db are ready to accept a db request
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		return diag.FromErr(err)
	}

	if err := api.Client.Database.Delete(ctx, subId, dbId); err != nil {
		return diag.FromErr(err)
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudProDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	_, dbId, err := ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subId := d.Get("subscription_id").(int)
	utils.SubscriptionMutex.Lock(subId)

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
		Name:                 utils.GetString(d, "name"),
		SupportOSSClusterAPI: utils.GetBool(d, "support_oss_cluster_api"),
		Replication:          utils.GetBool(d, "replication"),
		ThroughputMeasurement: &databases.UpdateThroughputMeasurement{
			By:    utils.GetString(d, "throughput_measurement_by"),
			Value: utils.GetInt(d, "throughput_measurement_value"),
		},

		DataPersistence:    utils.GetString(d, "data_persistence"),
		DataEvictionPolicy: utils.GetString(d, "data_eviction"),
		SourceIP:           utils.SetToStringSlice(d.Get("source_ips").(*schema.Set)),
		Alerts:             &alerts,
		RemoteBackup:       BuildBackupPlan(d.Get("remote_backup").([]interface{}), d.Get("periodic_backup_path")),
		EnableDefaultUser:  utils.GetBool(d, "enable_default_user"),
	}

	// One of the following fields must be set, validation is handled in the schema (ExactlyOneOf)
	if v, ok := d.GetOk("dataset_size_in_gb"); ok {
		update.DatasetSizeInGB = redis.Float64(v.(float64))
	} else {
		if v, ok := d.GetOk("memory_limit_in_gb"); ok {
			update.MemoryLimitInGB = redis.Float64(v.(float64))
		}
	}

	// The below fields are optional and will only be sent in the request if they are present in the Terraform configuration
	if len(utils.SetToStringSlice(d.Get("source_ips").(*schema.Set))) == 0 {
		update.SourceIP = []*string{redis.String("0.0.0.0/0")}
	}

	queryPerformanceFactor := d.Get("query_performance_factor").(string)
	if queryPerformanceFactor != "" {
		update.QueryPerformanceFactor = redis.String(queryPerformanceFactor)
	}

	if d.Get("password").(string) != "" {
		update.Password = redis.String(d.Get("password").(string))
	}

	update.ReplicaOf = utils.SetToStringSlice(d.Get("replica_of").(*schema.Set))
	if update.ReplicaOf == nil {
		update.ReplicaOf = make([]*string, 0)
	}

	// The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	clientTLSCertificates := utils.InterfaceToStringSlice(d.Get("client_tls_certificates").([]interface{}))
	enableTLS := d.Get("enable_tls").(bool)
	if enableTLS {
		update.EnableTls = redis.Bool(enableTLS)
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)

			// If the user has enableTls=true and provided an SSL certificate, we want to scrub any TLS certificates
			update.ClientTLSCertificates = &[]*string{}
		}
		update.ClientTLSCertificates = &clientTLSCertificates
	} else {
		// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		} else if len(clientTLSCertificates) > 0 {
			utils.SubscriptionMutex.Unlock(subId)
			return diag.Errorf("TLS certificates may not be provided while enable_tls is false")
		} else {
			// Default: enable_tls=false, client_ssl_certificate=""
			update.EnableTls = redis.Bool(enableTLS)
		}
	}

	regex := d.Get("hashing_policy").([]interface{})
	if len(regex) != 0 {
		update.RegexRules = utils.InterfaceToStringSlice(regex)
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

	// Confirm sub + db are ready to accept a db request
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	// if redis_version has changed, then upgrade first
	if d.HasChange("redis_version") {
		// if we have just created the database, it will detect an upgrade unnecessarily
		originalVersion, newVersion := d.GetChange("redis_version")

		// if either version is blank, it could attempt to upgrade unnecessarily.
		// only upgrade when a known version goes to another known version
		if originalVersion.(string) != "" && newVersion.(string) != "" {
			if diags, unlocked := upgradeRedisVersion(ctx, api, subId, dbId, newVersion.(string)); diags != nil {
				if !unlocked {
					utils.SubscriptionMutex.Unlock(subId)
				}
				return diags
			}
		}
	}

	// Confirm db + sub active status

	if err := api.Client.Database.Update(ctx, subId, dbId, update); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	// Confirm db + sub active status
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	// The Tags API is synchronous so we shouldn't have to wait for anything
	if err := WriteTags(ctx, api, subId, dbId, d); err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Unlock(subId)
	return resourceRedisCloudProDatabaseRead(ctx, d, meta)
}

func upgradeRedisVersion(ctx context.Context, api *client.ApiClient, subId int, dbId int, newVersion string) (diag.Diagnostics, bool) {
	log.Printf("[INFO] Requesting Redis version change to %s...", newVersion)

	upgrade := databases.UpgradeRedisVersion{
		TargetRedisVersion: redis.String(newVersion),
	}

	if err := api.Client.Database.UpgradeRedisVersion(ctx, subId, dbId, upgrade); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.Errorf("failed to change Redis version to %s: %v", newVersion, err), true
	}

	log.Printf("[INFO] Redis version change request to %s accepted by API", newVersion)

	// wait for upgrade
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err), true
	}

	return nil, false
}

func BuildBackupPlan(data interface{}, periodicBackupPath interface{}) *databases.DatabaseBackupConfig {
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

func FlattenBackupPlan(backup *databases.Backup, existing []interface{}, periodicBackupPath string) []map[string]interface{} {
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

func ToDatabaseId(id string) (int, int, error) {
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

func customizeDiff() schema.CustomizeDiffFunc {
	return func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
		if err := validateQueryPerformanceFactor()(ctx, diff, meta); err != nil {
			return err
		}
		if err := validateModulesForRedis8()(ctx, diff, meta); err != nil {
			return err
		}
		if err := RemoteBackupIntervalSetCorrectly("remote_backup")(ctx, diff, meta); err != nil {
			return err
		}
		return nil
	}
}

func validateQueryPerformanceFactor() schema.CustomizeDiffFunc {
	return func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
		// Check if "query_performance_factor" is set
		qpf, qpfExists := diff.GetOk("query_performance_factor")

		if qpfExists && qpf.(string) != "" {
			// Check if Redis version is 8.0 or later
			redisVersion, _ := diff.GetOk("redis_version")
			if redisVersion != nil && redisVersion.(string) >= "8.0" {
				// Redis 8.0+ has RediSearch bundled by default, no need to check modules
				return nil
			}

			// Ensure "modules" is explicitly defined in the HCL for Redis < 8.0
			_, modulesExists := diff.GetOkExists("modules")

			if !modulesExists {
				return fmt.Errorf(`"query_performance_factor" requires the "modules" key to be explicitly defined in HCL`)
			}

			// Retrieve modules as a slice of interfaces
			rawModules := diff.Get("modules").(*schema.Set).List()

			// Convert modules to []map[string]interface{}
			var modules []map[string]interface{}
			for _, rawModule := range rawModules {
				if moduleMap, ok := rawModule.(map[string]interface{}); ok {
					modules = append(modules, moduleMap)
				}
			}

			// Check if "RediSearch" exists
			if !containsDBModule(modules, "RediSearch") {
				return fmt.Errorf(`"query_performance_factor" requires the "modules" list to contain "RediSearch"`)
			}
		}
		return nil
	}
}

// Helper function to check if a module exists
func containsDBModule(modules []map[string]interface{}, moduleName string) bool {
	for _, module := range modules {
		if name, ok := module["name"].(string); ok && name == moduleName {
			return true
		}
	}
	return false
}

func validateModulesForRedis8() schema.CustomizeDiffFunc {
	return func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
		redisVersion, versionExists := diff.GetOk("redis_version")
		modules, modulesExists := diff.GetOkExists("modules")

		if versionExists && modulesExists {
			version := redisVersion.(string)
			// Check if version is >= 8.0
			if strings.HasPrefix(version, "8.") {
				moduleSet := modules.(*schema.Set)
				if moduleSet.Len() > 0 {
					return fmt.Errorf(`"modules" cannot be explicitly set for Redis version %s as modules are bundled by default. Remove the "modules" field from your configuration`, version)
				}
			}
		}
		return nil
	}
}

func RemoteBackupIntervalSetCorrectly(key string) schema.CustomizeDiffFunc {
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
