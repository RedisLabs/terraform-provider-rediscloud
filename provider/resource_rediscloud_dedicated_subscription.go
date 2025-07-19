package provider

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudDedicatedSubscription() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates a Dedicated Subscription within your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudDedicatedSubscriptionCreate,
		ReadContext:   resourceRedisCloudDedicatedSubscriptionRead,
		UpdateContext: resourceRedisCloudDedicatedSubscriptionUpdate,
		DeleteContext: resourceRedisCloudDedicatedSubscriptionDelete,

		Importer: &schema.ResourceImporter{
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
				Description: "A meaningful name to identify the dedicated subscription",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"payment_method": {
				Description:      "Payment method for the requested subscription. If credit card is specified, the payment method id must be defined.",
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^(credit-card|marketplace)$"), "must be 'credit-card' or 'marketplace'")),
				Optional:         true,
				Default:          "credit-card",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Id() == "" {
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
							Description:      "Cloud account identifier. Default: Redis Labs internal cloud account",
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
							Default:          "1",
						},
						"region": {
							Description: "Deployment region as defined by cloud provider",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"availability_zones": {
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
							Description:      "Deployment CIDR mask",
							Type:             schema.TypeString,
							ForceNew:         true,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
						},
						"networking_vpc_id": {
							Description: "Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified)",
							Type:        schema.TypeString,
							ForceNew:    true,
							Optional:    true,
							Default:     "",
						},
					},
				},
			},
			"instance_type": {
				Description: "Dedicated instance type specification",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_name": {
							Description: "The name of the dedicated instance type",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"replication": {
							Description: "Enable replication for high availability",
							Type:        schema.TypeBool,
							Optional:    true,
							ForceNew:    true,
							Default:     true,
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
						return false
					}
					if old != new {
						return false
					}
					return true
				},
			},
			"status": {
				Description: "Current status of the subscription",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"pricing": {
				Description: "Pricing details for this dedicated subscription",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Description: "The type of cost e.g. 'Instance'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"type_details": {
							Description: "Further detail e.g. instance type name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"quantity": {
							Description: "Number of instances",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"quantity_measurement": {
							Description: "Unit of measurement",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"price_per_unit": {
							Description: "Price per unit",
							Type:        schema.TypeFloat,
							Computed:    true,
						},
						"price_currency": {
							Description: "Currency e.g. 'USD'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"price_period": {
							Description: "Billing period e.g. 'hour'",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"region": {
							Description: "Region associated with the cost",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func resourceRedisCloudDedicatedSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_ = meta.(*apiClient) // Will be used when actual API client is available
	var diags diag.Diagnostics

	// Extract values from schema
	name := d.Get("name").(string)
	paymentMethod := d.Get("payment_method").(string)
	paymentMethodID, err := readPaymentMethodID(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Build cloud provider configuration
	cloudProviders, err := buildDedicatedCloudProviders(d.Get("cloud_provider"))
	if err != nil {
		return diag.FromErr(err)
	}

	// Build instance type configuration
	instanceType, err := buildDedicatedInstanceType(d.Get("instance_type"))
	if err != nil {
		return diag.FromErr(err)
	}

	// Create dedicated subscription request
	createRequest := map[string]interface{}{
		"name":            name,
		"paymentMethod":   paymentMethod,
		"paymentMethodId": paymentMethodID,
		"cloudProviders":  cloudProviders,
		"instanceType":    instanceType,
	}

	// Add Redis version if specified
	if redisVersion := d.Get("redis_version").(string); redisVersion != "" {
		createRequest["redisVersion"] = redisVersion
	}

	// TODO: Replace with dedicated subscription API when available:
	// subId, err := api.client.DedicatedSubscription.Create(ctx, createRequest)
	// if err != nil {
	//     return diag.FromErr(err)
	// }

	// DEVELOPMENT PLACEHOLDER: This is a placeholder implementation for development/testing
	// In production, this would use the actual dedicated subscription API
	log.Printf("[DEBUG] [PLACEHOLDER] Creating dedicated subscription with request: %+v", createRequest)
	log.Printf("[WARN] This is a placeholder implementation. Dedicated subscription API is not yet available.")

	// For development/testing, we'll use a predictable ID based on the name
	// This allows for consistent testing while clearly indicating it's not real
	subId := 999999 // Placeholder ID that's clearly not a real subscription

	d.SetId(strconv.Itoa(subId))

	// Skip the wait function for now since we can't actually create anything
	// err = waitForDedicatedSubscriptionToBeActive(ctx, subId, api)
	// if err != nil {
	//     return append(diags, diag.FromErr(err)...)
	// }

	return append(diags, resourceRedisCloudDedicatedSubscriptionRead(ctx, d, meta)...)
}

func resourceRedisCloudDedicatedSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_ = meta.(*apiClient) // Will be used when actual API client is available
	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// TODO: Replace with dedicated subscription API when available:
	// subscription, err := api.client.DedicatedSubscription.Get(ctx, subId)
	// if err != nil {
	//     if isNotFoundError(err) {
	//         log.Printf("[WARN] Dedicated subscription %d not found, removing from state", subId)
	//         d.SetId("")
	//         return diags
	//     }
	//     return diag.FromErr(err)
	// }

	// DEVELOPMENT PLACEHOLDER: This is a placeholder implementation
	log.Printf("[DEBUG] [PLACEHOLDER] Reading dedicated subscription %d", subId)
	log.Printf("[WARN] This is a placeholder implementation. Dedicated subscription API is not yet available.")

	// For the placeholder implementation, only handle our test ID
	if subId != 999999 {
		log.Printf("[WARN] Dedicated subscription %d not found (placeholder only supports ID 999999), removing from state", subId)
		d.SetId("")
		return diags
	}

	// Set placeholder computed values
	if err := d.Set("status", "active"); err != nil {
		return diag.FromErr(err)
	}

	// Note: In actual implementation, this would set all computed fields from the API response
	// including pricing, cloud provider details, etc.

	return diags
}

func resourceRedisCloudDedicatedSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Check what fields have changed
	if d.HasChange("name") {
		// Note: This would use the actual dedicated subscription API client when available
		// updateRequest := map[string]interface{}{
		//     "name": d.Get("name").(string),
		// }
		// err := api.client.DedicatedSubscription.Update(ctx, subId, updateRequest)

		log.Printf("[DEBUG] Updating dedicated subscription %d name", subId)
		// Placeholder for actual API call
	}

	// Wait for subscription to be active after update
	err = waitForDedicatedSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	return append(diags, resourceRedisCloudDedicatedSubscriptionRead(ctx, d, meta)...)
}

func resourceRedisCloudDedicatedSubscriptionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Note: This would use the actual dedicated subscription API client when available
	// err = api.client.DedicatedSubscription.Delete(ctx, subId)

	log.Printf("[DEBUG] Deleting dedicated subscription %d", subId)
	// Placeholder for actual API call

	// Wait for subscription to be deleted
	err = waitForDedicatedSubscriptionToBeDeleted(ctx, subId, api)
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	d.SetId("")
	return diags
}

// Helper functions for building request objects
func buildDedicatedCloudProviders(cloudProviderData interface{}) ([]map[string]interface{}, error) {
	cloudProviderList := cloudProviderData.([]interface{})
	if len(cloudProviderList) == 0 {
		return nil, fmt.Errorf("cloud_provider is required")
	}

	cloudProvider := cloudProviderList[0].(map[string]interface{})

	provider := map[string]interface{}{
		"provider":                   cloudProvider["provider"].(string),
		"cloudAccountId":            cloudProvider["cloud_account_id"].(string),
		"region":                    cloudProvider["region"].(string),
		"networkingDeploymentCidr":  cloudProvider["networking_deployment_cidr"].(string),
	}

	// Add optional fields
	if vpcId := cloudProvider["networking_vpc_id"].(string); vpcId != "" {
		provider["networkingVpcId"] = vpcId
	}

	if azs, ok := cloudProvider["availability_zones"].([]interface{}); ok && len(azs) > 0 {
		zones := make([]string, len(azs))
		for i, az := range azs {
			zones[i] = az.(string)
		}
		provider["availabilityZones"] = zones
	}

	return []map[string]interface{}{provider}, nil
}

func buildDedicatedInstanceType(instanceTypeData interface{}) (map[string]interface{}, error) {
	instanceTypeList := instanceTypeData.([]interface{})
	if len(instanceTypeList) == 0 {
		return nil, fmt.Errorf("instance_type is required")
	}

	instanceType := instanceTypeList[0].(map[string]interface{})

	return map[string]interface{}{
		"instanceName": instanceType["instance_name"].(string),
		"replication":  instanceType["replication"].(bool),
	}, nil
}

// Wait functions for async operations using resource status polling
func waitForDedicatedSubscriptionToBeActive(ctx context.Context, id int, api *apiClient) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusPending},
		Target:       []string{subscriptions.SubscriptionStatusActive},
		Timeout:      safetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for dedicated subscription %d to be active", id)

			// TODO: Replace with dedicated subscription API when available:
			// subscription, err := api.client.DedicatedSubscription.Get(ctx, id)

			// For now, use the existing Subscription service as dedicated subscriptions
			// are likely a type of subscription in the Redis Cloud API
			subscription, err := api.client.Subscription.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			// Use the same pattern as the existing subscription wait function
			var status string
			if subscription.Status != nil {
				status = *subscription.Status
			} else {
				status = subscriptions.SubscriptionStatusPending
			}

			log.Printf("[DEBUG] Dedicated subscription %d status: %s", id, status)
			return status, status, nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForDedicatedSubscriptionToBeDeleted(ctx context.Context, id int, api *apiClient) error {
	wait := &retry.StateChangeConf{
		Pending:      []string{subscriptions.SubscriptionStatusDeleting},
		Target:       []string{"deleted"},
		Timeout:      safetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for dedicated subscription %d to be deleted", id)

			// TODO: Replace with dedicated subscription API when available:
			// subscription, err := api.client.DedicatedSubscription.Get(ctx, id)

			// For now, use the existing Subscription service
			subscription, err := api.client.Subscription.Get(ctx, id)
			if err != nil {
				// TODO: When dedicated subscription API is available, check for specific NotFound error:
				// if _, ok := err.(*dedicated_subscriptions.NotFound); ok {
				//     log.Printf("[DEBUG] Dedicated subscription %d not found, considering it deleted", id)
				//     return "deleted", "deleted", nil
				// }

				// For now, assume any error means the subscription might be deleted
				log.Printf("[DEBUG] Error getting dedicated subscription %d (might be deleted): %v", id, err)
				return "deleted", "deleted", nil
			}

			// If we can still get the subscription, check its status
			var status string
			if subscription.Status != nil {
				status = *subscription.Status
			} else {
				status = subscriptions.SubscriptionStatusDeleting
			}

			log.Printf("[DEBUG] Dedicated subscription %d deletion status: %s", id, status)
			return status, status, nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
