package activeactive

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &activeActiveSubscriptionDataSource{}
	_ datasource.DataSourceWithConfigure = &activeActiveSubscriptionDataSource{}
)

// activeActiveSubscriptionDataSource is the data source implementation.
type activeActiveSubscriptionDataSource struct {
	client *client.ApiClient
}

// NewActiveActiveSubscriptionDataSource creates a new data source instance.
func NewActiveActiveSubscriptionDataSource() datasource.DataSource {
	return &activeActiveSubscriptionDataSource{}
}

// Metadata returns the data source type name.
func (d *activeActiveSubscriptionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_subscription"
}

// Configure adds the provider configured client to the data source.
func (d *activeActiveSubscriptionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema defines the schema for the data source.
func (d *activeActiveSubscriptionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Active Active Subscription data source allows access to the details of an existing AA subscription within your Redis Enterprise Cloud account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the subscription",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the subscription to filter returned subscriptions",
				Required:    true,
			},
			"payment_method": schema.StringAttribute{
				Description: "Payment method for the requested subscription.",
				Computed:    true,
			},
			"payment_method_id": schema.StringAttribute{
				Description: "A valid payment method pre-defined in the current account",
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
				Description: "The ARN of the IAM role used by the subscription to access the AWS KMS customer managed key",
				Computed:    true,
			},
			"public_endpoint_access": schema.BoolAttribute{
				Description: "Whether public endpoint access is enabled for databases in the subscription",
				Computed:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: "A cloud provider string either GCP or AWS",
				Computed:    true,
			},
			"aws_account_id": schema.StringAttribute{
				Description: "AWS account ID associated with the subscription (only applicable for AWS subscriptions)",
				Computed:    true,
			},
			"resource_tags": schema.MapAttribute{
				Description: "A string/string map of tags assigned to the cloud resources created by this subscription.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
		// maintenance_windows and pricing are blocks (not nested attributes) because the
		// original SDKv2 schema used TypeList with Elem: &schema.Resource{}, which is a
		// block in protocol v5. Using a nested attribute would cause a runtime panic in
		// the muxed provider.
		Blocks: map[string]schema.Block{
			"maintenance_windows": schema.ListNestedBlock{
				Description: "Details about the subscription's maintenance window specification",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"mode": schema.StringAttribute{
							Description: "Either automatic (Redis specified) or manual (User specified)",
							Computed:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"window": schema.ListNestedBlock{
							Description: "A list of maintenance windows for manual-mode",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"start_hour": schema.Int64Attribute{
										Description: "What hour in the day (0-23) the window opens",
										Computed:    true,
									},
									"duration_in_hours": schema.Int64Attribute{
										Description: "How long the window is open",
										Computed:    true,
									},
									"days": schema.ListAttribute{
										Description: "A list of weekdays on which the window is open ('Monday', 'Tuesday' etc)",
										Computed:    true,
										ElementType: types.StringType,
									},
								},
							},
						},
					},
				},
			},
			"pricing": schema.ListNestedBlock{
				Description: "Pricing details totalled over this Subscription",
				NestedObject: schema.NestedBlockObject{
					// CustomType strongly types each pricing entry (see customtypes.PricingType)
					CustomType: customtypes.NewPricingType(),
					Attributes: map[string]schema.Attribute{
						"database_name": schema.StringAttribute{
							Description: "The database this pricing entry applies to",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of cost e.g. 'Shards'",
							Computed:    true,
						},
						"type_details": schema.StringAttribute{
							Description: "Further detail e.g. 'micro'",
							Computed:    true,
						},
						"quantity": schema.Int64Attribute{
							Description: "The number of units this pricing entry covers",
							Computed:    true,
						},
						"quantity_measurement": schema.StringAttribute{
							Description: "The unit that 'quantity' is measured in, e.g. 'shards'",
							Computed:    true,
						},
						"price_per_unit": schema.Float64Attribute{
							Description: "The cost of a single unit",
							Computed:    true,
						},
						"price_currency": schema.StringAttribute{
							Description: "The currency the price is denominated in, e.g. 'USD'",
							Computed:    true,
						},
						"price_period": schema.StringAttribute{
							Description: "The billing period the price applies to, e.g. 'hour'",
							Computed:    true,
						},
						"region": schema.StringAttribute{
							Description: "The region this cost is associated with, if any",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
