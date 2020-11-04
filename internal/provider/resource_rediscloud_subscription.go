package provider

import (
	"context"
	rediscloud_api "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strconv"
)

func resourceRedisCloudSubscription() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRedisCloudSubscriptionCreate,
		ReadContext:   resourceRedisCloudSubscriptionRead,
		UpdateContext: resourceRedisCloudSubscriptionUpdate,
		DeleteContext: resourceRedisCloudSubscriptionDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dry_run": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"plan_id": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"payment_method_id": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"memory_storage": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "ram",
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"ram", "ram-and-flash"}, false)),
			},
			"persistent_storage_encryption": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cloud_providers": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "AWS",
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"AWS", "GCP"}, false)),
						},
						"cloud_account_id": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1,
						},
						"regions": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Type:     schema.TypeString,
										Required: true,
									},
									"multiple_availability_zones": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"preferred_availability_zones": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"networking_deployment_cidr": {
										Type:             schema.TypeString,
										Optional:         true,
										ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
									},
									"networking_vpc_id": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"databases": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:             schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:             schema.TypeString,
							Required: true,
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"redis", "memcached"}, false)),
						},
						"memory_limit_in_gb": {
							Type:             schema.TypeFloat,
							Required: true,
						},
						"support_oss_cluster_api": {
							Type: schema.TypeBool,
							Optional: true,
							Default: false,
						},
						"data_persistence": {
							Type: schema.TypeString,
							Optional: true,
							Default: "none",
						},
						"replication": {
							Type: schema.TypeBool,
							Optional: true,
							Default: true,
						},
						"throughput_measurement_by": {
							Type: schema.TypeString,
							Required: true,
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"number-of-shards", "operations-per-second"}, false)),
						},
						"throughput_measurement_value": {
							Type:             schema.TypeInt,
							Required: true,
						},
						"modules": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"quantity": {
							Type: schema.TypeInt,
							Optional: true,
							Default: 1,
						},
						"average_item_size_in_bytes": {
							Type: schema.TypeInt,
							Optional: true,
							Default: 1000,
						},
					},
				},
			},
		},
	}
}

func resourceRedisCloudSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*rediscloud_api.Client)

	var diags diag.Diagnostics

	// Create CloudProviders
	providers := make([]subscriptions.CreateCloudProvider, 0)
	cloudProviders := d.Get("cloud_providers").(*schema.Set).List()
	for _, cloudProvider := range cloudProviders {
		cloudProvider := cloudProvider.(map[string]interface{})

		regions := make([]subscriptions.CreateRegion, 0)
		if cloudRegions := cloudProvider["regions"].(*schema.Set).List(); cloudRegions != nil && len(cloudRegions) != 0 {

			for _, region := range cloudRegions {
				mRegion := region.(map[string]interface{})

				createRegion := subscriptions.CreateRegion{
					Region:                     mRegion["region"].(string),
					MultipleAvailabilityZones:  mRegion["multiple_availability_zones"].(bool),
					//PreferredAvailabilityZones: []string{"eu-west-1"},
					Networking:                 &subscriptions.CreateNetworking{
						DeploymentCIDR: mRegion["networking_deployment_cidr"].(string),
						//VPCId:          mRegion["networking_vpc_id"].(string),
					},
				}

				regions = append(regions, createRegion)
			}
		}

		createCloudProvider := subscriptions.CreateCloudProvider{
			Provider:       cloudProvider["provider"].(string),
			CloudAccountId: cloudProvider["cloud_account_id"].(int),
			Regions:        regions,
		}

		providers = append(providers, createCloudProvider)
	}

	// Create databases
	databases := make([]subscriptions.CreateDatabase, 0)
	cloudDatabases := d.Get("databases").(*schema.Set).List()
	for _, cloudDatabase := range cloudDatabases {
		mCloudDatabase := cloudDatabase.(map[string]interface{})

		// TODO - process modules.  Expand the schema to accept name and parameters.

		createDatabases := subscriptions.CreateDatabase{
			Name:                   mCloudDatabase["name"].(string),
			Protocol:               mCloudDatabase["protocol"].(string),
			MemoryLimitInGb:        mCloudDatabase["memory_limit_in_gb"].(float64),
			SupportOSSClusterApi:   mCloudDatabase["support_oss_cluster_api"].(bool),
			DataPersistence:        mCloudDatabase["data_persistence"].(string),
			Replication:            mCloudDatabase["replication"].(bool),
			ThroughputMeasurement:  &subscriptions.CreateThroughput{
				By:    mCloudDatabase["throughput_measurement_by"].(string),
				Value: mCloudDatabase["throughput_measurement_value"].(int),
			},
			Modules:                make([]subscriptions.CreateModules,0),
			Quantity:               mCloudDatabase["quantity"].(int),
			//AverageItemSizeInBytes: mCloudDatabase["average_item_size_in_bytes"].(int),
		}
		databases = append(databases, createDatabases)
	}

	createSubscriptionRequest := subscriptions.CreateSubscription{
		Name:                        d.Get("name").(string),
		DryRun:                      d.Get("dry_run").(bool),
		PaymentMethodId:             d.Get("payment_method_id").(int),
		MemoryStorage:               d.Get("memory_storage").(string),
		PersistentStorageEncryption: d.Get("persistent_storage_encryption").(bool),
		CloudProviders:              providers,
		Databases:                   databases,
	}

	id, err := client.Subscription.Create(ctx, createSubscriptionRequest)
	if err != nil {
		diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))
	return diags
}

func resourceRedisCloudSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	var diags diag.Diagnostics

	return diags
}

func resourceRedisCloudSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return resourceRedisCloudSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudSubscriptionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*rediscloud_api.Client)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		diag.FromErr(err)
	}

	// Locate databases for sub and delete.
	databases, err := client.Database.List(ctx, subId)
	if err != nil {
		diag.FromErr(err)
	}

	for _, database := range databases {

		dbErr := client.Database.Delete(ctx, subId, database.ID)
		if dbErr != nil {
			diag.FromErr(dbErr)
		}
	}

	err = client.Subscription.Delete(ctx, subId)
	if err != nil {
		diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
