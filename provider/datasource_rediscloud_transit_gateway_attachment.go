package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceTransitGatewayAttachment() *schema.Resource {
	return &schema.Resource{
		Description: "",
		ReadContext: dataSourceTransitGatewayAttachmentRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "",
				Type:        schema.TypeString,
				Required:    true,
			},
			"tgw_id": {
				Description: "",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"aws_tgw_uid": {
				Description: "",
				Type:        schema.TypeString,
				Optional:    true,
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
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceTransitGatewayAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tgwTask, err := api.client.TransitGatewayAttachments.Get(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(db *attachments.TransitGatewayAttachment) bool

	if v, ok := d.GetOk("tgw_id"); ok {
		filters = append(filters, func(tgwa *attachments.TransitGatewayAttachment) bool {
			return redis.IntValue(tgwa.Id) == v.(int)
		})
	}

	if v, ok := d.GetOk("aws_tgw_uid"); ok {
		filters = append(filters, func(tgwa *attachments.TransitGatewayAttachment) bool {
			return redis.StringValue(tgwa.AwsTgwUid) == v.(string)
		})
	}

	tgws := filterTgwAttachments(tgwTask, filters)

	if len(tgws) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(tgws) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	tgwa := tgws[0]
	tgwaId := redis.IntValue(tgwa.Id)
	d.SetId(buildResourceId(subId, tgwaId))
	if err := d.Set("tgw_id", tgwaId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("aws_tgw_uid", redis.StringValue(tgwa.AwsTgwUid)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("attachment_uid", redis.StringValue(tgwa.AttachmentUid)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(tgwa.Status)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("attachment_status", redis.StringValue(tgwa.AttachmentStatus)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("aws_account_id", redis.StringValue(tgwa.AwsAccountId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cidrs", flattenCidrs(tgwa.Cidrs)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterTgwAttachments(getAttachmentsTask *attachments.GetAttachmentsTask, filters []func(tgwa *attachments.TransitGatewayAttachment) bool) []*attachments.TransitGatewayAttachment {
	var filtered []*attachments.TransitGatewayAttachment
	for _, tgwa := range getAttachmentsTask.Response.Resource.TransitGatewayAttachment {
		if filterTgwAttachment(tgwa, filters) {
			filtered = append(filtered, tgwa)
		}
	}
	return filtered
}

func filterTgwAttachment(tgwa *attachments.TransitGatewayAttachment, filters []func(tgwa *attachments.TransitGatewayAttachment) bool) bool {
	for _, filter := range filters {
		if !filter(tgwa) {
			return false
		}
	}
	return true
}

// TODO Consider moving to the resource
func flattenCidrs(cidrs []*attachments.Cidr) []string {
	cidrStrings := make([]string, len(cidrs))
	for _, cidr := range cidrs {
		cidrStrings = append(cidrStrings, redis.StringValue(cidr.CidrAddress))
	}
	return cidrStrings
}
