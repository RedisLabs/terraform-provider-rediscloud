package provider

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
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
				subId, dbId, err := toSubscriptionId(d.Id())
				if err != nil {
					return nil, err
				}

				// Populate the names of databases that already exist so that `flattenDatabase` can iterate over them
				// The READ operation is triggered after IMPORT, so let it handle flattening the db.
				api := meta.(*apiClient)
				db, err := api.client.Database.Get(ctx, subId, dbId)

				var dbs []map[string]interface{}
				dbs = append(dbs, map[string]interface{}{
					"db_id": redis.Int(*db.ID),
				})

				if err != nil {
					d.SetId("")
					return nil, err
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
			"name": {
				Description: "A meaningful name to identify the subscription",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"payment_method": {
				Description:      "Payment method for the requested subscription. If credit card is specified, the payment method Id must be defined.",
				Type:             schema.TypeString,
				ForceNew:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^(credit-card|marketplace)$"), "must be 'credit-card' or 'marketplace'")),
				Optional:         true,
				Default:          "credit-card",
			},
			"payment_method_id": {
				Computed:         true,
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
			"creation_plan": {
				Description: "Information about the planned databases used to optimise the database infrastructure. This information is only used when creating a new subscription and any changes will be ignored after this.",
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				// TODO: diff suppress func is causing problems (i.e. plan = {})
				// DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				// 	return !(old == "")
				// },
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"memory_limit_in_gb": {
							Description: "Maximum memory usage for each database",
							Type:        schema.TypeFloat,
							Required:    true,
							ForceNew:    true,
						},
						"throughput_measurement_by": {
							Description:      "Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)",
							Type:             schema.TypeString,
							Required:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"number-of-shards", "operations-per-second"}, false)),
						},
						"throughput_measurement_value": {
							Description: "Throughput value (as applies to selected measurement method)",
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    true,
						},
						"average_item_size_in_bytes": {
							Description: "Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database",
							Type:        schema.TypeInt,
							Optional:    true,
							ForceNew:    true,
							// Setting default to 0 so that the hash func produces the same hash when this field is not
							// specified. SDK's catch-all issue around this: https://github.com/hashicorp/terraform-plugin-sdk/issues/261
							Default: 0,
						},
						"quantity": {
							Description: "The planned number of databases",
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    true,
						},
						"support_oss_cluster_api": {
							Description: "Support Redis open-source (OSS) Cluster API",
							Type:        schema.TypeBool,
							Required:    true,
						},
						"replication": {
							Description: "Databases replication",
							Type:        schema.TypeBool,
							Required:    true,
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

	// Create Subscription
	name := d.Get("name").(string)

	paymentMethod := d.Get("payment_method").(string)
	paymentMethodID, err := readPaymentMethodID(d)
	if err != nil {
		return diag.FromErr(err)
	}

	memoryStorage := d.Get("memory_storage").(string)

	// Create databases
	var dbs []*subscriptions.CreateDatabase

	plan := d.Get("creation_plan")
	// Create dummy databases
	dbs = buildSubscriptionCreatePlanDatabases(plan)

	createSubscriptionRequest := subscriptions.CreateSubscription{
		Name:            redis.String(name),
		DryRun:          redis.Bool(false),
		PaymentMethodID: paymentMethodID,
		PaymentMethod:   redis.String(paymentMethod),
		MemoryStorage:   redis.String(memoryStorage),
		CloudProviders:  providers,
		Databases:       dbs,
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
		// Delete each dummy database
		dbErr := api.client.Database.Delete(ctx, subId, dbId)
		if dbErr != nil {
			diag.FromErr(dbErr)
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

	subId, _, err := toSubscriptionId(d.Id())
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

func buildSubscriptionCreatePlanDatabases(plans interface{}) []*subscriptions.CreateDatabase {

	createDatabases := make([]*subscriptions.CreateDatabase, 0)
	planMap := plans.([]interface{})[0].(map[string]interface{})

	memoryLimitInGB := planMap["memory_limit_in_gb"].(float64)
	throughputMeasurementBy := planMap["throughput_measurement_by"].(string)
	throughputMeasurementValue := planMap["throughput_measurement_value"].(int)
	averageItemSizeInBytes := planMap["average_item_size_in_bytes"].(int)
	quantity := planMap["quantity"].(int)
	supportOSSClusterAPI := planMap["support_oss_cluster_api"].(bool)
	replication := planMap["Replication"].(bool)

	createDatabase := &subscriptions.CreateDatabase{
		Name:                   redis.String("dummy-database"),
		Protocol:               redis.String("redis"),
		MemoryLimitInGB:        redis.Float64(memoryLimitInGB),
		SupportOSSClusterAPI:   redis.Bool(supportOSSClusterAPI),
		Replication:            redis.Bool(replication),
		AverageItemSizeInBytes: redis.Int(averageItemSizeInBytes),
		ThroughputMeasurement: &subscriptions.CreateThroughput{
			By:    redis.String(throughputMeasurementBy),
			Value: redis.Int(throughputMeasurementValue),
		},
		Quantity: redis.Int(quantity),
	}
	createDatabases = append(createDatabases, createDatabase)
	return createDatabases
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

func toSubscriptionId(id string) (int, int, error) {
	parts := strings.Split(id, "/")

	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid id: %s", id)
	}

	if len(parts) == 1 {
		subId, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		return subId, 0, nil
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
