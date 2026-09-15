package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func dataSourceTransitGateway() *schema.Resource {
	return &schema.Resource{
		Description: "The Transit Gateway data source allows access to an available Transit Gateway within your Redis Enterprise Cloud Account.",
		ReadContext: dataSourceTransitGatewayRead,
		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The id of a Pro/Flexible subscription",
				Type:        schema.TypeString,
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
			"wait_for_tgw_timeout": {
				Description: "Overrides the read timeout, in seconds, while waiting for a Transit Gateway matching the configured filters.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Deprecated: "Configure the `read` timeout in the `timeouts` block instead. " +
					"This attribute will be removed in version 3.0 of the provider.",
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

func dataSourceTransitGatewayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// Build filters for matching TGWs
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

	waitTimeoutSeconds := d.Get("wait_for_tgw_timeout").(int)

	timeout := d.Timeout(schema.TimeoutRead)
	if waitTimeoutSeconds > 0 {
		// Preserve the deprecated timeout as an override while configurations migrate to the read timeout.
		timeout = time.Duration(waitTimeoutSeconds) * time.Second
	}

	// Always apply the filters during polling so the read completes only when its query can succeed.
	tgwTask, err := utils.WaitForTransitGatewayResourceToMatchFilters(ctx, subId, api, filters, timeout, subscriptions.SubscriptionDeploymentTypeSingleRegion, 0)
	if err != nil {
		return diag.FromErr(err)
	}

	// The waiter uses the filters to determine readiness. Apply them again to obtain the data source result.
	filteredTgws := filterTgwAttachments(tgwTask, filters)

	if len(filteredTgws) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(filteredTgws) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	tgw := filteredTgws[0]
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

func filterTgwAttachments(getAttachmentsTask *attachments.GetAttachmentsTask, filters []func(tgwa *attachments.TransitGatewayAttachment) bool) []*attachments.TransitGatewayAttachment {
	var filtered []*attachments.TransitGatewayAttachment

	// Defensive nil checks - callers should validate before calling, but we guard here too
	if getAttachmentsTask == nil || getAttachmentsTask.Response == nil || getAttachmentsTask.Response.Resource == nil {
		return filtered
	}

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
