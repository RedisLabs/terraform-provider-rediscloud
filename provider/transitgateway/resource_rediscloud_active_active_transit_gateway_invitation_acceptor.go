package transitgateway

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func ResourceRedisCloudActiveActiveTransitGatewayInvitationAcceptor() *schema.Resource {
	return &schema.Resource{
		Description:   "Accepts or rejects an AWS Transit Gateway resource share invitation for a specific region in an Active-Active subscription. Invitations must be created externally via AWS Resource Manager.",
		CreateContext: resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorCreate,
		ReadContext:   resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorRead,
		UpdateContext: resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorUpdate,
		DeleteContext: resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of the Active-Active subscription",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"region_id": {
				Description: "The region ID",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"tgw_invitation_id": {
				Description: "The Transit Gateway invitation ID",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"action": {
				Description:      "Action to perform: accept or reject",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"accept", "reject"}, false)),
			},
			"name": {
				Description: "The name of the resource share",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"resource_share_uid": {
				Description: "The AWS Resource Share ARN",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"aws_account_id": {
				Description: "The AWS account ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"status": {
				Description: "The invitation status",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"shared_date": {
				Description: "The date the resource was shared",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subscriptionId)
	defer utils.SubscriptionMutex.Unlock(subscriptionId)

	regionId := d.Get("region_id").(int)
	tgwInvitationId := d.Get("tgw_invitation_id").(int)
	action := d.Get("action").(string)

	// Verify invitation exists
	invitations, err := api.Client.TransitGatewayAttachments.ListInvitationsActiveActive(ctx, subscriptionId, regionId)
	if err != nil {
		return diag.FromErr(err)
	}

	invitation := findTransitGatewayInvitation(tgwInvitationId, invitations)
	if invitation == nil {
		return diag.Errorf("Transit gateway invitation with id %d not found", tgwInvitationId)
	}

	// Check if already in desired state
	currentStatus := redis.StringValue(invitation.Status)
	if action == "accept" && currentStatus == "accepted" {
		log.Printf("[INFO] Invitation %d already accepted, skipping API call", tgwInvitationId)
	} else if action == "reject" && currentStatus == "rejected" {
		log.Printf("[INFO] Invitation %d already rejected, skipping API call", tgwInvitationId)
	} else {
		// Perform action
		if action == "accept" {
			err = api.Client.TransitGatewayAttachments.AcceptInvitationActiveActive(ctx, subscriptionId, regionId, tgwInvitationId)
		} else {
			err = api.Client.TransitGatewayAttachments.RejectInvitationActiveActive(ctx, subscriptionId, regionId, tgwInvitationId)
		}

		if err != nil {
			return diag.FromErr(err)
		}

		// Wait for subscription to be active
		err = utils.WaitForSubscriptionToBeActive(ctx, subscriptionId, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(buildActiveActiveTransitGatewayInvitationAcceptorId(subscriptionId, regionId, tgwInvitationId))

	return resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorRead(ctx, d, meta)
}

func resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subscriptionId, regionId, tgwInvitationId, err := parseActiveActiveTransitGatewayInvitationAcceptorId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	invitations, err := api.Client.TransitGatewayAttachments.ListInvitationsActiveActive(ctx, subscriptionId, regionId)
	if err != nil {
		return diag.FromErr(err)
	}

	invitation := findTransitGatewayInvitation(tgwInvitationId, invitations)
	if invitation == nil {
		log.Printf("[WARN] Transit gateway invitation %d not found, removing from state", tgwInvitationId)
		d.SetId("")
		return diags
	}

	if err := d.Set("subscription_id", strconv.Itoa(subscriptionId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region_id", regionId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tgw_invitation_id", redis.IntValue(invitation.Id)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", redis.StringValue(invitation.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_share_uid", redis.StringValue(invitation.ResourceShareUid)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("aws_account_id", redis.StringValue(invitation.AwsAccountId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(invitation.Status)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("shared_date", redis.StringValue(invitation.SharedDate)); err != nil {
		return diag.FromErr(err)
	}

	// Warn if status doesn't match action
	action := d.Get("action").(string)
	currentStatus := redis.StringValue(invitation.Status)
	if action == "accept" && currentStatus != "accepted" {
		log.Printf("[WARN] Drift detected: action is 'accept' but invitation status is '%s'", currentStatus)
	}
	if action == "reject" && currentStatus != "rejected" {
		log.Printf("[WARN] Drift detected: action is 'reject' but invitation status is '%s'", currentStatus)
	}

	return diags
}

func resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Update = re-apply the action (drift correction)
	return resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorCreate(ctx, d, meta)
}

func resourceRedisCloudActiveActiveTransitGatewayInvitationAcceptorDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// No-op pattern - just remove from state
	log.Printf("[INFO] Removing Active-Active Transit Gateway Invitation Acceptor from state")
	d.SetId("")
	return nil
}

func buildActiveActiveTransitGatewayInvitationAcceptorId(subscriptionId int, regionId int, tgwInvitationId int) string {
	return fmt.Sprintf("%d/%d/%d", subscriptionId, regionId, tgwInvitationId)
}

func parseActiveActiveTransitGatewayInvitationAcceptorId(id string) (subscriptionId int, regionId int, tgwInvitationId int, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid ID format, expected: subscription_id/region_id/tgw_invitation_id")
	}

	subscriptionId, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid subscription_id: %w", err)
	}

	regionId, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid region_id: %w", err)
	}

	tgwInvitationId, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid tgw_invitation_id: %w", err)
	}

	return subscriptionId, regionId, tgwInvitationId, nil
}
