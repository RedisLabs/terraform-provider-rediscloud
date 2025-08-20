package psc

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceRedisCloudSubscriptionPeering() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates a VPC peering for an existing Redis Enterprise Cloud Subscription, allowing access to your subscription databases as if they were on the same network.",
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
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				ForceNew:         true,
			},
			"provider_name": {
				Type:             schema.TypeString,
				Description:      "The cloud provider to use with the vpc peering, (either `AWS` or `GCP`)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
				Optional:         true,
				ForceNew:         true,
				Default:          "AWS",
			},
			"region": {
				Description: "AWS Region that the VPC to be peered lives in",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"aws_account_id": {
				Description: "AWS account id that the VPC to be peered lives in",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"vpc_id": {
				Description: "Identifier of the VPC to be peered",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"vpc_cidr": {
				Description:   "CIDR range of the VPC to be peered",
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"vpc_cidrs"},
			},
			"vpc_cidrs": {
				Description: "CIDR ranges of the VPC to be peered",
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
				},
				ConflictsWith: []string{"vpc_cidr"},
			},
			"gcp_project_id": {
				Description: "GCP project ID that the VPC to be peered lives in",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"gcp_network_name": {
				Description: "The name of the network to be peered",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"status": {
				Description: "Current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"aws_peering_id": {
				Description: "Identifier of the AWS cloud peering",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"gcp_redis_project_id": {
				Description: "Identifier of the Redis Enterprise Cloud GCP project to be peered",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"gcp_redis_network_name": {
				Description: "The name of the Redis Enterprise Cloud network to be peered",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"gcp_peering_id": {
				Description: "Identifier of the cloud peering",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudSubscriptionPeeringCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*utils.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	providerName := d.Get("provider_name").(string)

	peeringRequest := subscriptions.CreateVPCPeering{}

	if providerName == "AWS" {

		region, ok := d.GetOk("region")
		if !ok {
			return diag.Errorf("`region` must be set when `provider_name` is `AWS`")
		}

		awsAccountID, ok := d.GetOk("aws_account_id")
		if !ok {
			return diag.Errorf("`aws_account_id` must be set when `provider_name` is `AWS`")
		}

		vpcID, ok := d.GetOk("vpc_id")
		if !ok {
			return diag.Errorf("`vpc_id` must be set when `provider_name` is `AWS`")
		}

		if vpcCIDR, ok := d.GetOk("vpc_cidr"); ok {
			peeringRequest.VPCCidr = redis.String(vpcCIDR.(string))
		} else if vpcCIDRs, ok := d.GetOk("vpc_cidrs"); ok {
			peeringRequest.VPCCidrs = setToStringSlice(vpcCIDRs.(*schema.Set))
		} else {
			return diag.Errorf("`vpc_cidr` or `vpc_cidrs` must be set when `provider_name` is `AWS`")
		}

		peeringRequest.Region = redis.String(region.(string))
		peeringRequest.AWSAccountID = redis.String(awsAccountID.(string))
		peeringRequest.VPCId = redis.String(vpcID.(string))
	}

	if providerName == "GCP" {

		gcpProjectID, ok := d.GetOk("gcp_project_id")
		if !ok {
			return diag.Errorf("`gcp_project_id` must be set when `provider_name` is `GCP`")
		}

		gcpNetworkName, ok := d.GetOk("gcp_network_name")
		if !ok {
			return diag.Errorf("`network_name` must be set when `provider_name` is `GCP`")
		}

		peeringRequest.Provider = redis.String(strings.ToLower(providerName))
		peeringRequest.VPCProjectUID = redis.String(gcpProjectID.(string))
		peeringRequest.VPCNetworkName = redis.String(gcpNetworkName.(string))
	}

	peering, err := api.Client.Subscription.CreateVPCPeering(ctx, subId, peeringRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildResourceId(subId, peering))

	err = waitForPeeringToBeInitiated(ctx, subId, peering, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudSubscriptionPeeringRead(ctx, d, meta)
}

func resourceRedisCloudSubscriptionPeeringRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*utils.ApiClient)
	var diags diag.Diagnostics

	subId, id, err := toVpcPeeringId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("subscription_id", strconv.Itoa(subId)); err != nil {
		return diag.FromErr(err)
	}

	peerings, err := api.Client.Subscription.ListVPCPeering(ctx, subId)
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

	if err := d.Set("status", redis.StringValue(peering.Status)); err != nil {
		return diag.FromErr(err)
	}

	providerName := "AWS"

	if redis.StringValue(peering.GCPProjectUID) != "" {
		providerName = "GCP"
	}

	if err := d.Set("provider_name", providerName); err != nil {
		return diag.FromErr(err)
	}

	if providerName == "AWS" {
		if err := d.Set("aws_account_id", redis.StringValue(peering.AWSAccountID)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("aws_peering_id", redis.StringValue(peering.AWSPeeringID)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("vpc_id", redis.StringValue(peering.VPCId)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("region", redis.StringValue(peering.Region)); err != nil {
			return diag.FromErr(err)
		}

		// A peering that was created with `VPCCidrs` containing a single item will be read back with the `VPCCidr` set
		// and `VPCCidrs` unset.
		var vpcCidr *string
		if peering.VPCCidr != nil {
			vpcCidr = peering.VPCCidr
		}

		var cidrs []string
		if len(peering.VPCCidrs) != 0 {
			for _, cidr := range peering.VPCCidrs {
				if vpcCidr == nil {
					vpcCidr = cidr.VPCCidr
				}
				cidrs = append(cidrs, redis.StringValue(cidr.VPCCidr))
			}
		} else {
			cidrs = []string{redis.StringValue(vpcCidr)}
		}

		if err := d.Set("vpc_cidr", redis.StringValue(vpcCidr)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("vpc_cidrs", cidrs); err != nil {
			return diag.FromErr(err)
		}
	}
	if providerName == "GCP" {
		if err := d.Set("gcp_project_id", redis.StringValue(peering.GCPProjectUID)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("gcp_network_name", redis.StringValue(peering.NetworkName)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("gcp_redis_project_id", redis.StringValue(peering.RedisProjectUID)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("gcp_redis_network_name", redis.StringValue(peering.RedisNetworkName)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("gcp_peering_id", redis.StringValue(peering.CloudPeeringID)); err != nil {
			return diag.FromErr(err)
		}

	}
	return diags
}

func resourceRedisCloudSubscriptionPeeringDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*utils.ApiClient)
	var diags diag.Diagnostics

	subId, id, err := toVpcPeeringId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	err = api.Client.Subscription.DeleteVPCPeering(ctx, subId, id)
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

func waitForPeeringToBeInitiated(ctx context.Context, subId, id int, api *utils.ApiClient) error {
	wait := &retry.StateChangeConf{
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

			list, err := api.Client.Subscription.ListVPCPeering(ctx, subId)
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
