package pro

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var (
	_ datasource.DataSource              = &proSubscriptionDataSource{}
	_ datasource.DataSourceWithConfigure = &proSubscriptionDataSource{}
)

type proSubscriptionDataSource struct {
	client *client.ApiClient
}

// NewProSubscriptionDataSource provides the Framework implementation for rediscloud_subscription.
func NewProSubscriptionDataSource() datasource.DataSource {
	return &proSubscriptionDataSource{}
}

// Metadata keeps the public pro/flexible subscription name as rediscloud_subscription.
func (d *proSubscriptionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription"
}

func (d *proSubscriptionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *proSubscriptionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Pro Subscription data source allows access to the details of an existing pro subscription within your Redis Enterprise Cloud account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the subscription",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the subscription to filter returned subscriptions",
				Computed:    true,
				Optional:    true,
			},
			"payment_method": schema.StringAttribute{
				Description: "Payment method for the requested subscription. Either 'credit-card' or 'marketplace'",
				Computed:    true,
			},
			"payment_method_id": schema.StringAttribute{
				Description: "A valid payment method pre-defined in the current account",
				Computed:    true,
			},
			"memory_storage": schema.StringAttribute{
				Description: "Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’",
				Computed:    true,
			},
			"number_of_databases": schema.Int64Attribute{
				Description: "The number of databases that are linked to this subscription",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the subscription",
				Computed:    true,
			},
			"customer_managed_key_enabled": schema.BoolAttribute{
				Description: "Whether customer managed key encryption is enabled for the subscription",
				Computed:    true,
			},
			"customer_managed_key_deletion_grace_period": schema.StringAttribute{
				Description: "The deletion grace period for the customer managed key (e.g. 'immediate', '15-minutes')",
				Computed:    true,
			},
			"customer_managed_key_redis_service_account": schema.StringAttribute{
				Description: "The Redis service account principal associated with the subscription. This is used to grant access to the customer managed encryption key",
				Computed:    true,
			},
			"customer_managed_key_aws_role_arn": schema.StringAttribute{
				Description: "The ARN of the IAM role used by the subscription to access the AWS KMS customer managed key. Grant this role access to your KMS key via key policy",
				Computed:    true,
			},
			"public_endpoint_access": schema.BoolAttribute{
				Description: "Whether public endpoint access is enabled for databases in the subscription",
				Computed:    true,
			},
			"prometheus_endpoint": schema.StringAttribute{
				Description: "The Prometheus scrape endpoint for databases in this subscription. Use this to configure your Prometheus server to scrape metrics from your Redis Cloud databases.",
				Computed:    true,
			},
			// Object collections are computed list attributes, not blocks: a deferred read
			// plans blocks as empty lists instead of unknown, which breaks [0] references.
			// Nested attributes are not an option because the mux serves protocol v5.
			"cloud_provider": schema.ListAttribute{
				Description: "A cloud provider object",
				Computed:    true,
				ElementType: utils.CloudProviderElemType,
			},
			"maintenance_windows": schema.ListAttribute{
				Description: "Details about the subscription's maintenance window specification",
				Computed:    true,
				CustomType:  customtypes.NewMaintenanceListType(),
				ElementType: customtypes.NewMaintenanceListType().ElementType(),
			},
			"pricing": schema.ListAttribute{
				Description: "Pricing details totalled over this Subscription",
				Computed:    true,
				ElementType: utils.PricingElemType,
			},
		},
	}
}
