package provider

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudSubscription() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates a Subscription and database resources within your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudSubscriptionCreate,
		ReadContext:   resourceRedisCloudSubscriptionRead,
		UpdateContext: resourceRedisCloudSubscriptionUpdate,
		DeleteContext: resourceRedisCloudSubscriptionDelete,

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
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the subscription",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"payment_method_id": {
				Description:      "A valid payment method pre-defined in the current account",
				Type:             schema.TypeString,
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				Optional:         true,
			},
			"memory_storage": {
				Description:      "Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’",
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Default:          "ram",
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.MemoryStorageValues(), false)),
			},
			"persistent_storage_encryption": {
				Description: "Encrypt data stored in persistent storage. Required for a GCP subscription",
				Type:        schema.TypeBool,
				ForceNew:    true,
				Optional:    true,
				Default:     true,
			},
			"allowlist": {
				Description: "An allowlist object",
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidrs": {
							Description: "Set of CIDR ranges that are allowed to access the databases associated with this subscription",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
							},
						},
						"security_group_ids": {
							Description: "Set of security groups that are allowed to access the databases associated with this subscription",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"cloud_provider": {
				Description: "A cloud provider object",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Description:      "The cloud provider to use with the subscription, (either `AWS` or `GCP`)",
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							Default:          "AWS",
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
						},
						"cloud_account_id": {
							Description:      "Cloud account identifier. Default: Redis Labs internal cloud account (using Cloud Account Id = 1 implies using Redis Labs internal cloud account). Note that a GCP subscription can be created only with Redis Labs internal cloud account",
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
							Default:          "1",
						},
						"region": {
							Description: "Cloud networking details, per region (single region or multiple regions for Active-Active cluster only)",
							Type:        schema.TypeSet,
							Required:    true,
							ForceNew:    true,
							MinItems:    1,
							Set: func(v interface{}) int {
								var buf bytes.Buffer
								m := v.(map[string]interface{})
								buf.WriteString(fmt.Sprintf("%s-", m["region"].(string)))
								buf.WriteString(fmt.Sprintf("%t-", m["multiple_availability_zones"].(bool)))
								buf.WriteString(fmt.Sprintf("%s-", m["preferred_availability_zones"].([]interface{})))
								if v, ok := m["multiple_availability_zones"].(bool); ok && !v {
									buf.WriteString(fmt.Sprintf("%s-", m["networking_deployment_cidr"].(string)))
								}

								return schema.HashString(buf.String())
							},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Description: "Deployment region as defined by cloud provider",
										Type:        schema.TypeString,
										Required:    true,
										ForceNew:    true,
									},
									"multiple_availability_zones": {
										Description: "Support deployment on multiple availability zones within the selected region",
										Type:        schema.TypeBool,
										ForceNew:    true,
										Optional:    true,
										Default:     false,
									},
									"preferred_availability_zones": {
										Description: "List of availability zones used",
										Type:        schema.TypeList,
										Required:    true,
										ForceNew:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"networking_deployment_cidr": {
										Description:      "Deployment CIDR mask",
										Type:             schema.TypeString,
										ForceNew:         true,
										Required:         true,
										ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
									},
									"networking_vpc_id": {
										Description: "Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified)",
										Type:        schema.TypeString,
										ForceNew:    true,
										Optional:    true,
										Default:     "",
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
													Description: "Deployment CIDR mask",
													Type:        schema.TypeString,
													Computed:    true,
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
			"database": {
				Description: "A database object",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"db_id": {
							Description: "Identifier of the database created",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"name": {
							Description: "A meaningful name to identify the database",
							Type:        schema.TypeString,
							Required:    true,
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
							Type:        schema.TypeList,
							Optional:    true,
							MinItems:    1,
							MaxItems:    1,
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
					},
				},
			},
		},
	}
}

func resourceRedisCloudSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	// Create CloudProviders
	providers, err := buildCreateCloudProviders(d.Get("cloud_provider"))
	if err != nil {
		return diag.FromErr(err)
	}

	// Create databases
	dbs := buildSubscriptionCreateDatabases(d.Get("database"))

	// Create Subscription
	name := d.Get("name").(string)
	paymentMethodID, err := strconv.Atoi(d.Get("payment_method_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	memoryStorage := d.Get("memory_storage").(string)
	persistentStorageEncryption := d.Get("persistent_storage_encryption").(bool)

	createSubscriptionRequest := subscriptions.CreateSubscription{
		Name:                        redis.String(name),
		DryRun:                      redis.Bool(false),
		PaymentMethodID:             redis.Int(paymentMethodID),
		MemoryStorage:               redis.String(memoryStorage),
		PersistentStorageEncryption: redis.Bool(persistentStorageEncryption),
		CloudProviders:              providers,
		Databases:                   dbs,
	}

	subId, err := api.client.Subscription.Create(ctx, createSubscriptionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	// Confirm Subscription Active status
	err = waitForSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	// Locate Databases to confirm Active status
	dbList := api.client.Database.List(ctx, subId)

	for dbList.Next() {
		dbId := *dbList.Value().ID

		if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
			return diag.FromErr(err)
		}
	}
	if dbList.Err() != nil {
		return diag.FromErr(dbList.Err())
	}

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	return resourceRedisCloudSubscriptionUpdate(ctx, d, meta)
}

func resourceRedisCloudSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscription, err := api.client.Subscription.Get(ctx, subId)
	if err != nil {
		if _, ok := err.(*subscriptions.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(subscription.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method_id", strconv.Itoa(redis.IntValue(subscription.PaymentMethodID))); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_storage", redis.StringValue(subscription.MemoryStorage)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("persistent_storage_encryption", redis.BoolValue(subscription.StorageEncryption)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("cloud_provider", flattenCloudDetails(subscription.CloudDetails, true)); err != nil {
		return diag.FromErr(err)
	}

	providers, err := buildCreateCloudProviders(d.Get("cloud_provider"))
	if err != nil {
		return diag.FromErr(err)
	}

	// CIDR allowlist is not allowed for Redis Labs internal resources subscription.
	if len(providers) > 0 && redis.IntValue(providers[0].CloudAccountID) != 1 {
		allowlist, err := flattenSubscriptionAllowlist(ctx, subId, api)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("allowlist", allowlist); err != nil {
			return diag.FromErr(err)
		}
	}

	flatDbs, err := flattenDatabases(ctx, subId, d.Get("database").(*schema.Set).List(), api)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("database", flatDbs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	if d.HasChange("allowlist") {
		cidrs := setToStringSlice(d.Get("allowlist.0.cidrs").(*schema.Set))
		sgs := setToStringSlice(d.Get("allowlist.0.security_group_ids").(*schema.Set))

		err := api.client.Subscription.UpdateCIDRAllowlist(ctx, subId, subscriptions.UpdateCIDRAllowlist{
			CIDRIPs:          cidrs,
			SecurityGroupIDs: sgs,
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChanges("name", "payment_method_id") {
		updateSubscriptionRequest := subscriptions.UpdateSubscription{}

		if d.HasChange("name") {
			name := d.Get("name").(string)
			updateSubscriptionRequest.Name = &name
		}

		if d.HasChange("payment_method_id") {
			paymentMethodID, err := strconv.Atoi(d.Get("payment_method_id").(string))
			if err != nil {
				return diag.FromErr(err)
			}

			updateSubscriptionRequest.PaymentMethodID = &paymentMethodID
		}

		err = api.client.Subscription.Update(ctx, subId, updateSubscriptionRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("database") || d.IsNewResource() {
		oldDb, newDb := d.GetChange("database")
		addition, existing, deletion := diff(oldDb.(*schema.Set), newDb.(*schema.Set), func(v interface{}) string {
			m := v.(map[string]interface{})
			return m["name"].(string)
		})

		if d.IsNewResource() {
			// Terraform will report all of the databases that were just created in resourceRedisCloudSubscriptionCreate
			// as newly added, but they have been created by the create subscription call. All that needs to happen to
			// them is to be updated like 'normal' existing databases.
			existing = addition
		} else {
			// this is not a new resource, so these databases really do new to be created
			for _, db := range addition {
				request := buildCreateDatabase(db)
				id, err := api.client.Database.Create(ctx, subId, request)
				if err != nil {
					return diag.FromErr(err)
				}

				log.Printf("[DEBUG] Created database %d", id)

				if err := waitForDatabaseToBeActive(ctx, subId, id, api); err != nil {
					return diag.FromErr(err)
				}
			}

			// Certain values - like the hashing policy - can only be set on an update, so the newly created databases
			// need to be updated straight away
			existing = append(existing, addition...)
		}

		nameId, err := getDatabaseNameIdMap(ctx, subId, api)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, db := range existing {
			update := buildUpdateDatabase(db)
			dbId := nameId[redis.StringValue(update.Name)]

			log.Printf("[DEBUG] Updating database %s (%d)", redis.StringValue(update.Name), dbId)

			err = api.client.Database.Update(ctx, subId, dbId, update)
			if err != nil {
				return diag.FromErr(err)
			}

			if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
				return diag.FromErr(err)
			}
		}

		for _, db := range deletion {
			name := db["name"].(string)
			id := nameId[name]

			log.Printf("[DEBUG] Deleting database %s (%d)", name, id)

			err = api.client.Database.Delete(ctx, subId, id)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudSubscriptionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	nameId, err := getDatabaseNameIdMap(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, v := range d.Get("database").(*schema.Set).List() {
		database := v.(map[string]interface{})

		name := database["name"].(string)
		if id, ok := nameId[name]; ok {
			log.Printf("[DEBUG] Deleting database %d on subscription %d", id, subId)

			dbErr := api.client.Database.Delete(ctx, subId, id)
			if dbErr != nil {
				diag.FromErr(dbErr)
			}
		} else {
			log.Printf("[DEBUG] Database %s no longer exists", name)
		}
	}

	// Delete subscription once all databases are deleted
	err = api.client.Subscription.Delete(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	err = waitForSubscriptionToBeDeleted(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func buildCreateCloudProviders(providers interface{}) ([]*subscriptions.CreateCloudProvider, error) {
	createCloudProviders := make([]*subscriptions.CreateCloudProvider, 0)

	for _, provider := range providers.([]interface{}) {
		providerMap := provider.(map[string]interface{})

		providerStr := providerMap["provider"].(string)
		cloudAccountID, err := strconv.Atoi(providerMap["cloud_account_id"].(string))
		if err != nil {
			return nil, err
		}

		createRegions := make([]*subscriptions.CreateRegion, 0)
		if regions := providerMap["region"].(*schema.Set).List(); len(regions) != 0 {

			for _, region := range regions {
				regionMap := region.(map[string]interface{})

				regionStr := regionMap["region"].(string)
				multipleAvailabilityZones := regionMap["multiple_availability_zones"].(bool)
				preferredAZs := regionMap["preferred_availability_zones"].([]interface{})

				createRegion := subscriptions.CreateRegion{
					Region:                     redis.String(regionStr),
					MultipleAvailabilityZones:  redis.Bool(multipleAvailabilityZones),
					PreferredAvailabilityZones: interfaceToStringSlice(preferredAZs),
				}

				if v, ok := regionMap["networking_deployment_cidr"]; ok && v != "" {
					createRegion.Networking = &subscriptions.CreateNetworking{
						DeploymentCIDR: redis.String(v.(string)),
					}
				}

				if v, ok := regionMap["networking_vpc_id"]; ok && v != "" {
					if createRegion.Networking == nil {
						createRegion.Networking = &subscriptions.CreateNetworking{}
					}
					createRegion.Networking.VPCId = redis.String(v.(string))
				}

				createRegions = append(createRegions, &createRegion)
			}
		}

		createCloudProvider := &subscriptions.CreateCloudProvider{
			Provider:       redis.String(providerStr),
			CloudAccountID: redis.Int(cloudAccountID),
			Regions:        createRegions,
		}

		createCloudProviders = append(createCloudProviders, createCloudProvider)
	}

	return createCloudProviders, nil
}

func buildSubscriptionCreateDatabases(databases interface{}) []*subscriptions.CreateDatabase {
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
		modules := databaseMap["module"]
		for _, module := range modules.([]interface{}) {
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

func buildCreateDatabase(db map[string]interface{}) databases.CreateDatabase {
	var alerts []*databases.CreateAlert
	for _, alert := range db["alert"].(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		alerts = append(alerts, &databases.CreateAlert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	create := databases.CreateDatabase{
		DryRun:               redis.Bool(false),
		Name:                 redis.String(db["name"].(string)),
		Protocol:             redis.String(db["protocol"].(string)),
		MemoryLimitInGB:      redis.Float64(db["memory_limit_in_gb"].(float64)),
		SupportOSSClusterAPI: redis.Bool(db["support_oss_cluster_api"].(bool)),
		DataPersistence:      redis.String(db["data_persistence"].(string)),
		Replication:          redis.Bool(db["replication"].(bool)),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    redis.String(db["throughput_measurement_by"].(string)),
			Value: redis.Int(db["throughput_measurement_value"].(int)),
		},
		Alerts:    alerts,
		ReplicaOf: setToStringSlice(db["replica_of"].(*schema.Set)),
		Password:  redis.String(db["password"].(string)),
		SourceIP:  setToStringSlice(db["source_ips"].(*schema.Set)),
	}

	averageItemSize := db["average_item_size_in_bytes"].(int)
	if averageItemSize > 0 {
		create.AverageItemSizeInBytes = redis.Int(averageItemSize)
	}

	clientSSLCertificate := db["client_ssl_certificate"].(string)
	if clientSSLCertificate != "" {
		create.ClientSSLCertificate = redis.String(clientSSLCertificate)
	}

	backupPath := db["periodic_backup_path"].(string)
	if backupPath != "" {
		create.PeriodicBackupPath = redis.String(backupPath)
	}

	if v, ok := db["external_endpoint_for_oss_cluster_api"]; ok {
		create.UseExternalEndpointForOSSClusterAPI = redis.Bool(v.(bool))
	}

	return create
}

func buildUpdateDatabase(db map[string]interface{}) databases.UpdateDatabase {
	var alerts []*databases.UpdateAlert
	for _, alert := range db["alert"].(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		alerts = append(alerts, &databases.UpdateAlert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	update := databases.UpdateDatabase{
		Name:                 redis.String(db["name"].(string)),
		MemoryLimitInGB:      redis.Float64(db["memory_limit_in_gb"].(float64)),
		SupportOSSClusterAPI: redis.Bool(db["support_oss_cluster_api"].(bool)),
		Replication:          redis.Bool(db["replication"].(bool)),
		ThroughputMeasurement: &databases.UpdateThroughputMeasurement{
			By:    redis.String(db["throughput_measurement_by"].(string)),
			Value: redis.Int(db["throughput_measurement_value"].(int)),
		},
		DataPersistence: redis.String(db["data_persistence"].(string)),
		Password:        redis.String(db["password"].(string)),
		SourceIP:        setToStringSlice(db["source_ips"].(*schema.Set)),
		Alerts:          alerts,
		ReplicaOf:       setToStringSlice(db["replica_of"].(*schema.Set)),
	}

	clientSSLCertificate := db["client_ssl_certificate"].(string)
	if clientSSLCertificate != "" {
		update.ClientSSLCertificate = redis.String(clientSSLCertificate)
	}

	regex := db["hashing_policy"].([]interface{})
	if len(regex) != 0 {
		update.RegexRules = interfaceToStringSlice(regex)
	}

	backupPath := db["periodic_backup_path"].(string)
	if backupPath != "" {
		update.PeriodicBackupPath = redis.String(backupPath)
	}

	if v, ok := db["external_endpoint_for_oss_cluster_api"]; ok {
		update.UseExternalEndpointForOSSClusterAPI = redis.Bool(v.(bool))
	}

	return update
}

func waitForSubscriptionToBeActive(ctx context.Context, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusPending},
		Target:  []string{subscriptions.SubscriptionStatusActive},
		Timeout: 20 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be active", id)

			subscription, err := api.client.Subscription.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForSubscriptionToBeDeleted(ctx context.Context, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusDeleting},
		Target:  []string{"deleted"},
		Timeout: 10 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be deleted", id)

			subscription, err := api.client.Subscription.Get(ctx, id)
			if err != nil {
				if _, ok := err.(*subscriptions.NotFound); ok {
					return "deleted", "deleted", nil
				}
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForDatabaseToBeActive(ctx context.Context, subId, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay: 10 * time.Second,
		Pending: []string{
			databases.StatusDraft,
			databases.StatusPending,
			databases.StatusActiveChangePending,
			databases.StatusRCPActiveChangeDraft,
			databases.StatusActiveChangeDraft,
			databases.StatusRCPDraft,
			databases.StatusRCPChangePending,
		},
		Target:  []string{databases.StatusActive},
		Timeout: 10 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for database %d to be active", id)

			database, err := api.client.Database.Get(ctx, subId, id)
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

func flattenSubscriptionAllowlist(ctx context.Context, subId int, api *apiClient) ([]map[string]interface{}, error) {
	allowlist, err := api.client.Subscription.GetCIDRAllowlist(ctx, subId)
	if err != nil {
		return nil, err
	}

	if !isNil(allowlist.Errors) {
		return nil, fmt.Errorf("unable to read allowlist for subscription %d: %v", subId, allowlist.Errors)
	}

	var cidrs []string
	for _, cidr := range allowlist.CIDRIPs {
		cidrs = append(cidrs, redis.StringValue(cidr))
	}
	var sgs []string
	for _, sg := range allowlist.SecurityGroupIDs {
		sgs = append(sgs, redis.StringValue(sg))
	}

	tfs := map[string]interface{}{}

	if len(cidrs) != 0 {
		tfs["cidrs"] = cidrs
	}
	if len(sgs) != 0 {
		tfs["security_group_ids"] = sgs
	}
	if len(tfs) == 0 {
		return nil, nil
	}

	return []map[string]interface{}{tfs}, nil
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}

	if l, ok := i.([]interface{}); ok {
		if len(l) == 0 {
			return true
		}
	}

	if m, ok := i.(map[string]interface{}); ok {
		if len(m) == 0 {
			return true
		}
	}

	return false
}

func flattenCloudDetails(cloudDetails []*subscriptions.CloudDetail, isResource bool) []map[string]interface{} {
	var cdl []map[string]interface{}

	for _, currentCloudDetail := range cloudDetails {

		var regions []interface{}
		for _, currentRegion := range currentCloudDetail.Regions {

			regionMapString := map[string]interface{}{
				"region":                       currentRegion.Region,
				"multiple_availability_zones":  currentRegion.MultipleAvailabilityZones,
				"preferred_availability_zones": currentRegion.PreferredAvailabilityZones,
				"networks":                     flattenNetworks(currentRegion.Networking),
			}

			if isResource {
				regionMapString["networking_deployment_cidr"] = currentRegion.Networking[0].DeploymentCIDR

				if redis.BoolValue(currentRegion.MultipleAvailabilityZones) {
					regionMapString["networking_deployment_cidr"] = ""
				}
			}

			regions = append(regions, regionMapString)
		}

		cdlMapString := map[string]interface{}{
			"provider":         currentCloudDetail.Provider,
			"cloud_account_id": strconv.Itoa(redis.IntValue(currentCloudDetail.CloudAccountID)),
			"region":           regions,
		}
		cdl = append(cdl, cdlMapString)
	}

	return cdl
}

func flattenNetworks(networks []*subscriptions.Networking) []map[string]interface{} {
	var cdl []map[string]interface{}

	for _, currentNetwork := range networks {

		networkMapString := map[string]interface{}{
			"networking_deployment_cidr": currentNetwork.DeploymentCIDR,
			"networking_vpc_id":          currentNetwork.VPCId,
			"networking_subnet_id":       currentNetwork.SubnetID,
		}

		cdl = append(cdl, networkMapString)
	}

	return cdl
}

func flattenDatabases(ctx context.Context, subId int, databases []interface{}, api *apiClient) ([]interface{}, error) {
	nameId, err := getDatabaseNameIdMap(ctx, subId, api)
	if err != nil {
		return nil, err
	}

	var flattened []interface{}
	for _, v := range databases {
		database := v.(map[string]interface{})
		name := database["name"].(string)
		id, ok := nameId[name]
		if !ok {
			log.Printf("database %d not found: %s", id, err)
			continue
		}

		db, err := api.client.Database.Get(ctx, subId, id)
		if err != nil {
			return nil, err
		}

		cert := database["client_ssl_certificate"].(string)
		backupPath := database["periodic_backup_path"].(string)
		averageItemSize := database["average_item_size_in_bytes"].(int)
		existingPassword := database["password"].(string)
		existingSourceIPs := database["source_ips"].(*schema.Set)
		external := database["external_endpoint_for_oss_cluster_api"].(bool)

		flattened = append(flattened, flattenDatabase(cert, external, backupPath, averageItemSize, existingPassword, existingSourceIPs, db))
	}
	return flattened, nil
}

func flattenDatabase(certificate string, externalOSSAPIEndpoint bool, backupPath string, averageItemSizeInBytes int, existingPassword string, existingSourceIp *schema.Set, db *databases.Database) map[string]interface{} {
	password := existingPassword
	if redis.StringValue(db.Protocol) == "redis" {
		// TODO need to check if this is expected behaviour or not
		password = redis.StringValue(db.Security.Password)
	}

	var sourceIPs []string
	if len(db.Security.SourceIPs) == 1 && redis.StringValue(db.Security.SourceIPs[0]) == "0.0.0.0/0" {
		// The API handles an empty list as ["0.0.0.0/0"] but need to be careful to match the input to avoid Terraform detecting drift
		if existingSourceIp.Len() != 0 {
			sourceIPs = redis.StringSliceValue(db.Security.SourceIPs...)
		}
	} else {
		sourceIPs = redis.StringSliceValue(db.Security.SourceIPs...)
	}

	tf := map[string]interface{}{
		"db_id":                                 redis.IntValue(db.ID),
		"name":                                  redis.StringValue(db.Name),
		"protocol":                              redis.StringValue(db.Protocol),
		"memory_limit_in_gb":                    redis.Float64Value(db.MemoryLimitInGB),
		"support_oss_cluster_api":               redis.BoolValue(db.SupportOSSClusterAPI),
		"data_persistence":                      redis.StringValue(db.DataPersistence),
		"replication":                           redis.BoolValue(db.Replication),
		"throughput_measurement_by":             redis.StringValue(db.ThroughputMeasurement.By),
		"throughput_measurement_value":          redis.IntValue(db.ThroughputMeasurement.Value),
		"public_endpoint":                       redis.StringValue(db.PublicEndpoint),
		"private_endpoint":                      redis.StringValue(db.PrivateEndpoint),
		"module":                                flattenModules(db.Modules),
		"alert":                                 flattenAlerts(db.Alerts),
		"external_endpoint_for_oss_cluster_api": externalOSSAPIEndpoint,
		"password":                              password,
		"source_ips":                            sourceIPs,
		"hashing_policy":                        flattenRegexRules(db.Clustering.RegexRules),
	}

	if db.ReplicaOf != nil {
		tf["replica_of"] = redis.StringSliceValue(db.ReplicaOf.Endpoints...)
	}

	if redis.BoolValue(db.Security.SSLClientAuthentication) {
		tf["client_ssl_certificate"] = certificate
	}

	if averageItemSizeInBytes > 0 {
		tf["average_item_size_in_bytes"] = averageItemSizeInBytes
	}

	if backupPath != "" {
		tf["periodic_backup_path"] = backupPath
	}

	return tf
}

func flattenAlerts(alerts []*databases.Alert) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, alert := range alerts {
		tf := map[string]interface{}{
			"name":  redis.StringValue(alert.Name),
			"value": redis.IntValue(alert.Value),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func flattenModules(modules []*databases.Module) []map[string]interface{} {

	var tfs = make([]map[string]interface{}, 0)
	for _, module := range modules {

		tf := map[string]interface{}{
			"name": redis.StringValue(module.Name),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func flattenRegexRules(rules []*databases.RegexRule) []string {
	ret := make([]string, len(rules))
	for _, rule := range rules {
		ret[rule.Ordinal] = rule.Pattern
	}

	if len(ret) == 2 && ret[0] == ".*\\{(?<tag>.*)\\}.*" && ret[1] == "(?<tag>.*)" {
		// This is the default regex rules - https://docs.redislabs.com/latest/rc/concepts/clustering/#custom-hashing-policy
		return []string{}
	}

	return ret
}

func getDatabaseNameIdMap(ctx context.Context, subId int, client *apiClient) (map[string]int, error) {
	ret := map[string]int{}
	list := client.client.Database.List(ctx, subId)
	for list.Next() {
		db := list.Value()
		ret[redis.StringValue(db.Name)] = redis.IntValue(db.ID)
	}
	if list.Err() != nil {
		return nil, list.Err()
	}
	return ret, nil
}

func diff(oldSet *schema.Set, newSet *schema.Set, hashKey func(interface{}) string) ([]map[string]interface{}, []map[string]interface{}, []map[string]interface{}) {
	oldMap := map[string]map[string]interface{}{}
	newMap := map[string]map[string]interface{}{}

	for _, v := range oldSet.List() {
		oldMap[hashKey(v)] = v.(map[string]interface{})
	}
	for _, v := range newSet.List() {
		newMap[hashKey(v)] = v.(map[string]interface{})
	}

	var addition, existing, deletion []map[string]interface{}

	for k, v := range newMap {
		if _, ok := oldMap[k]; ok {
			existing = append(existing, v)
		} else {
			addition = append(addition, v)
		}
	}

	for k, v := range oldMap {
		if _, ok := newMap[k]; !ok {
			deletion = append(deletion, v)
		}
	}

	return addition, existing, deletion
}
