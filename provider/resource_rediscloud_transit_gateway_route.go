package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRedisCloudTransitGatewayRoute() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages Transit Gateway routing (CIDRs) for a Pro/Flexible Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudTransitGatewayRouteCreate,
		ReadContext:   resourceRedisCloudTransitGatewayRouteRead,
		UpdateContext: resourceRedisCloudTransitGatewayRouteUpdate,
		DeleteContext: resourceRedisCloudTransitGatewayRouteDelete,

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
				Description: "The ID of the Pro/Flexible subscription",
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

func resourceRedisCloudTransitGatewayRouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwId := d.Get("tgw_id").(int)

	cidrs := utils.InterfaceToStringSlice(d.Get("cidrs").([]interface{}))

	err = api.Client.TransitGatewayAttachments.Update(ctx, subId, tgwId, cidrs)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(utils.BuildResourceId(subId, tgwId))

	return resourceRedisCloudTransitGatewayRouteRead(ctx, d, meta)
}

func resourceRedisCloudTransitGatewayRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, tgwId, err := transitgateway.ParseTransitGatewayAttachmentId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("subscription_id", strconv.Itoa(subId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tgw_id", tgwId); err != nil {
		return diag.FromErr(err)
	}

	tgwTask, err := api.Client.TransitGatewayAttachments.Get(ctx, subId)
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
		return diag.Errorf("More than one Transit Gateway identified! subscription_id/tgw_id: %d/%d", subId, tgwId)
	}

	tgw := tgws[0]
	if err := d.Set("cidrs", flattenCidrs(tgw.Cidrs)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudTransitGatewayRouteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwId := d.Get("tgw_id").(int)

	cidrs := utils.InterfaceToStringSlice(d.Get("cidrs").([]interface{}))

	err = api.Client.TransitGatewayAttachments.Update(ctx, subId, tgwId, cidrs)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudTransitGatewayRouteRead(ctx, d, meta)
}

func resourceRedisCloudTransitGatewayRouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwId := d.Get("tgw_id").(int)

	// Clear CIDRs by sending an empty list
	emptyCidrs := make([]*string, 0)
	err = api.Client.TransitGatewayAttachments.Update(ctx, subId, tgwId, emptyCidrs)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
