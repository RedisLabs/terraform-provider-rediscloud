package provider

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const CMK_ENABLED_STRING = "customer-managed-key"

func containsModule(modules []interface{}, requiredModule string) bool {
	for _, m := range modules {
		if mod, ok := m.(string); ok && mod == requiredModule {
			return true
		}
	}
	return false
}

func resourceRedisCloudProSubscription() *schema.Resource {
	return &schema.Resource{

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {

			// Ensure the "creation_plan" block exists
			_, creationPlanExists := diff.GetOk("creation_plan")
			if !creationPlanExists {
				if diff.Id() == "" {
					return fmt.Errorf(`the "creation_plan" block is required`)
				}
				return nil
			}

			// Validate "query_performance_factor" dependency on "modules"
			creationPlan := diff.Get("creation_plan").([]interface{})
			if len(creationPlan) > 0 {
				plan := creationPlan[0].(map[string]interface{})

				qpf, qpfExists := plan["query_performance_factor"].(string)

				// Ensure "modules" key is explicitly defined in HCL
				_, modulesExists := diff.GetOkExists("creation_plan.0.modules")

				if qpfExists && qpf != "" {
					if !modulesExists {
						return fmt.Errorf(`"query_performance_factor" requires the "modules" key to be explicitly defined in HCL`)
					}

					modules, _ := plan["modules"].([]interface{})
					if !containsModule(modules, "RediSearch") {
						return fmt.Errorf(`"query_performance_factor" requires the "modules" list to contain "RediSearch"`)
					}
				}
			}

			//err := cloudRegionsForceNewDiff(ctx, diff, meta)
			//if err != nil {
			//	return err
			//}

			return nil
		},

		Description:   "Creates a Pro Subscription within your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudProSubscriptionCreate,
		ReadContext:   resourceRedisCloudProSubscriptionRead,
		UpdateContext: resourceRedisCloudProSubscriptionUpdate,
		DeleteContext: resourceRedisCloudProSubscriptionDelete,

		Importer: &schema.ResourceImporter{
			// Let the READ operation do the heavy lifting for importing values from the API.
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the subscription",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"payment_method": {
				Description:      "Payment method for the requested subscription. If credit card is specified, the payment method id must be defined. This information is only used when creating a new subscription and any changes will be ignored after this.",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^(credit-card|marketplace)$"), "must be 'credit-card' or 'marketplace'")),
				Optional:         true,
				Default:          "credit-card",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Id() == "" {
						// We don't want to ignore the block if the resource is about to be created.
						return false
					}
					return true
				},
			},
			"payment_method_id": {
				Computed:         true,
				Description:      "A valid payment method pre-defined in the current account",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				Optional:         true,
			},
			"memory_storage": {
				Description:      "Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’",
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Default:          "ram",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.MemoryStorageValues(), false)),
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
								ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
							},
						},
						"security_group_ids": {
							Description: "Set of security groups that are allowed to access the databases associated with this subscription",
							Type:        schema.TypeSet,
							Required:    true,
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
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
						},
						"cloud_account_id": {
							Description:      "Cloud account identifier. Default: Redis Labs internal cloud account (using Cloud Account Id = 1 implies using Redis Labs internal cloud account). Note that a GCP subscription can be created only with Redis Labs internal cloud account",
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
							Default:          "1",
						},
						"region": {
							Description: "Cloud networking details, per region (single region or multiple regions for Active-Active cluster only)",
							Type:        schema.TypeSet,
							Required:    true,
							ForceNew:    false, // custom force new logic
							MinItems:    1,
							Set: func(v interface{}) int {
								var buf bytes.Buffer
								m := v.(map[string]interface{})
								buf.WriteString(fmt.Sprintf("%s-", m["region"].(string)))
								buf.WriteString(fmt.Sprintf("%t-", m["multiple_availability_zones"].(bool)))
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
										Optional:    true,
										ForceNew:    true,
										Computed:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"networking_deployment_cidr": {
										Description: "Deployment CIDR mask",
										Type:        schema.TypeString,
										//ForceNew:         true,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
									},
									"networking_vpc_id": {
										Description: "Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified)",
										Type:        schema.TypeString,
										//ForceNew:    true,
										Optional: true,
										Default:  "",
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
			"creation_plan": {
				Description: "Information about the planned databases used to optimise the database infrastructure. This information is only used when creating a new subscription and any changes will be ignored after this.",
				Type:        schema.TypeList,
				MaxItems:    1,
				// The block is required when the user provisions a new subscription.
				// The block is ignored in the UPDATE operation or after IMPORTing the resource.
				// Custom validation is handled in CustomizeDiff.
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Id() == "" {
						// We don't want to ignore the block if the resource is about to be created.
						return false
					}
					return true
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"memory_limit_in_gb": {
							Description:   "(Deprecated) Maximum memory usage for each database",
							Type:          schema.TypeFloat,
							Optional:      true,
							ConflictsWith: []string{"creation_plan.0.dataset_size_in_gb"},
						},
						"dataset_size_in_gb": {
							Description:   "Maximum amount of data in the dataset for this specific database in GB",
							Type:          schema.TypeFloat,
							Optional:      true,
							ConflictsWith: []string{"creation_plan.0.memory_limit_in_gb"},
						},
						"query_performance_factor": {
							Description: "Query performance factor for this specific database",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
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
							Description:  "Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database",
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							Default:      nil,
							ValidateFunc: validation.IntAtLeast(1),
						},
						"quantity": {
							Description:  "The planned number of databases",
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntAtLeast(1),
						},
						"support_oss_cluster_api": {
							Description: "Support Redis open-source (OSS) Cluster API",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"replication": {
							Description: "Databases replication",
							Type:        schema.TypeBool,
							Required:    true,
						},
						"modules": {
							Description: "Modules that will be used by the databases in this subscription.",
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"redis_version": {
				Description: "Version of Redis to create",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Id() == "" {
						// Consider the property if the resource is about to be created.
						return false
					}

					if old != new {
						// The user is requesting a change
						return false
					}

					return true
				},
			},
			"maintenance_windows": {
				Description: "Specify the subscription's maintenance windows",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MinItems:    1,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Description:      "Either automatic (Redis specified) or manual (User specified)",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"automatic", "manual"}, false)),
						},
						"window": {
							Description: "A list of maintenance windows for manual-mode",
							Type:        schema.TypeList,
							Optional:    true, // if mode==automatic, no windows
							MinItems:    1,    // if mode==manual, need at least 1 window
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"start_hour": {
										Description: "What hour in the day (0-23) may maintenance start",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"duration_in_hours": {
										Description: "How long maintenance may take",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"days": {
										Description: "A list of days on which the window is open ('Monday', 'Tuesday' etc)",
										Type:        schema.TypeList,
										Required:    true,
										MinItems:    1,
										MaxItems:    7,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
			"pricing": {
				Description: "Pricing details totalled over this Subscription",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database_name": {
							Description: "The database this pricing entry applies to",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"type": {
							Description: "The type of cost e.g. 'Shards'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"type_details": {
							Description: "Further detail e.g. 'micro'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"quantity": {
							Description: "Self-explanatory",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"quantity_measurement": {
							Description: "Self-explanatory",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"price_per_unit": {
							Description: "Self-explanatory",
							Type:        schema.TypeFloat,
							Computed:    true,
						},
						"price_currency": {
							Description: "Self-explanatory e.g. 'USD'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"price_period": {
							Description: "Self-explanatory e.g. 'hour'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"region": {
							Description: "Self-explanatory, if the cost is associated with a particular region",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"customer_managed_key_enabled": {
				Description: "Whether to enable CMK (customer managed key) for the subscription. If this is true, then the subscription will be put in a pending state until you supply the CMEK. See documentation for further details on this process. Do not supply a creation plan if this set as true. Defaults to false.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"customer_managed_key_deletion_grace_period": {
				Description: "The grace period for deleting the subscription. If not set, will default to immediate deletion grace period.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "immediate",
			},
			"customer_managed_key": {
				Description: "CMK resources used to encrypt the databases in this subscription. Ignored if `customer_managed_key_enabled` set to false. Supply after the database has been put into database pending state. See documentation for CMK flow.",
				Type:        schema.TypeList,
				Optional:    true,
				Default:     nil,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_name": {
							Description: "Resource name of the customer managed key as defined by the cloud provider.",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"customer_managed_key_redis_service_account": {
				Description: "The principal of the Redis service account that the subscription is created in. This is used by the user to give access to their customer managed key",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

// customised force new logic for CMK
// region changes is not available whilst the subscription is in a pending state
func cloudRegionsForceNewDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	// Add custom force new logic for region changes based on CMK
	// Get the current subscription status
	api := meta.(*apiClient)
	subId, err := strconv.Atoi(diff.Id())
	if err != nil {
		return err
	}

	subscription, err := api.client.Subscription.Get(ctx, subId)
	if err != nil {
		return err
	}

	// Only check for force new if the subscription is not in CMK pending state
	if redis.StringValue(subscription.Status) != subscriptions.SubscriptionStatusEncryptionKeyPending {
		o, n := diff.GetChange("cloud_provider.0.region")
		if o != nil && n != nil && !reflect.DeepEqual(o, n) {
			diff.ForceNew("cloud_provider.0.region")
		}
	}
	return nil
}

func resourceRedisCloudProSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	// Create CloudProviders
	providers, err := buildCreateCloudProviders(d.Get("cloud_provider"))
	if err != nil {
		return diag.FromErr(err)
	}

	// Create Subscription
	name := d.Get("name").(string)

	paymentMethod := d.Get("payment_method").(string)
	paymentMethodID, err := readPaymentMethodID(d)
	if err != nil {
		return diag.FromErr(err)
	}

	memoryStorage := d.Get("memory_storage").(string)

	// Create databases
	plan := d.Get("creation_plan").([]interface{})

	// Create creation-plan databases
	planMap := plan[0].(map[string]interface{})
	dbs, diags := buildSubscriptionCreatePlanDatabases(memoryStorage, planMap)
	if diags.HasError() {
		return diags
	}

	createSubscriptionRequest := subscriptions.CreateSubscription{
		Name:            redis.String(name),
		DryRun:          redis.Bool(false),
		PaymentMethodID: paymentMethodID,
		PaymentMethod:   redis.String(paymentMethod),
		MemoryStorage:   redis.String(memoryStorage),
		CloudProviders:  providers,
		Databases:       dbs,
	}

	redisVersion := d.Get("redis_version").(string)
	if d.Get("redis_version").(string) != "" {
		createSubscriptionRequest.RedisVersion = redis.String(redisVersion)
	}

	isCmk := d.Get("customer_managed_key_enabled").(bool)

	if isCmk == true {
		createSubscriptionRequest.PersistentStorageEncryptionType = redis.String(CMK_ENABLED_STRING)
	}

	log.Printf("createSubscriptionRequest: %v", createSubscriptionRequest)

	subId, err := api.client.Subscription.Create(ctx, createSubscriptionRequest)
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	d.SetId(strconv.Itoa(subId))

	// If we are in a CMK flow then we need to verify a specific state.
	if isCmk == true {
		err = waitForSubscriptionToBeEncryptionKeyPending(ctx, subId, api)
		if err != nil {
			return append(diags, diag.FromErr(err)...)
		}
		return diags
	}

	// Confirm Subscription Active status
	err = waitForSubscriptionToBeActive(ctx, subId, api)

	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	// There is a timing issue where the subscription is marked as active before the creation-plan databases are listed .
	// This additional wait ensures that the databases will be listed before calling api.client.Database.List()
	time.Sleep(30 * time.Second) //lintignore:R018
	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	// Locate Databases to confirm Active status
	dbList := api.client.Database.List(ctx, subId)

	for dbList.Next() {
		dbId := *dbList.Value().ID

		if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
			return append(diags, diag.FromErr(err)...)
		}
		// Delete each creation-plan database
		dbErr := api.client.Database.Delete(ctx, subId, dbId)
		if dbErr != nil {
			diag.FromErr(dbErr)
		}
	}
	if dbList.Err() != nil {
		return append(diags, diag.FromErr(dbList.Err())...)
	}

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	return append(diags, resourceRedisCloudProSubscriptionUpdate(ctx, d, meta)...)
}

func resourceRedisCloudProSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cmkEnabled := d.Get("customer_managed_key_enabled").(bool)

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

	if subscription.PaymentMethodID != nil && redis.IntValue(subscription.PaymentMethodID) != 0 {
		paymentMethodID := strconv.Itoa(redis.IntValue(subscription.PaymentMethodID))
		if err := d.Set("payment_method_id", paymentMethodID); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("payment_method", redis.StringValue(subscription.PaymentMethod)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_storage", redis.StringValue(subscription.MemoryStorage)); err != nil {
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

	if cmkEnabled == false {
		m, err := api.client.Maintenance.Get(ctx, subId)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("maintenance_windows", flattenMaintenance(m)); err != nil {
			return diag.FromErr(err)
		}

		pricingList, err := api.client.Pricing.List(ctx, subId)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("pricing", flattenPricing(pricingList)); err != nil {
			return diag.FromErr(err)
		}
	}

	if cmkEnabled == true {
		if err := d.Set("customer_managed_key_redis_service_account", subscription.CustomerManagedKeyAccessDetails.RedisServiceAccount); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceRedisCloudProSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	subscription, err := api.client.Subscription.Get(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	// CMK flow
	if *subscription.Status == subscriptions.SubscriptionStatusEncryptionKeyPending {
		diags := resourceRedisCloudProSubscriptionUpdateCmk(ctx, d, api, subId)

		if diags != nil {
			return diags
		}

		return resourceRedisCloudProSubscriptionRead(ctx, d, meta)
	}

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
			paymentMethodID, err := readPaymentMethodID(d)
			if err != nil {
				return diag.FromErr(err)
			}

			updateSubscriptionRequest.PaymentMethodID = paymentMethodID
		}

		err = api.client.Subscription.Update(ctx, subId, updateSubscriptionRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("maintenance_windows") {
		var updateMaintenanceRequest maintenance.Maintenance
		if m, ok := d.GetOk("maintenance_windows"); ok {
			mMap := m.([]interface{})[0].(map[string]interface{})

			windows := make([]*maintenance.Window, 0)
			for _, w := range mMap["window"].([]interface{}) {
				wMap := w.(map[string]interface{})
				windows = append(windows, &maintenance.Window{
					StartHour:       redis.Int(wMap["start_hour"].(int)),
					DurationInHours: redis.Int(wMap["duration_in_hours"].(int)),
					Days:            interfaceToStringSlice(wMap["days"].([]interface{})),
				})
			}

			updateMaintenanceRequest = maintenance.Maintenance{
				Mode:    redis.String(mMap["mode"].(string)),
				Windows: windows,
			}
		} else {
			updateMaintenanceRequest = maintenance.Maintenance{
				Mode: redis.String("automatic"),
			}
		}
		err = api.client.Maintenance.Update(ctx, subId, updateMaintenanceRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRedisCloudProSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudProSubscriptionUpdateCmk(ctx context.Context, d *schema.ResourceData, api *apiClient, subId int) diag.Diagnostics {
	cmkResourcesRaw, ok := d.Get("customer_managed_key").([]interface{})
	if !ok {
		return diag.Errorf("invalid type for customer_managed_key")
	}

	if len(cmkResourcesRaw) == 0 {
		return diag.Errorf("customer_managed_key must be set when subscription is in encryption key pending state")
	}

	cmks := buildCMKs(cmkResourcesRaw)
	deletionGracePeriod := d.Get("customer_managed_key_deletion_grace_period").(string)

	updateCmkRequest := subscriptions.UpdateSubscriptionCMKs{
		DeletionGracePeriod: redis.String(deletionGracePeriod),
		CustomerManagedKeys: &cmks,
	}

	if err := api.client.Subscription.UpdateCMKs(ctx, subId, updateCmkRequest); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func buildCMKs(cmkResources []interface{}) []subscriptions.CustomerManagedKey {
	cmks := make([]subscriptions.CustomerManagedKey, 0, len(cmkResources))
	for _, resource := range cmkResources {
		cmkMap := resource.(map[string]interface{})

		cmk := subscriptions.CustomerManagedKey{
			ResourceName: redis.String(cmkMap["resource_name"].(string)),
		}

		cmks = append(cmks, cmk)
	}

	return cmks
}

func resourceRedisCloudProSubscriptionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)
	log.Printf("[DEBUG] Current state before deletion:\n%s", getResourceStateString(d))

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	if cmkEnabled, ok := d.GetOk("customer_managed_key_enabled"); ok && cmkEnabled.(bool) == true {
		if err := waitForSubscriptionToBeEncryptionKeyPending(ctx, subId, api); err != nil {
			return diag.FromErr(err)
		}
	} else {
		// Wait for the subscription to be active before deleting it.
		if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
			return diag.FromErr(err)
		}

		// There is a timing issue where the subscription is marked as active before the creation-plan databases are deleted.
		// This additional wait ensures that the databases are deleted before the subscription is deleted.
		time.Sleep(30 * time.Second) //lintignore:R018
		if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
			return diag.FromErr(err)
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

func getResourceStateString(d *schema.ResourceData) string {
	var stateStr string

	// Add basic resource information
	stateStr += fmt.Sprintf("Resource ID: %s\n", d.Id())

	// Get all attributes
	for k, v := range d.State().Attributes {
		stateStr += fmt.Sprintf("%-40s = %v\n", k, v)
	}

	// Add any known changes
	stateStr += "\nChanges detected:\n"
	for _, k := range d.State().Attributes {
		if d.HasChange(k) {
			old, new := d.GetChange(k)
			stateStr += fmt.Sprintf("%-40s: %v -> %v\n", k, old, new)
		}
	}

	return stateStr
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

func buildSubscriptionCreatePlanDatabases(memoryStorage string, planMap map[string]interface{}) ([]*subscriptions.CreateDatabase, diag.Diagnostics) {

	createDatabases := make([]*subscriptions.CreateDatabase, 0)

	dbName := "creation-plan-db-"
	idx := 1
	throughputMeasurementBy := planMap["throughput_measurement_by"].(string)
	throughputMeasurementValue := planMap["throughput_measurement_value"].(int)
	averageItemSizeInBytes := planMap["average_item_size_in_bytes"].(int)
	numDatabases := planMap["quantity"].(int)
	supportOSSClusterAPI := planMap["support_oss_cluster_api"].(bool)
	replication := planMap["replication"].(bool)
	planModules := interfaceToStringSlice(planMap["modules"].([]interface{}))

	memoryLimitInGB := 0.0
	if v, ok := planMap["memory_limit_in_gb"]; ok && v != nil {
		memoryLimitInGB = v.(float64)
	}

	datasetSizeInGB := 0.0
	if v, ok := planMap["dataset_size_in_gb"]; ok && v != nil {
		datasetSizeInGB = v.(float64)
	}

	queryPerformanceFactor := ""
	if v, ok := planMap["query_performance_factor"]; ok && v != nil {
		queryPerformanceFactor = v.(string)
	}

	var diags diag.Diagnostics
	if memoryStorage == databases.MemoryStorageRam && averageItemSizeInBytes != 0 {
		// TODO This should be changed to an error when releasing 2.0 of the provider
		diags = diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "`average_item_size_in_bytes` not applicable for `ram` memory storage ",
				Detail:   "`average_item_size_in_bytes` is only applicable when `memory_storage` is `ram-and-flash`. This will be an error in a future release of the provider",
			},
		}
	}

	// Check if one of the modules is RedisGraph
	containsGraph := false
	for _, module := range planModules {
		if *module == "RedisGraph" {
			containsGraph = true
			break
		}
	}

	if !containsGraph || len(planModules) <= 1 {
		var modules []*subscriptions.CreateModules
		for _, v := range planModules {
			modules = append(modules, &subscriptions.CreateModules{Name: v})
		}
		createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules, throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, supportOSSClusterAPI, replication, numDatabases, queryPerformanceFactor)...)
	} else {
		// make RedisGraph module the first module, then append the rest of the modules
		var modules []*subscriptions.CreateModules
		modules = append(modules, &subscriptions.CreateModules{Name: redis.String("RedisGraph")})
		for _, v := range planModules {
			if *v != "RedisGraph" {
				modules = append(modules, &subscriptions.CreateModules{Name: v})
			}
		}
		// create a DB with the RedisGraph module
		createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules[:1], throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, supportOSSClusterAPI, replication, 1, queryPerformanceFactor)...)
		if numDatabases == 1 {
			// create one extra DB with all other modules
			createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules[1:], throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, supportOSSClusterAPI, replication, 1, queryPerformanceFactor)...)
		} else if numDatabases > 1 {
			// create the remaining DBs with all other modules
			createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules[1:], throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, supportOSSClusterAPI, replication, numDatabases-1, queryPerformanceFactor)...)
		}
	}
	return createDatabases, diags
}

// createDatabase returns a CreateDatabase struct with the given parameters
func createDatabase(dbName string, idx *int, modules []*subscriptions.CreateModules, throughputMeasurementBy string, throughputMeasurementValue int, memoryLimitInGB float64, datasetSizeInGB float64, averageItemSizeInBytes int, supportOSSClusterAPI bool, replication bool, numDatabases int, queryPerformanceFactor string) []*subscriptions.CreateDatabase {
	createThroughput := &subscriptions.CreateThroughput{
		By:    redis.String(throughputMeasurementBy),
		Value: redis.Int(throughputMeasurementValue),
	}
	if len(modules) > 0 {
		// if RedisGraph is in the modules, set throughput to operations-per-second and convert the value
		if *modules[0].Name == "RedisGraph" {
			if *createThroughput.By == "number-of-shards" {
				createThroughput.By = redis.String("operations-per-second")
				if replication {
					createThroughput.Value = redis.Int(*createThroughput.Value * 500)
				} else {
					createThroughput.Value = redis.Int(*createThroughput.Value * 250)
				}
			}
		}
	}
	var dbs []*subscriptions.CreateDatabase
	for i := 0; i < numDatabases; i++ {
		createDatabase := subscriptions.CreateDatabase{
			Name:                  redis.String(dbName + strconv.Itoa(*idx)),
			Protocol:              redis.String("redis"),
			SupportOSSClusterAPI:  redis.Bool(supportOSSClusterAPI),
			Replication:           redis.Bool(replication),
			ThroughputMeasurement: createThroughput,
			Quantity:              redis.Int(1),
			Modules:               modules,
		}
		if averageItemSizeInBytes > 0 {
			createDatabase.AverageItemSizeInBytes = redis.Int(averageItemSizeInBytes)
		}

		if datasetSizeInGB > 0 {
			createDatabase.DatasetSizeInGB = redis.Float64(datasetSizeInGB)
		}
		if memoryLimitInGB > 0 {
			createDatabase.MemoryLimitInGB = redis.Float64(memoryLimitInGB)
		}

		if queryPerformanceFactor != "" {
			createDatabase.QueryPerformanceFactor = redis.String(queryPerformanceFactor)
		}

		*idx++
		dbs = append(dbs, &createDatabase)
	}
	return dbs
}

func waitForSubscriptionToBeActive(ctx context.Context, id int, api *apiClient) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusPending},
		Target:       []string{subscriptions.SubscriptionStatusActive},
		Timeout:      safetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be %s", id, subscriptions.SubscriptionStatusActive)

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

func waitForSubscriptionToBeEncryptionKeyPending(ctx context.Context, id int, api *apiClient) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusPending},
		Target:       []string{subscriptions.SubscriptionStatusEncryptionKeyPending},
		Timeout:      safetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be %s", id, subscriptions.SubscriptionStatusEncryptionKeyPending)

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
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusDeleting},
		Target:       []string{"deleted"}, // TODO: update this with deleted field in SDK
		Timeout:      safetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be deleted", id)

			subscription, err := api.client.Subscription.Get(ctx, id)
			if err != nil {
				if _, ok := err.(*subscriptions.NotFound); ok {
					return "deleted", "deleted", nil
				} // TODO: update this with deleted field in SDK
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
	wait := &retry.StateChangeConf{
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
		Timeout:      safetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

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
				if len(currentRegion.Networking) > 0 && !redis.BoolValue(currentRegion.MultipleAvailabilityZones) {
					regionMapString["networking_deployment_cidr"] = currentRegion.Networking[0].DeploymentCIDR
				} else {
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

func readPaymentMethodID(d *schema.ResourceData) (*int, error) {
	pmID := d.Get("payment_method_id").(string)
	if pmID != "" {
		pmID, err := strconv.Atoi(pmID)
		if err != nil {
			return nil, err
		}
		return redis.Int(pmID), nil
	}
	return nil, nil
}

func flattenPricing(pricing []*pricing.Pricing) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)
	for _, p := range pricing {

		tf := map[string]interface{}{
			"database_name":        p.DatabaseName,
			"type":                 p.Type,
			"type_details":         p.TypeDetails,
			"quantity":             p.Quantity,
			"quantity_measurement": p.QuantityMeasurement,
			"price_per_unit":       p.PricePerUnit,
			"price_currency":       p.PriceCurrency,
			"price_period":         p.PricePeriod,
			"region":               p.Region,
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func flattenMaintenance(m *maintenance.Maintenance) []map[string]interface{} {
	var windows []map[string]interface{}
	for _, w := range m.Windows {
		tfw := map[string]interface{}{
			"start_hour":        w.StartHour,
			"duration_in_hours": w.DurationInHours,
			"days":              w.Days,
		}
		windows = append(windows, tfw)
	}

	tf := map[string]interface{}{
		"mode":   m.Mode,
		"window": windows,
	}

	return []map[string]interface{}{tf}
}
