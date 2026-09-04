package pro

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

var (
	_ datasource.DataSource              = &proSubscriptionDataSource{}
	_ datasource.DataSourceWithConfigure = &proSubscriptionDataSource{}
)

// proSubscriptionDataSource is the data source implementation.
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
	// Reuse the shared per-subscription attribute set.
	// Then mark "name" Optional so it can be used as a filter.
	attributes := subscriptionDataSourceAttributes()
	attributes["name"] = schema.StringAttribute{
		Description: "The name of the subscription to filter returned subscriptions",
		Computed:    true,
		Optional:    true,
	}

	resp.Schema = schema.Schema{
		Description: "The Pro Subscription data source allows access to the details of an existing pro subscription within your Redis Enterprise Cloud account.",
		Attributes:  attributes,
		Blocks: map[string]schema.Block{
			"cloud_provider":      cloudProviderBlock(),
			"maintenance_windows": maintenanceWindowsBlock(),
			"pricing":             pricingBlock(),
		},
	}
}

// subscriptionDataSourceAttributes returns the attribute set describing a single pro subscription
func subscriptionDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "The identifier of the subscription",
			Computed:    true,
		},
		"name": schema.StringAttribute{
			Description: "The name of the subscription",
			Computed:    true,
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
	}
}

// Every nested structure is a block (not a nested attribute) because the original SDKv2
// schema used TypeList/TypeSet with Elem: &schema.Resource{}, which is a block in
// protocol v5. Using a nested attribute would cause a runtime panic in the muxed
// provider. Note: region was a TypeSet, so it becomes a SetNestedBlock.
func cloudProviderBlock() schema.Block {
	return schema.ListNestedBlock{
		Description: "A cloud provider object",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"provider": schema.StringAttribute{
					Description: "The cloud provider to use with the subscription, (either `AWS` or `GCP`)",
					Computed:    true,
				},
				"cloud_account_id": schema.StringAttribute{
					Description: "Cloud account identifier, (A Cloud Account Id = 1 implies using Redis Labs internal cloud account)",
					Computed:    true,
				},
				"aws_account_id": schema.StringAttribute{
					Description: "AWS account ID associated with the subscription (only applicable for AWS subscriptions)",
					Computed:    true,
				},
				"resource_tags": schema.MapAttribute{
					Description: "A map of resource tags associated with this subscription.",
					Computed:    true,
					ElementType: types.StringType,
				},
			},
			Blocks: map[string]schema.Block{
				"region": schema.SetNestedBlock{
					Description: "Cloud networking details, per region",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"region": schema.StringAttribute{
								Description: "Deployment region as defined by cloud provider",
								Computed:    true,
							},
							"multiple_availability_zones": schema.BoolAttribute{
								Description: "Support deployment on multiple availability zones within the selected region",
								Computed:    true,
							},
							"preferred_availability_zones": schema.ListAttribute{
								Description: "List of availability zones used",
								Computed:    true,
								ElementType: types.StringType,
							},
							"networking_vpc_id": schema.StringAttribute{
								Description: "The ID of the VPC where the Redis Cloud subscription is deployed",
								Computed:    true,
							},
						},
						Blocks: map[string]schema.Block{
							"networks": schema.ListNestedBlock{
								Description: "List of networks used",
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"networking_subnet_id": schema.StringAttribute{
											Description: "The subnet that the subscription deploys into",
											Computed:    true,
										},
										"networking_deployment_cidr": schema.StringAttribute{
											Description: "Deployment CIDR mask",
											Computed:    true,
										},
										"networking_vpc_id": schema.StringAttribute{
											Description: "Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified)",
											Computed:    true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Every nested structure is a block (not a nested attribute) because the original SDKv2
// schema used TypeList/TypeSet with Elem: &schema.Resource{}, which is a block in
// protocol v5. Using a nested attribute would cause a runtime panic in the muxed
// provider. Note: region was a TypeSet, so it becomes a SetNestedBlock.
func maintenanceWindowsBlock() schema.Block {
	return schema.ListNestedBlock{
		Description: "Details about the subscription's maintenance window specification",
		CustomType:  customtypes.NewMaintenanceListType(),
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
					CustomType:  customtypes.NewMaintenanceWindowListType(),
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"start_hour": schema.Int64Attribute{
								Description: "What hour in the day (0-23) may maintenance start",
								Computed:    true,
							},
							"duration_in_hours": schema.Int64Attribute{
								Description: "How long maintenance may take",
								Computed:    true,
							},
							"days": schema.ListAttribute{
								Description: "A list of days on which the window is open ('Monday', 'Tuesday' etc)",
								Computed:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
		},
	}
}

func pricingBlock() schema.Block {
	return schema.ListNestedBlock{
		Description: "Pricing details totalled over this Subscription",
		NestedObject: schema.NestedBlockObject{
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
	}
}
