package provider

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fixedSubscriptions "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRedisCloudEssentialsSubscription() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages an Essentials Subscription within your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudEssentialsSubscriptionCreate,
		ReadContext:   resourceRedisCloudEssentialsSubscriptionRead,
		UpdateContext: resourceRedisCloudEssentialsSubscriptionUpdate,
		DeleteContext: resourceRedisCloudEssentialsSubscriptionDelete,

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
				Required:    true,
			},
			"status": {
				Description: "The status of this subscription",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"plan_id": {
				Description: "The identifier of the plan to template the subscription",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"payment_method": {
				Description:      "Payment method for the requested subscription. If credit-card is specified, the payment method id must be defined. This information is only used when creating a new subscription and any changes will be ignored after this.",
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
				Description: "The identifier of the method which will be charged for this subscription. Not required for free plans",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"creation_date": {
				Description: "The date/time this subscription was created",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudEssentialsSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	createSubscriptionRequest := fixedSubscriptions.FixedSubscriptionRequest{
		Name:   redis.String(d.Get("name").(string)),
		PlanId: redis.Int(d.Get("plan_id").(int)),
	}

	// payment_method_id only matters if it is a credit card
	if d.Get("payment_method").(string) != "credit-card" && d.Get("payment_method_id") != 0 {
		return diag.FromErr(errors.New("payment methods aside from credit-card cannot have a payment ID"))
	}

	if v, ok := d.GetOk("payment_method"); ok {
		createSubscriptionRequest.PaymentMethod = redis.String(v.(string))
	}

	if v, ok := d.GetOk("payment_method_id"); ok {
		createSubscriptionRequest.PaymentMethodID = redis.Int(v.(int))
	}

	// Create Subscription
	subId, err := api.Client.FixedSubscriptions.Create(ctx, createSubscriptionRequest)
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	d.SetId(strconv.Itoa(subId))

	// Confirm Subscription Active status
	err = waitForEssentialsSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	return resourceRedisCloudEssentialsSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudEssentialsSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscription, err := api.Client.FixedSubscriptions.Get(ctx, subId)
	if err != nil {
		notFound := &fixedSubscriptions.NotFound{}
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(subscription.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(subscription.Status)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("plan_id", redis.IntValue(subscription.PlanId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method_id", redis.IntValue(subscription.PaymentMethodID)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method", redis.StringValue(subscription.PaymentMethod)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("creation_date", redis.TimeValue(subscription.CreationDate).String()); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudEssentialsSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	change := d.HasChanges("name", "plan_id", "payment_method_id")

	if !change {
		return diags
	}

	updateSubscriptionRequest := fixedSubscriptions.FixedSubscriptionRequest{
		Name:   redis.String(d.Get("name").(string)),
		PlanId: redis.Int(d.Get("plan_id").(int)),
	}

	if v, ok := d.GetOk("payment_method"); ok {
		updateSubscriptionRequest.PaymentMethod = redis.String(v.(string))
	}

	if v, ok := d.GetOk("payment_method_id"); ok {
		updateSubscriptionRequest.PaymentMethodID = redis.Int(v.(int))
	}

	err = api.Client.FixedSubscriptions.Update(ctx, subId, updateSubscriptionRequest)
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	return resourceRedisCloudEssentialsSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudEssentialsSubscriptionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	// Wait for the subscription to be active before deleting it.
	if err := waitForEssentialsSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	// Delete subscription once all databases are deleted
	err = api.Client.FixedSubscriptions.Delete(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	err = waitForEssentialsSubscriptionToBeDeleted(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func waitForEssentialsSubscriptionToBeActive(ctx context.Context, id int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusPending},
		Target:  []string{subscriptions.SubscriptionStatusActive},
		Timeout: utils.SafetyTimeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed subscription %d to be active", id)

			subscription, err := api.Client.FixedSubscriptions.Get(ctx, id)
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

func waitForEssentialsSubscriptionToBeDeleted(ctx context.Context, id int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusDeleting},
		Target:  []string{"deleted"},
		Timeout: utils.SafetyTimeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed subscription %d to be deleted", id)

			subscription, err := api.Client.FixedSubscriptions.Get(ctx, id)
			if err != nil {
				notFound := &fixedSubscriptions.NotFound{}
				if errors.As(err, &notFound) {
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
