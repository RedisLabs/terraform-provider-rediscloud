package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"time"
)

func resourceRedisCloudTransitGatewayAttachment() *schema.Resource {
	return &schema.Resource{
		Description:   "",
		CreateContext: resourceRedisCloudTransitGatewayAttachmentCreate,
		ReadContext:   resourceRedisCloudTransitGatewayAttachmentRead,
		UpdateContext: resourceRedisCloudTransitGatewayAttachmentUpdate,
		DeleteContext: resourceRedisCloudTransitGatewayAttachmentDelete,

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
				Description: "",
				Type:        schema.TypeString,
				Required:    true,
			},
			"tgw_id": {
				Description: "",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"aws_tgw_uid": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"attachment_uid": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"status": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"attachment_status": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"aws_account_id": {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cidrs": {
				Description: "",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceRedisCloudTransitGatewayAttachmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	tgwId := d.Get("tgw_id").(int)
	if err != nil {
		return diag.FromErr(err)
	}

	// At this point, cidrs has to be empty. We cannot honour the user's configuration until the invitation has been accepted
	cidrs := interfaceToStringSlice(d.Get("cidrs").([]interface{}))
	if len(cidrs) > 0 {
		return diag.Errorf("Attachment cannot be created with Cidrs provided, it must be accepted first. This resource may then be updated with Cidrs.")
	}

	_, err = api.client.TransitGatewayAttachments.Create(ctx, subscriptionId, tgwId)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudTransitGatewayAttachmentRead(ctx, d, meta)
}

func resourceRedisCloudTransitGatewayAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwId := d.Get("tgw_id").(int)

	tgwTask, err := api.client.TransitGatewayAttachments.Get(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(db *attachments.TransitGatewayAttachment) bool
	filters = append(filters, func(tgw *attachments.TransitGatewayAttachment) bool {
		return redis.IntValue(tgw.Id) == tgwId
	})

	tgws := filterTgwAttachments(tgwTask, filters)

	if len(tgws) == 0 {
		return diag.Errorf("No such Transit Gateway! subscription_id/tgw_id: %d/%d", subId, tgwId)
	}

	if len(tgws) > 1 {
		return diag.Errorf("More than one Transit Gateway identified! subscription_id/tgw_id: %d/%d", subId, tgwId)
	}

	tgw := tgws[0]
	d.SetId(buildResourceId(subId, tgwId))
	if err := d.Set("aws_tgw_uid", redis.StringValue(tgw.AwsTgwUid)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("attachment_uid", redis.StringValue(tgw.AttachmentUid)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(tgw.Status)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("attachment_status", redis.StringValue(tgw.AttachmentStatus)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("aws_account_id", redis.StringValue(tgw.AwsAccountId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cidrs", flattenCidrs(tgw.Cidrs)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudTransitGatewayAttachmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	tgwId := d.Get("tgw_id").(int)
	if err != nil {
		return diag.FromErr(err)
	}

	cidrs := interfaceToStringSlice(d.Get("cidrs").([]interface{}))

	err = api.client.TransitGatewayAttachments.Update(ctx, subId, tgwId, cidrs)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudTransitGatewayAttachmentRead(ctx, d, meta)
}

func resourceRedisCloudTransitGatewayAttachmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	tgwId := d.Get("tgw_id").(int)
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.client.TransitGatewayAttachments.Delete(ctx, subscriptionId, tgwId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func flattenCidrs(cidrs []*attachments.Cidr) []string {
	cidrStrings := make([]string, len(cidrs))
	for _, cidr := range cidrs {
		cidrStrings = append(cidrStrings, redis.StringValue(cidr.CidrAddress))
	}
	return cidrStrings
}
