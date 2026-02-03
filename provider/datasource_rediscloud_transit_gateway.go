package provider

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func dataSourceTransitGateway() *schema.Resource {
	return &schema.Resource{
		Description: "The Transit Gateway data source allows access to an available Transit Gateway within your Redis Enterprise Cloud Account.",
		ReadContext: dataSourceTransitGatewayRead,

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
				Description: "When set, retry fetching until a Transit Gateway matching the filters is found or the specified timeout (in seconds) is reached. " +
					"Useful when accepting a TGW invitation and querying the TGW in the same Terraform run.",
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
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

	var tgws []*attachments.TransitGatewayAttachment

	if waitTimeoutSeconds > 0 {
		// Wait for a matching TGW to appear
		wait := &retry.StateChangeConf{
			Pending:      []string{"waiting"},
			Target:       []string{"found"},
			Timeout:      time.Duration(waitTimeoutSeconds) * time.Second,
			Delay:        5 * time.Second,
			PollInterval: 10 * time.Second,

			Refresh: func() (result interface{}, state string, err error) {
				log.Printf("[DEBUG] Waiting for Transit Gateway to appear for subscription %d", subId)

				tgwTask, err := api.Client.TransitGatewayAttachments.Get(ctx, subId)
				if err != nil {
					return nil, "", err
				}

				// Check for nil response structure - keep waiting
				if tgwTask == nil || tgwTask.Response == nil || tgwTask.Response.Resource == nil {
					return nil, "waiting", nil
				}

				filtered := filterTgwAttachments(tgwTask, filters)
				if len(filtered) == 0 {
					return nil, "waiting", nil
				}

				return filtered, "found", nil
			},
		}

		result, err := wait.WaitForStateContext(ctx)
		if err != nil {
			return diag.Errorf("Timeout waiting for Transit Gateway to appear for subscription %d: %s", subId, err)
		}
		tgws = result.([]*attachments.TransitGatewayAttachment)
	} else {
		// No waiting - fetch once
		tgwTask, err := api.Client.TransitGatewayAttachments.Get(ctx, subId)
		if err != nil {
			return diag.FromErr(err)
		}

		// Check for nil response structure
		if tgwTask == nil {
			return diag.Errorf("Transit Gateway API returned nil task for subscription %d", subId)
		}
		if tgwTask.Response == nil {
			return diag.Errorf("Transit Gateway API returned nil response for subscription %d", subId)
		}
		if tgwTask.Response.Resource == nil {
			return diag.Errorf("Transit Gateway API returned nil resource for subscription %d - subscription may not be fully provisioned yet", subId)
		}

		tgws = filterTgwAttachments(tgwTask, filters)
	}

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
