package provider

import (
	"context"
	"log"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/tags"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudEssentialsDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates database resource within an essentials subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudEssentialsDatabaseCreate,
		ReadContext:   resourceRedisCloudEssentialsDatabaseRead,
		UpdateContext: resourceRedisCloudEssentialsDatabaseUpdate,
		DeleteContext: resourceRedisCloudEssentialsDatabaseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				subId, dbId, err := pro.ToDatabaseId(d.Id())
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

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "Identifier of the essentials subscription",
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
				Description:      "The protocol that will be used to access the database, (either 'redis', 'memcached' or 'stack')",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(fixedDatabases.ProtocolValues(), false)),
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
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
				Description: "The compliance level (redis version) of this database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"resp_version": {
				Description: "RESP version must be compatible with Redis version.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"data_persistence": {
				Description: "Rate of database data persistence (in persistent storage).",
				Type:        schema.TypeString,
				Required:    true,
			},
			"data_eviction": {
				Description: "The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "volatile-lru",
			},
			"replication": {
				Description: "Database's replication",
				Type:        schema.TypeBool,
				Required:    true,
			},
			"activated_on": {
				Description: "When this database was activated",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"periodic_backup_path": {
				Description: "Path that will be used to store database backup files",
				Type:        schema.TypeString,
				Optional:    true,
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
				Description: "Set of CIDR addresses to allow access to the database.",
				Type:        schema.TypeList,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
				},
				DiffSuppressFunc: suppressIfPaygDisabled,
			},
			"replica": {
				Description: "Details of database replication",
				Type:        schema.TypeList,
				MinItems:    1,
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sync_source": {
							Description: "A source database to replicate here",
							Type:        schema.TypeSet,
							MinItems:    1,
							Required:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"endpoint": {
										Description: "A Redis URI (sample format: 'redis://user:password@host:port)'. If the URI provided is Redis Cloud instance, only host and port should be provided (using the format: ['redis://endpoint1:6379', 'redis://endpoint2:6380'])",
										Type:        schema.TypeString,
										Required:    true,
									},
									"encryption": {
										Description: "Defines if encryption should be used to connect to the sync source. If not provided and if the source is a Redis Cloud instance, it will automatically detect if the source uses encryption",
										Type:        schema.TypeBool,
										Optional:    true,
									},
									"server_cert": {
										Description: "TLS/SSL certificate chain of the sync source. If left null and if the source is a Redis Cloud instance, it will automatically detect the certificate to use",
										Type:        schema.TypeString,
										Optional:    true,
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
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"password": {
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Computed:    true,
			},
			"enable_default_user": {
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"alert": {
				Description: "Set of alerts to enable on the database",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Alert name",
							Type:        schema.TypeString,
							Required:    true,
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
				Description: "Modules to be provisioned in the database. Note: Not supported for Redis 8.0 and higher as modules are bundled by default.",
				Type:        schema.TypeSet,
				// In TF <0.12 List of objects is not supported, so we need to opt-in to use this old behaviour.
				ConfigMode: schema.SchemaConfigModeAttr,
				Optional:   true,
				Computed:   true,
				ForceNew:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the module to enable",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"enable_payg_features": {
				Description: "Enable features for PAYG databases",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"memory_limit_in_gb": {
				Description:      "Maximum memory usage for this specific database",
				Type:             schema.TypeFloat,
				Optional:         true,
				DiffSuppressFunc: suppressIfPaygDisabled,
			},
			"support_oss_cluster_api": {
				Description:      "Support Redis open-source (OSS) Cluster API",
				Type:             schema.TypeBool,
				Optional:         true,
				DiffSuppressFunc: suppressIfPaygDisabled,
			},
			"external_endpoint_for_oss_cluster_api": {
				Description:      "Should use the external endpoint for open-source (OSS) Cluster API",
				Type:             schema.TypeBool,
				Optional:         true,
				DiffSuppressFunc: suppressIfPaygDisabled,
			},
			"enable_database_clustering": {
				Description:      "Distributes database data to different cloud instances",
				Type:             schema.TypeBool,
				Optional:         true,
				DiffSuppressFunc: suppressIfPaygDisabled,
			},
			"regex_rules": {
				Description: "Shard regex rules. Relevant only for a sharded database. Supported only for 'Pay-As-You-Go' subscriptions",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					// Can't check that these are valid regex rules as the service wants something like `(?<tag>.*)`
					// which isn't a valid Go regex
				},
				DiffSuppressFunc: suppressIfPaygDisabled,
			},
			"enable_tls": {
				Description:      "Use TLS for authentication",
				Type:             schema.TypeBool,
				Optional:         true,
				DiffSuppressFunc: suppressIfPaygDisabled,
			},
			"tags": {
				Description: "Tags for database management",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:         true,
				ValidateDiagFunc: pro.ValidateTagsfunc,
			},
		},
	}
}

func resourceRedisCloudEssentialsDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId := d.Get("subscription_id").(int)

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	createDatabaseRequest := fixedDatabases.CreateFixedDatabase{
		Name:               redis.String(d.Get("name").(string)),
		DataPersistence:    redis.String(d.Get("data_persistence").(string)),
		DataEvictionPolicy: redis.String(d.Get("data_eviction").(string)),
		Replication:        redis.Bool(d.Get("replication").(bool)),
		PeriodicBackupPath: redis.String(d.Get("periodic_backup_path").(string)),
	}

	protocol := d.Get("protocol").(string)
	if protocol != "" {
		createDatabaseRequest.Protocol = redis.String(protocol)
	}

	respVersion := d.Get("resp_version").(string)
	if respVersion != "" {
		createDatabaseRequest.RespVersion = redis.String(respVersion)
	}

	sourceIps := utils.InterfaceToStringSlice(d.Get("source_ips").([]interface{}))
	if len(sourceIps) == 0 {
		createDatabaseRequest.SourceIPs = []*string{redis.String("0.0.0.0/0")}
	} else {
		createDatabaseRequest.SourceIPs = sourceIps
	}

	replicaRaw := d.Get("replica").([]interface{})
	if len(replicaRaw) == 1 {
		firstReplica := replicaRaw[0].(map[string]interface{})
		syncSources := make([]*fixedDatabases.SyncSource, 0)

		for _, sSourceRaw := range firstReplica["sync_source"].(*schema.Set).List() {
			sSource := sSourceRaw.(map[string]interface{})
			syncSources = append(syncSources, &fixedDatabases.SyncSource{
				Endpoint:   redis.String(sSource["endpoint"].(string)),
				Encryption: redis.Bool(sSource["encryption"].(bool)),
				ServerCert: redis.String(sSource["server_cert"].(string)),
			})
		}

		createReplica := &fixedDatabases.ReplicaOf{
			SyncSources: syncSources,
		}
		createDatabaseRequest.Replica = createReplica
	}

	tlsCertificates := utils.InterfaceToStringSlice(d.Get("client_tls_certificates").([]interface{}))
	if len(tlsCertificates) > 0 {
		createCertificates := make([]*fixedDatabases.DatabaseCertificate, 0)
		for _, cert := range tlsCertificates {
			createCertificates = append(createCertificates, &fixedDatabases.DatabaseCertificate{
				PublicCertificatePEMString: cert,
			})
		}
		createDatabaseRequest.ClientTlsCertificates = createCertificates
	}

	password := d.Get("password").(string)
	if password != "" {
		createDatabaseRequest.Password = redis.String(password)
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
	createDatabaseRequest.Alerts = &createAlerts

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
	createDatabaseRequest.Modules = &createModules

	if d.Get("enable_payg_features").(bool) {
		createDatabaseRequest.MemoryLimitInGB = redis.Float64(d.Get("memory_limit_in_gb").(float64))
		createDatabaseRequest.SupportOSSClusterAPI = redis.Bool(d.Get("support_oss_cluster_api").(bool))
		createDatabaseRequest.UseExternalEndpointForOSSClusterAPI = redis.Bool(d.Get("external_endpoint_for_oss_cluster_api").(bool))
		createDatabaseRequest.EnableDatabaseClustering = redis.Bool(d.Get("enable_database_clustering").(bool))
		createDatabaseRequest.RegexRules = utils.InterfaceToStringSlice(d.Get("regex_rules").([]interface{}))
		createDatabaseRequest.EnableTls = redis.Bool(d.Get("enable_tls").(bool))
	}

	log.Printf("[DEBUG] Full essentials database create request - Name: %s, Protocol: %s, DataPersistence: %s, DataEviction: %s, Replication: %v, SourceIPs: %v, EnablePaygFeatures: %v, MemoryLimitInGB: %v, SupportOSSClusterAPI: %v, ExternalEndpointForOSSClusterAPI: %v, EnableDatabaseClustering: %v, EnableTls: %v",
		redis.StringValue(createDatabaseRequest.Name),
		redis.StringValue(createDatabaseRequest.Protocol),
		redis.StringValue(createDatabaseRequest.DataPersistence),
		redis.StringValue(createDatabaseRequest.DataEvictionPolicy),
		redis.BoolValue(createDatabaseRequest.Replication),
		createDatabaseRequest.SourceIPs,
		d.Get("enable_payg_features").(bool),
		createDatabaseRequest.MemoryLimitInGB,
		createDatabaseRequest.SupportOSSClusterAPI,
		createDatabaseRequest.UseExternalEndpointForOSSClusterAPI,
		createDatabaseRequest.EnableDatabaseClustering,
		createDatabaseRequest.EnableTls)

	log.Printf("[DEBUG] Calling FixedDatabases.Create API for subscription %d with database name: %s", subId, redis.StringValue(createDatabaseRequest.Name))
	databaseId, err := api.Client.FixedDatabases.Create(ctx, subId, createDatabaseRequest)
	if err != nil {
		log.Printf("[ERROR] FixedDatabases.Create failed for subscription %d: %v", subId, err)
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] FixedDatabases.Create succeeded for subscription %d, database ID: %d", subId, databaseId)

	d.SetId(utils.BuildResourceId(subId, databaseId))

	// Confirm Subscription Active status
	err = waitForEssentialsDatabaseToBeActive(ctx, subId, databaseId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases (enableDefaultUser)
	// Others are omitted here _because_ the update will take care of them, such as tags
	return resourceRedisCloudEssentialsDatabaseUpdate(ctx, d, meta)
}

func resourceRedisCloudEssentialsDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	var diags diag.Diagnostics

	subId, databaseId, err := pro.ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// We are not import this resource, so we can read the subscription_id defined in this resource.
	if subId == 0 {
		subId = d.Get("subscription_id").(int)
	}

	db, err := api.Client.FixedDatabases.Get(ctx, subId, databaseId)
	if err != nil {
		if _, ok := err.(*fixedDatabases.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId(utils.BuildResourceId(subId, databaseId))

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

	if db.Security == nil {
		if err := d.Set("source_ips", []string{}); err != nil {
			return diag.FromErr(err)
		}
	} else {
		var sourceIPs []string
		if !(len(db.Security.SourceIPs) == 1 && redis.StringValue(db.Security.SourceIPs[0]) == "0.0.0.0/0") {
			// The API handles an empty list as ["0.0.0.0/0"] but need to be careful to match the input to avoid Terraform detecting drift
			sourceIPs = redis.StringSliceValue(db.Security.SourceIPs...)
		}
		if err := d.Set("source_ips", sourceIPs); err != nil {
			return diag.FromErr(err)
		}
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

	if err := d.Set("password", redis.StringValue(db.Security.Password)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("enable_default_user", redis.Bool(*db.Security.EnableDefaultUser)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("alert", pro.FlattenAlerts(*db.Alerts)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("modules", pro.FlattenModules(*db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	// PAYG features
	if err := d.Set("memory_limit_in_gb", redis.Float64Value(db.MemoryLimitInGb)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("external_endpoint_for_oss_cluster_api", redis.BoolValue(db.UseExternalEndpointForOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}

	if db.Clustering == nil {
		if err := d.Set("enable_database_clustering", false); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("regex_rules", []interface{}{}); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("enable_database_clustering", redis.BoolValue(db.Clustering.Enabled)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("regex_rules", pro.FlattenRegexRules(db.Clustering.RegexRules)); err != nil {
			return diag.FromErr(err)
		}
	}

	if db.Security == nil {
		if err := d.Set("enable_tls", false); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("enable_tls", redis.BoolValue(db.Security.EnableTls)); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := readFixedTags(ctx, api, subId, databaseId, d); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudEssentialsDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	_, databaseId, err := pro.ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subId := d.Get("subscription_id").(int)
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	updateDatabaseRequest := fixedDatabases.UpdateFixedDatabase{
		Name:               redis.String(d.Get("name").(string)),
		DataPersistence:    redis.String(d.Get("data_persistence").(string)),
		DataEvictionPolicy: redis.String(d.Get("data_eviction").(string)),
		Replication:        redis.Bool(d.Get("replication").(bool)),
		PeriodicBackupPath: redis.String(d.Get("periodic_backup_path").(string)),
		EnableDefaultUser:  redis.Bool(d.Get("enable_default_user").(bool)),
	}

	respVersion := d.Get("resp_version").(string)
	if respVersion != "" {
		updateDatabaseRequest.RespVersion = redis.String(respVersion)
	}

	sourceIps := utils.InterfaceToStringSlice(d.Get("source_ips").([]interface{}))
	if len(sourceIps) == 0 {
		updateDatabaseRequest.SourceIPs = []*string{redis.String("0.0.0.0/0")}
	} else {
		updateDatabaseRequest.SourceIPs = sourceIps
	}

	replicaRaw := d.Get("replica").([]interface{})
	if len(replicaRaw) == 1 {
		firstReplica := replicaRaw[0].(map[string]interface{})
		syncSources := make([]*fixedDatabases.SyncSource, 0)

		for _, sSourceRaw := range firstReplica["sync_source"].(*schema.Set).List() {
			sSource := sSourceRaw.(map[string]interface{})
			syncSources = append(syncSources, &fixedDatabases.SyncSource{
				Endpoint:   redis.String(sSource["endpoint"].(string)),
				Encryption: redis.Bool(sSource["encryption"].(bool)),
				ServerCert: redis.String(sSource["server_cert"].(string)),
			})
		}

		createReplica := &fixedDatabases.ReplicaOf{
			SyncSources: syncSources,
		}
		updateDatabaseRequest.Replica = createReplica
	}

	tlsCertificates := utils.InterfaceToStringSlice(d.Get("client_tls_certificates").([]interface{}))
	if len(tlsCertificates) > 0 {
		createCertificates := make([]*fixedDatabases.DatabaseCertificate, 0)
		for _, cert := range tlsCertificates {
			createCertificates = append(createCertificates, &fixedDatabases.DatabaseCertificate{
				PublicCertificatePEMString: cert,
			})
		}
		updateDatabaseRequest.ClientTlsCertificates = createCertificates
	}

	password := d.Get("password").(string)
	if password != "" {
		updateDatabaseRequest.Password = redis.String(password)
	}

	// can't update the password and disable the default user at once
	if !d.Get("enable_default_user").(bool) {
		updateDatabaseRequest.Password = nil
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
	updateDatabaseRequest.Alerts = &createAlerts

	if d.Get("enable_payg_features").(bool) {
		updateDatabaseRequest.MemoryLimitInGB = redis.Float64(d.Get("memory_limit_in_gb").(float64))
		updateDatabaseRequest.SupportOSSClusterAPI = redis.Bool(d.Get("support_oss_cluster_api").(bool))
		updateDatabaseRequest.UseExternalEndpointForOSSClusterAPI = redis.Bool(d.Get("external_endpoint_for_oss_cluster_api").(bool))
		updateDatabaseRequest.EnableDatabaseClustering = redis.Bool(d.Get("enable_database_clustering").(bool))
		updateDatabaseRequest.RegexRules = utils.InterfaceToStringSlice(d.Get("regex_rules").([]interface{}))
		updateDatabaseRequest.EnableTls = redis.Bool(d.Get("enable_tls").(bool))
	}

	err = api.Client.FixedDatabases.Update(ctx, subId, databaseId, updateDatabaseRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := waitForEssentialsDatabaseToBeActive(ctx, subId, databaseId, api); err != nil {
		return diag.FromErr(err)
	}

	if err := waitForEssentialsSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	// The Tags API is synchronous so we shouldn't need to do any more waiting
	if err := writeFixedTags(ctx, api, subId, databaseId, d); err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudEssentialsDatabaseRead(ctx, d, meta)
}

func resourceRedisCloudEssentialsDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*client.ApiClient)

	var diags diag.Diagnostics
	subId := d.Get("subscription_id").(int)

	_, databaseId, err := pro.ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := waitForEssentialsDatabaseToBeActive(ctx, subId, databaseId, api); err != nil {
		return diag.FromErr(err)
	}

	dbErr := api.Client.FixedDatabases.Delete(ctx, subId, databaseId)
	if dbErr != nil {
		return diag.FromErr(dbErr)
	}
	return diags
}

func waitForEssentialsDatabaseToBeActive(ctx context.Context, subId, id int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay: 30 * time.Second,
		Pending: []string{
			databases.StatusDraft,
			databases.StatusPending,
			databases.StatusActiveChangePending,
			databases.StatusRCPActiveChangeDraft,
			databases.StatusActiveChangeDraft,
			databases.StatusRCPDraft,
			databases.StatusRCPChangePending,
			databases.StatusProxyPolicyChangePending,
			databases.StatusProxyPolicyChangeDraft,
			databases.StatusDynamicEndpointsCreationPending,
		},
		Target:       []string{databases.StatusActive},
		Timeout:      utils.SafetyTimeout,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed database %d to be active", id)

			database, err := api.Client.FixedDatabases.Get(ctx, subId, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(database.Status), redis.StringValue(database.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func writeReplica(replica fixedDatabases.ReplicaOf) []map[string]interface{} {
	tf := map[string]interface{}{}
	syncSources := make([]map[string]interface{}, 0)

	for _, ss := range replica.SyncSources {
		syncSources = append(syncSources, map[string]interface{}{
			"endpoint":    redis.StringValue(ss.Endpoint),
			"encryption":  redis.BoolValue(ss.Encryption),
			"server_cert": redis.StringValue(ss.ServerCert),
		})
	}

	tf["sync_source"] = syncSources
	return []map[string]interface{}{tf}
}

func suppressIfPaygDisabled(_, _, _ string, d *schema.ResourceData) bool {
	// If payg is disabled, suppress diff checks on payg attributes
	return !d.Get("enable_payg_features").(bool)
}

func readFixedTags(ctx context.Context, api *client.ApiClient, subId int, databaseId int, d *schema.ResourceData) error {
	t := make(map[string]string)
	tagResponse, err := api.Client.Tags.GetFixed(ctx, subId, databaseId)
	if err != nil {
		return err
	}
	if tagResponse.Tags != nil {
		for _, tag := range *tagResponse.Tags {
			t[redis.StringValue(tag.Key)] = redis.StringValue(tag.Value)
		}
	}
	return d.Set("tags", t)
}

func writeFixedTags(ctx context.Context, api *client.ApiClient, subId int, databaseId int, d *schema.ResourceData) error {
	t := make([]*tags.Tag, 0)
	tState := d.Get("tags").(map[string]interface{})
	for k, v := range tState {
		t = append(t, &tags.Tag{
			Key:   redis.String(k),
			Value: redis.String(v.(string)),
		})
	}
	return api.Client.Tags.PutFixed(ctx, subId, databaseId, tags.AllTags{Tags: &t})
}

func essentialsCustomizeDiff() schema.CustomizeDiffFunc {
	return func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
		// Check if user is trying to specify modules
		modules, modulesExists := diff.GetOkExists("modules")

		if modulesExists {
			moduleSet := modules.(*schema.Set)
			if moduleSet.Len() > 0 {
				// Warn, don't error
				log.Printf("[WARN] Modules are explicitly configured. Note that some plans may use Redis 8.0+ where modules are bundled by default. The API will reject invalid configurations.")
			}
		}
		return nil
	}
}
