package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates database resource within a subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudDatabaseCreate,
		ReadContext:   resourceRedisCloudDatabaseRead,
		UpdateContext: resourceRedisCloudDatabaseUpdate,
		DeleteContext: resourceRedisCloudDatabaseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				subId, err := strconv.Atoi(d.Id())
				if err != nil {
					return nil, err
				}

				// Populate the names of databases that already exist so that `flattenDatabases` can iterate over them
				api := meta.(*apiClient)
				list := api.client.Database.List(ctx, subId)
				var dbs []map[string]interface{}
				for list.Next() {
					dbs = append(dbs, map[string]interface{}{
						"name": redis.StringValue(list.Value().Name),
					})
				}
				if list.Err() != nil {
					return nil, list.Err()
				}
				if err := d.Set("database", dbs); err != nil {
					return nil, err
				}

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
				Description: "Identifier of the subscription",
				Type:        schema.TypeInt,
				Required:    true,
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
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.ProtocolValues(), false)),
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
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"number-of-shards", "operations-per-second"}, false)),
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
				Description: "Password used to access the database",
				Type:        schema.TypeString,
				Required:    true,
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
			"client_ssl_certificate": {
				Description: "SSL certificate to authenticate user connections",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"periodic_backup_path": {
				Description: "Path that will be used to store database backup files",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"replica_of": {
				Description: "Set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateDiagFunc(validation.IsURLWithScheme([]string{"redis"})),
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
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.AlertNameValues(), false)),
						},
						"value": {
							Description: "Alert value",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
			"module": {
				Description: "A module object",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
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
			"source_ips": {
				Description: "Set of CIDR addresses to allow access to the database",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
				},
			},
			"hashing_policy": {
				Description: "List of regular expression rules to shard the database by. See the documentation on clustering for more information on the hashing policy - https://docs.redislabs.com/latest/rc/concepts/clustering/",
				Type:        schema.TypeList,
				Optional:    true,
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
		},
	}
}

func resourceRedisCloudDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	subId := d.Get("subscription_id").(int)

	// NEW
	name := d.Get("name").(string)
	protocol := d.Get("protocol").(string)
	memoryLimitInGB := d.Get("memory_limit_in_gb").(float64)
	supportOSSClusterAPI := d.Get("support_oss_cluster_api").(bool)
	dataPersistence := d.Get("data_persistence").(string)
	replication := d.Get("replication").(bool)
	throughputMeasurementBy := d.Get("throughput_measurement_by").(string)
	throughputMeasurementValue := d.Get("throughput_measurement_value").(int)
	averageItemSizeInBytes := d.Get("average_item_size_in_bytes").(int)

	createModules := make([]*databases.CreateModule, 0)
	modules := d.Get("module").(*schema.Set)
	for _, module := range modules.List() {
		moduleMap := module.(map[string]interface{})

		modName := moduleMap["name"].(string)

		createModule := &databases.CreateModule{
			Name: redis.String(modName),
		}

		createModules = append(createModules, createModule)
	}

	createAlerts := make([]*databases.CreateAlert, 0)
	alerts := d.Get("alert").(*schema.Set)
	for _, alert := range alerts.List() {
		alertMap := alert.(map[string]interface{})

		alertName := alertMap["name"].(string)
		alertValue := alertMap["value"].(int)

		createAlert := &databases.CreateAlert{
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
		Replication:          redis.Bool(replication),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    redis.String(throughputMeasurementBy),
			Value: redis.Int(throughputMeasurementValue),
		},
		Modules:                createModules,
		AverageItemSizeInBytes: redis.Int(averageItemSizeInBytes),
		Alerts:                 createAlerts,
	}

	dbId, err := api.client.Database.Create(ctx, subId, createDatabase)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(dbId))

	// Confirm Subscription Active status
	err = waitForDatabaseToBeActive(ctx, subId, dbId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	// Locate Databases to confirm Active status

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	return resourceRedisCloudSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	dbId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	subId := d.Get("subscription_id").(int)

	//subscription, err := api.client.Subscription.Get(ctx, subId)
	db, err := api.client.Database.Get(ctx, subId, dbId)
	if err != nil {
		return diag.FromErr(err)
	}

	// Sets

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

	if err := d.Set("data_persistence", redis.StringValue(db.DataPersistence)); err != nil {
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

	if err := d.Set("module", flattenModules(db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("alert", flattenAlerts(db.Alerts)); err != nil {
		return diag.FromErr(err)
	}

	//if err := d.Set("external_endpoint_for_oss_cluster_api"); err != nil {
	//	return diag.FromErr(err)
	//}

	//if redis.StringValue(db.Protocol) == "redis" {
	//	// TODO need to check if this is expected behaviour or not
	//	password = redis.StringValue(db.Security.Password)
	//}
	if err := d.Set("password", db.Security.Password); err != nil {
		return diag.FromErr(err)
	}

	var sourceIPs []string
	if len(db.Security.SourceIPs) == 1 && redis.StringValue(db.Security.SourceIPs[0]) == "0.0.0.0/0" {
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

	return diags
}

func resourceRedisCloudDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	//api := meta.(*apiClient)

	//subId, err := strconv.Atoi(d.Id())
	//if err != nil {
	//	return diag.FromErr(err)
	//}
	//
	//subscriptionMutex.Lock(subId)
	//defer subscriptionMutex.Unlock(subId)
	//
	//if d.HasChange("allowlist") {
	//	cidrs := setToStringSlice(d.Get("allowlist.0.cidrs").(*schema.Set))
	//	sgs := setToStringSlice(d.Get("allowlist.0.security_group_ids").(*schema.Set))
	//
	//	err := api.client.Subscription.UpdateCIDRAllowlist(ctx, subId, subscriptions.UpdateCIDRAllowlist{
	//		CIDRIPs:          cidrs,
	//		SecurityGroupIDs: sgs,
	//	})
	//	if err != nil {
	//		return diag.FromErr(err)
	//	}
	//}
	//
	//if d.HasChanges("name", "payment_method_id") {
	//	updateSubscriptionRequest := subscriptions.UpdateSubscription{}
	//
	//	if d.HasChange("name") {
	//		name := d.Get("name").(string)
	//		updateSubscriptionRequest.Name = &name
	//	}
	//
	//	if d.HasChange("payment_method_id") {
	//		paymentMethodID, err := readPaymentMethodID(d)
	//		if err != nil {
	//			return diag.FromErr(err)
	//		}
	//
	//		updateSubscriptionRequest.PaymentMethodID = paymentMethodID
	//	}
	//
	//	err = api.client.Subscription.Update(ctx, subId, updateSubscriptionRequest)
	//	if err != nil {
	//		return diag.FromErr(err)
	//	}
	//}
	//
	//if d.HasChange("database") || d.IsNewResource() {
	//	oldDb, newDb := d.GetChange("database")
	//	addition, existing, deletion := diff(oldDb.(*schema.Set), newDb.(*schema.Set), func(v interface{}) string {
	//		m := v.(map[string]interface{})
	//		return m["name"].(string)
	//	})
	//
	//	if d.IsNewResource() {
	//		// Terraform will report all of the databases that were just created in resourceRedisCloudSubscriptionCreate
	//		// as newly added, but they have been created by the create subscription call. All that needs to happen to
	//		// them is to be updated like 'normal' existing databases.
	//		existing = addition
	//	} else {
	//		// this is not a new resource, so these databases really do new to be created
	//		for _, db := range addition {
	//			// This loop with addition is triggered when another database is added to the subscription.
	//			request := buildCreateDatabase(db)
	//			id, err := api.client.Database.Create(ctx, subId, request)
	//			if err != nil {
	//				return diag.FromErr(err)
	//			}
	//
	//			log.Printf("[DEBUG] Created database %d", id)
	//
	//			if err := waitForDatabaseToBeActive(ctx, subId, id, api); err != nil {
	//				return diag.FromErr(err)
	//			}
	//		}
	//
	//		// Certain values - like the hashing policy - can only be set on an update, so the newly created databases
	//		// need to be updated straight away
	//		existing = append(existing, addition...)
	//	}
	//
	//	nameId, err := getDatabaseNameIdMap(ctx, subId, api)
	//	if err != nil {
	//		return diag.FromErr(err)
	//	}
	//
	//	for _, db := range existing {
	//		update := buildUpdateDatabase(db)
	//		dbId := nameId[redis.StringValue(update.Name)]
	//
	//		log.Printf("[DEBUG] Updating database %s (%d)", redis.StringValue(update.Name), dbId)
	//
	//		err = api.client.Database.Update(ctx, subId, dbId, update)
	//		if err != nil {
	//			return diag.FromErr(err)
	//		}
	//
	//		if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
	//			return diag.FromErr(err)
	//		}
	//	}
	//
	//	for _, db := range deletion {
	//		name := db["name"].(string)
	//		id := nameId[name]
	//
	//		log.Printf("[DEBUG] Deleting database %s (%d)", name, id)
	//
	//		err = api.client.Database.Delete(ctx, subId, id)
	//		if err != nil {
	//			return diag.FromErr(err)
	//		}
	//	}
	//}
	//
	//if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
	//	return diag.FromErr(err)
	//}

	return resourceRedisCloudSubscriptionRead(ctx, d, meta)
}

func buildCreateDatabases(databases interface{}) []*subscriptions.CreateDatabase {
	createDatabases := make([]*subscriptions.CreateDatabase, 0)

	for _, database := range databases.(*schema.Set).List() {
		databaseMap := database.(map[string]interface{})

		name := databaseMap["name"].(string)
		protocol := databaseMap["protocol"].(string)
		memoryLimitInGB := databaseMap["memory_limit_in_gb"].(float64)
		supportOSSClusterAPI := databaseMap["support_oss_cluster_api"].(bool)
		dataPersistence := databaseMap["data_persistence"].(string)
		replication := databaseMap["replication"].(bool)
		throughputMeasurementBy := databaseMap["throughput_measurement_by"].(string)
		throughputMeasurementValue := databaseMap["throughput_measurement_value"].(int)
		averageItemSizeInBytes := databaseMap["average_item_size_in_bytes"].(int)

		createModules := make([]*subscriptions.CreateModules, 0)
		modules := databaseMap["module"].(*schema.Set)
		for _, module := range modules.List() {
			moduleMap := module.(map[string]interface{})

			modName := moduleMap["name"].(string)

			createModule := &subscriptions.CreateModules{
				Name: redis.String(modName),
			}

			createModules = append(createModules, createModule)
		}

		createDatabase := &subscriptions.CreateDatabase{
			Name:                 redis.String(name),
			Protocol:             redis.String(protocol),
			MemoryLimitInGB:      redis.Float64(memoryLimitInGB),
			SupportOSSClusterAPI: redis.Bool(supportOSSClusterAPI),
			DataPersistence:      redis.String(dataPersistence),
			Replication:          redis.Bool(replication),
			ThroughputMeasurement: &subscriptions.CreateThroughput{
				By:    redis.String(throughputMeasurementBy),
				Value: redis.Int(throughputMeasurementValue),
			},
			Quantity: redis.Int(1),
			Modules:  createModules,
		}

		if averageItemSizeInBytes > 0 {
			createDatabase.AverageItemSizeInBytes = &averageItemSizeInBytes
		}

		createDatabases = append(createDatabases, createDatabase)
	}

	return createDatabases
}
