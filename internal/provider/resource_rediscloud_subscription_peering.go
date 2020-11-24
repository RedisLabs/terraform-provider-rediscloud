package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func resourceRedisCloudSubscriptionPeering() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates an AWS VPC peering for an existing Redis Enterprise Cloud Subscription, allowing access to your subscription databases as if they were on the same network.",
		CreateContext: resourceRedisCloudSubscriptionPeeringCreate,
		ReadContext:   resourceRedisCloudSubscriptionPeeringRead,
		DeleteContext: resourceRedisCloudSubscriptionPeeringDelete,
		// UpdateContext - not set as all attributes are not updatable or computed

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				_, _, err := toVpcPeeringId(d.Id())
				if err != nil {
					return nil, err
				}
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "A valid subscription predefined in the current account",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				ForceNew:         true,
			},
			"region": {
				Description: "AWS Region that the VPC to be peered lives in",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"aws_account_id": {
				Description: "AWS account id that the VPC to be peered lives in",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"vpc_id": {
				Description: "Identifier of the VPC to be peered",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"vpc_cidr": {
				Description: "CIDR range of the VPC to be peered",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"status": {
				Description: "Current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudSubscriptionPeeringCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	peering, err := api.client.Subscription.CreateVPCPeering(ctx, subId, subscriptions.CreateVPCPeering{
		Region:       redis.String(d.Get("region").(string)),
		AWSAccountID: redis.String(d.Get("aws_account_id").(string)),
		VPCId:        redis.String(d.Get("vpc_id").(string)),
		VPCCidr:      redis.String(d.Get("vpc_cidr").(string)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildVpcPeeringId(subId, peering))

	err = waitForPeeringToBeInitiated(ctx, subId, peering, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudSubscriptionPeeringRead(ctx, d, meta)
}

func resourceRedisCloudSubscriptionPeeringRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	subId, id, err := toVpcPeeringId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	peerings, err := api.client.Subscription.ListVPCPeering(ctx, subId)
	if err != nil {
		if _, ok := err.(*subscriptions.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	peering := findVpcPeering(id, peerings)
	if peering == nil {
		d.SetId("")
		return diags
	}

	if err := d.Set("aws_account_id", redis.StringValue(peering.AWSAccountID)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("vpc_id", redis.StringValue(peering.VPCId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("vpc_cidr", redis.StringValue(peering.VPCCidr)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(peering.Status)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudSubscriptionPeeringDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	subId, id, err := toVpcPeeringId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	err = api.client.Subscription.DeleteVPCPeering(ctx, subId, id)
	if err != nil {
		if _, ok := err.(*subscriptions.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func buildVpcPeeringId(subId int, id int) string {
	return fmt.Sprintf("%d/%d", subId, id)
}

func toVpcPeeringId(id string) (int, int, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid id: %s", id)
	}

	sub, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	peering, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return sub, peering, nil
}

func findVpcPeering(id int, peerings []*subscriptions.VPCPeering) *subscriptions.VPCPeering {
	for _, peering := range peerings {
		if redis.IntValue(peering.ID) == id {
			return peering
		}
	}
	return nil
}

func waitForPeeringToBeInitiated(ctx context.Context, subId, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay: 10 * time.Second,
		Pending: []string{
			subscriptions.VPCPeeringStatusInitiatingRequest,
		},
		Target: []string{
			subscriptions.VPCPeeringStatusActive,
			subscriptions.VPCPeeringStatusInactive,
			subscriptions.VPCPeeringStatusPendingAcceptance,
		},
		Timeout: 10 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for vpc peering %d to be initiated", id)

			list, err := api.client.Subscription.ListVPCPeering(ctx, subId)
			if err != nil {
				return nil, "", err
			}

			peering := findVpcPeering(id, list)
			if peering == nil {
				log.Printf("Peering %d/%d not present yet", subId, id)
				return nil, "", nil
			}

			return redis.StringValue(peering.Status), redis.StringValue(peering.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
