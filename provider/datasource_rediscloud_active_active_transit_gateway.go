package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceActiveActiveTransitGateway() *schema.Resource {
	return &schema.Resource{
		Description: "The Active Active Transit Gateway data source allows access to an available Transit Gateway within your Redis Enterprise Cloud Account.",
		ReadContext: dataSourceActiveActiveTransitGatewayRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The id of an Active Active subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"region_id": {
				Description: "The id of the AWS region",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"tgw_id": {
				Description: "The id of the Transit Gateway relative to the associated subscription",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"aws_tgw_uid": {
				Description: "The id of the Transit Gateway as known to AWS",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"attachment_uid": {
				Description: "A unique identifier for the Subscription/Transit Gateway attachment, if any",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"status": {
				Description: "The status of the Transit Gateway",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"attachment_status": {
				Description: "The status of the Subscription/Transit Gateway attachment, if any",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"aws_account_id": {
				Description: "The Transit Gateway's AWS account id",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cidrs": {
				Description: "A list of consumer Cidr blocks, if an attachment exists",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceActiveActiveTransitGatewayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	regionId := d.Get("region_id").(int)
	tgwTask, err := api.Client.TransitGatewayAttachments.GetActiveActive(ctx, subId, regionId)
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

	tgw := tgws[0]
	tgwId := redis.IntValue(tgw.Id)
	d.SetId(utils.BuildResourceId(subId, tgwId))
	if err := d.Set("tgw_id", tgwId); err != nil {
		return diag.FromErr(err)
	}
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
