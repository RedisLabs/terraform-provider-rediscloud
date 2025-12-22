package provider

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func resourceRedisCloudActiveActiveTransitGatewayRoute() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages Transit Gateway routing (CIDRs) for an Active-Active Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActiveTransitGatewayRouteCreate,
		ReadContext:   resourceRedisCloudActiveActiveTransitGatewayRouteRead,
		UpdateContext: resourceRedisCloudActiveActiveTransitGatewayRouteUpdate,
		DeleteContext: resourceRedisCloudActiveActiveTransitGatewayRouteDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of the Active-Active subscription",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"region_id": {
				Description: "The ID of the AWS region",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"tgw_id": {
				Description: "The ID of the Transit Gateway",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"cidrs": {
				Description: "A list of consumer CIDR blocks",
				Type:        schema.TypeList,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceRedisCloudActiveActiveTransitGatewayRouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	regionId, err := strconv.Atoi(d.Get("region_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwId := d.Get("tgw_id").(int)

	cidrs := utils.InterfaceToStringSlice(d.Get("cidrs").([]interface{}))

	err = api.Client.TransitGatewayAttachments.UpdateActiveActive(ctx, subId, regionId, tgwId, cidrs)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(transitgateway.BuildActiveActiveTransitGatewayAttachmentId(subId, regionId, tgwId))

	return resourceRedisCloudActiveActiveTransitGatewayRouteRead(ctx, d, meta)
}

func resourceRedisCloudActiveActiveTransitGatewayRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, regionId, tgwId, err := transitgateway.ParseActiveActiveTransitGatewayAttachmentId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("subscription_id", strconv.Itoa(subId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region_id", strconv.Itoa(regionId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tgw_id", tgwId); err != nil {
		return diag.FromErr(err)
	}

	// Wait for Transit Gateway resource to become available (handles subscription provisioning delays)
	tgwTask, err := utils.WaitForActiveActiveTransitGatewayResourceToBeAvailable(ctx, subId, regionId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(tgw *attachments.TransitGatewayAttachment) bool
	filters = append(filters, func(tgw *attachments.TransitGatewayAttachment) bool {
		return redis.IntValue(tgw.Id) == tgwId
	})

	tgws := filterTgwAttachments(tgwTask, filters)

	if len(tgws) == 0 {
		d.SetId("")
		return diags
	}

	if len(tgws) > 1 {
		return diag.Errorf("More than one Transit Gateway identified! subscription_id/region_id/tgw_id: %d/%d/%d", subId, regionId, tgwId)
	}

	tgw := tgws[0]
	if err := d.Set("cidrs", flattenCidrs(tgw.Cidrs)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudActiveActiveTransitGatewayRouteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	regionId, err := strconv.Atoi(d.Get("region_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwId := d.Get("tgw_id").(int)

	cidrs := utils.InterfaceToStringSlice(d.Get("cidrs").([]interface{}))

	err = api.Client.TransitGatewayAttachments.UpdateActiveActive(ctx, subId, regionId, tgwId, cidrs)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudActiveActiveTransitGatewayRouteRead(ctx, d, meta)
}

func resourceRedisCloudActiveActiveTransitGatewayRouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	regionId, err := strconv.Atoi(d.Get("region_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwId := d.Get("tgw_id").(int)

	// Clear CIDRs by sending an empty list
	emptyCidrs := make([]*string, 0)
	err = api.Client.TransitGatewayAttachments.UpdateActiveActive(ctx, subId, regionId, tgwId, emptyCidrs)
	if err != nil {
		if strings.Contains(err.Error(), "TGW_ATTACHMENT_DOES_NOT_EXIST") {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
