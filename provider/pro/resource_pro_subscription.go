package pro

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var (
	_ resource.Resource                 = &proSubscriptionResource{}
	_ resource.ResourceWithConfigure    = &proSubscriptionResource{}
	_ resource.ResourceWithImportState  = &proSubscriptionResource{}
	_ resource.ResourceWithModifyPlan   = &proSubscriptionResource{}
	_ resource.ResourceWithUpgradeState = &proSubscriptionResource{}
)

const (
	proSubscriptionCreateDefaultTimeout = 30 * time.Minute
	proSubscriptionReadDefaultTimeout   = 10 * time.Minute
	proSubscriptionUpdateDefaultTimeout = 30 * time.Minute
	proSubscriptionDeleteDefaultTimeout = 10 * time.Minute
)

type proSubscriptionResource struct {
	client *client.ApiClient
}

// NewProSubscriptionResource returns the rediscloud_subscription resource.
func NewProSubscriptionResource() resource.Resource {
	return &proSubscriptionResource{}
}

func (r *proSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription"
}

func (r *proSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = apiClient
}

func (r *proSubscriptionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Pro Subscription within your Redis Enterprise Cloud Account.",
		Version:     1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the subscription",
				Optional:    true,
			},
			"payment_method": schema.StringAttribute{
				Description: "Payment method for the requested subscription. If credit card is specified, the payment method id must be defined. This information is only used when creating a new subscription and any changes will be ignored after this.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("credit-card"),
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^(credit-card|marketplace)$"), "must be 'credit-card' or 'marketplace'"),
				},
				PlanModifiers: []planmodifier.String{
					createOnly(true),
				},
			},
			"payment_method_id": schema.StringAttribute{
				Description: "A valid payment method pre-defined in the current account",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a number"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"memory_storage": schema.StringAttribute{
				Description: "Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ram"),
				Validators: []validator.String{
					stringvalidator.OneOf(databases.MemoryStorageValues()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"redis_version": schema.StringAttribute{
				Description:        "Version of Redis to create",
				Optional:           true,
				DeprecationMessage: "This attribute is deprecated on pro subscriptions. Please specify `redis_version` on databases directly instead.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						redisVersionRequiresReplace,
						"Changing the Redis version of an existing subscription requires replacement. Removing the attribute is ignored.",
						"Changing the Redis version of an existing subscription requires replacement. Removing the attribute is ignored.",
					),
				},
			},
			"pricing": schema.ListAttribute{
				Description: "Pricing details totalled over this Subscription",
				Computed:    true,
				ElementType: utils.PricingElemType,
			},
			"customer_managed_key_enabled": schema.BoolAttribute{
				Description: "Whether to enable CMK (customer managed key) for the subscription. If this is true, then the subscription will be put in a pending state until you supply the CMEK. See documentation for further details on this process. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"customer_managed_key_deletion_grace_period": schema.StringAttribute{
				Description: "The grace period for deleting the subscription. If not set, will default to immediate deletion grace period.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("immediate"),
				PlanModifiers: []planmodifier.String{
					gracePeriodUpgradeModifier{},
				},
			},
			"customer_managed_key_redis_service_account": schema.StringAttribute{
				Description: "The principal of the Redis service account that the subscription is created in. This is used by the user to give access to their customer managed key",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_managed_key_aws_role_arn": schema.StringAttribute{
				Description: "The ARN of the IAM role used by the subscription to access the AWS KMS customer managed key. Grant this role access to your KMS key via key policy",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_endpoint_access": schema.BoolAttribute{
				Description: "Whether databases in the subscription should have public endpoints. When set to false, databases will only have private endpoints. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"prometheus_endpoint": schema.StringAttribute{
				Description: "The Prometheus scrape endpoint for databases in this subscription. Use this to configure your Prometheus server to scrape metrics from your Redis Cloud databases.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
			"allowlist": schema.ListNestedBlock{
				Description: "An allowlist object",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"cidrs": schema.SetAttribute{
							Description: "Set of CIDR ranges that are allowed to access the databases associated with this subscription",
							Optional:    true,
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(cidrValidator{}),
							},
						},
						"security_group_ids": schema.SetAttribute{
							Description: "Set of security groups that are allowed to access the databases associated with this subscription",
							Required:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"cloud_provider": schema.ListNestedBlock{
				Description: "A cloud provider object",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"provider": schema.StringAttribute{
							Description: "The cloud provider to use with the subscription, (either `AWS` or `GCP`)",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("AWS"),
							Validators: []validator.String{
								stringvalidator.OneOf(cloud_accounts.ProviderValues()...),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"cloud_account_id": schema.StringAttribute{
							Description: "Cloud account identifier. Default: Redis Labs internal cloud account (using Cloud Account Id = 1 implies using Redis Labs internal cloud account). Note that a GCP subscription can be created only with Redis Labs internal cloud account",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("1"),
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a number"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"aws_account_id": schema.StringAttribute{
							Description: "AWS account ID associated with the subscription (only applicable for AWS subscriptions)",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"resource_tags": schema.MapAttribute{
							Description: "A map of tags to associate with this subscription.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"region": schema.SetNestedBlock{
							Description: "Cloud networking details, per region (single region or multiple regions for Active-Active cluster only)",
							Validators: []validator.Set{
								setvalidator.IsRequired(),
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"region": schema.StringAttribute{
										Description: "Deployment region as defined by cloud provider",
										Required:    true,
									},
									"multiple_availability_zones": schema.BoolAttribute{
										Description: "Support deployment on multiple availability zones within the selected region",
										Optional:    true,
										Computed:    true,
									},
									"preferred_availability_zones": schema.ListAttribute{
										Description: "List of availability zones used",
										Optional:    true,
										Computed:    true,
										ElementType: types.StringType,
									},
									// TODO(TF3.0): Remove networking_deployment_cidr and networking_vpc_id from the region and read and write them through the networks object.
									"networking_deployment_cidr": schema.StringAttribute{
										Description: "Deployment CIDR mask",
										Required:    true,
										Validators: []validator.String{
											cidrValidator{},
										},
									},
									"networking_vpc_id": schema.StringAttribute{
										Description: "Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified)",
										Optional:    true,
										Computed:    true,
									},
									"networks": schema.ListAttribute{
										Description: "List of networks used",
										Computed:    true,
										ElementType: subscriptionNetworkElemType,
									},
								},
							},
						},
					},
				},
			},
			"creation_plan": schema.ListNestedBlock{
				Description: "Information about the planned databases used to optimise the database infrastructure. This information is only used when creating a new subscription and any changes will be ignored after this.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"memory_limit_in_gb": schema.Float64Attribute{
							Description: "(Deprecated) Maximum memory usage for each database",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Float64{
								float64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("dataset_size_in_gb")),
							},
							PlanModifiers: []planmodifier.Float64{createOnly(true)},
						},
						"dataset_size_in_gb": schema.Float64Attribute{
							Description: "Maximum amount of data in the dataset for this specific database in GB",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Float64{
								float64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("memory_limit_in_gb")),
							},
							PlanModifiers: []planmodifier.Float64{createOnly(true)},
						},
						"query_performance_factor": schema.StringAttribute{
							Description: "Query performance factor for this specific database",
							Optional:    true,
							Computed:    true,
							//TODO(TF3.0) add Default "Standard". The API only applies it on creation, so a default is a breaking change.
							PlanModifiers: []planmodifier.String{createOnly(true)},
						},
						"throughput_measurement_by": schema.StringAttribute{
							Description: "Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("number-of-shards", "operations-per-second"),
							},
							PlanModifiers: []planmodifier.String{createOnly(false)},
						},
						"throughput_measurement_value": schema.Int64Attribute{
							Description:   "Throughput value (as applies to selected measurement method)",
							Required:      true,
							PlanModifiers: []planmodifier.Int64{createOnly(false)},
						},
						"average_item_size_in_bytes": schema.Int64Attribute{
							Description:        "(Deprecated) Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database",
							Optional:           true,
							Computed:           true,
							DeprecationMessage: "Configure `ram_percentage` instead. This attribute will be removed in the next major version of the provider.",
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
								int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("ram_percentage")),
							},
							PlanModifiers: []planmodifier.Int64{createOnly(true)},
						},
						"ram_percentage": schema.Int64Attribute{
							Description: "Relevant only to ram-and-flash subscriptions. The percentage of data to be stored in RAM",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 100),
								int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("average_item_size_in_bytes")),
							},
							PlanModifiers: []planmodifier.Int64{createOnly(true)},
						},
						"quantity": schema.Int64Attribute{
							Description: "The planned number of databases",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
							PlanModifiers: []planmodifier.Int64{createOnly(false)},
						},
						"support_oss_cluster_api": schema.BoolAttribute{
							Description:   "Support Redis open-source (OSS) Cluster API",
							Optional:      true,
							Computed:      true,
							Default:       booldefault.StaticBool(false),
							PlanModifiers: []planmodifier.Bool{createOnly(true)},
						},
						"replication": schema.BoolAttribute{
							Description:   "Databases replication",
							Required:      true,
							PlanModifiers: []planmodifier.Bool{createOnly(false)},
						},
						"modules": schema.ListAttribute{
							Description:   "Modules that will be used by the databases in this subscription.",
							Optional:      true,
							Computed:      true,
							ElementType:   types.StringType,
							PlanModifiers: []planmodifier.List{createOnly(true)},
						},
					},
				},
			},
			"maintenance_windows": schema.ListNestedBlock{
				Description: "Specify the subscription's maintenance windows",
				CustomType:  customtypes.NewMaintenanceListType(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"mode": schema.StringAttribute{
							Description: "Either automatic (Redis specified) or manual (User specified)",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("automatic", "manual"),
							},
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
										Required:    true,
									},
									"duration_in_hours": schema.Int64Attribute{
										Description: "How long maintenance may take",
										Required:    true,
									},
									"days": schema.ListAttribute{
										Description: "A list of days on which the window is open ('Monday', 'Tuesday' etc)",
										Required:    true,
										ElementType: types.StringType,
										Validators: []validator.List{
											listvalidator.SizeBetween(1, 7),
										},
									},
								},
							},
						},
					},
				},
			},
			"customer_managed_key": schema.ListNestedBlock{
				Description: "CMK resources used to encrypt the databases in this subscription. Ignored if `customer_managed_key_enabled` set to false. Supply after the database has been put into database pending state. See documentation for CMK flow.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"resource_name": schema.StringAttribute{
							Description: "Resource name of the customer managed key as defined by the cloud provider.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *proSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	// The API only returns the grace period for CMK subscriptions; Read keeps this value otherwise.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("customer_managed_key_deletion_grace_period"), "immediate")...)
}

func (r *proSubscriptionResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradeSDKProSubscriptionState},
	}
}

// upgradeSDKProSubscriptionState reads a state stored by SDKv2 (version 0), whose
// attributes match the current schema. SDKv2 filled an omitted maintenance_windows
// block from the API, and a block cannot be kept in a plan when the configuration
// omits it, so a plan without refresh would remove it and reset the mode to automatic.
// The block is dropped here as Read drops it without a record of the configuration.
func upgradeSDKProSubscriptionState(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	raw, err := req.RawState.UnmarshalWithOpts(resp.State.Schema.Type().TerraformType(ctx), tfprotov6.UnmarshalOpts{
		ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the SDKv2 state", err.Error())
		return
	}
	resp.State.Raw = raw
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("maintenance_windows"), emptyMaintenanceList())...)
}

// redisVersionRequiresReplace mirrors the SDKv2 diff suppression: removing the
// deprecated attribute or keeping its value is ignored, any other change replaces.
func redisVersionRequiresReplace(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() == "" {
		return
	}
	resp.RequiresReplace = req.ConfigValue.ValueString() != req.StateValue.ValueString()
}

// createOnlyModifier keeps the prior state value on update, so that changes to
// create-only fields are ignored after creation, like the SDKv2 diff suppression did.
// Terraform accepts a prior value in place of a configured one, but a non-computed
// attribute removed from the configuration must still be planned as null.
type createOnlyModifier struct {
	computed bool
}

func createOnly(computed bool) createOnlyModifier {
	return createOnlyModifier{computed: computed}
}

func (m createOnlyModifier) Description(_ context.Context) string {
	return "Uses the prior state value for existing resources. Changes to this attribute are ignored after creation."
}

func (m createOnlyModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m createOnlyModifier) keepState(isCreate bool, config, state attr.Value) bool {
	if isCreate || state.IsNull() || state.IsUnknown() {
		return false
	}
	return m.computed || !config.IsNull()
}

func (m createOnlyModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if m.keepState(req.State.Raw.IsNull(), req.ConfigValue, req.StateValue) {
		resp.PlanValue = req.StateValue
	}
}

func (m createOnlyModifier) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if m.keepState(req.State.Raw.IsNull(), req.ConfigValue, req.StateValue) {
		resp.PlanValue = req.StateValue
	}
}

func (m createOnlyModifier) PlanModifyFloat64(_ context.Context, req planmodifier.Float64Request, resp *planmodifier.Float64Response) {
	if m.keepState(req.State.Raw.IsNull(), req.ConfigValue, req.StateValue) {
		resp.PlanValue = req.StateValue
	}
}

func (m createOnlyModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if m.keepState(req.State.Raw.IsNull(), req.ConfigValue, req.StateValue) {
		resp.PlanValue = req.StateValue
	}
}

func (m createOnlyModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if m.keepState(req.State.Raw.IsNull(), req.ConfigValue, req.StateValue) {
		resp.PlanValue = req.StateValue
	}
}

// gracePeriodUpgradeModifier keeps an empty prior grace period when the default is
// planned and CMK is disabled. Subscriptions created before the attribute existed have
// no value in state, and the API does not return one for them.
// TODO: remove this when customer_managed_key_deletion_grace_period is supported on the API side.
type gracePeriodUpgradeModifier struct{}

func (m gracePeriodUpgradeModifier) Description(_ context.Context) string {
	return "Ignores the default grace period for subscriptions without CMK that have no grace period in state."
}

func (m gracePeriodUpgradeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m gracePeriodUpgradeModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.StateValue.IsUnknown() || req.StateValue.ValueString() != "" {
		return
	}
	if req.PlanValue.ValueString() != "immediate" {
		return
	}
	// Terraform only accepts a null prior value in place of a configured value when the configuration is null.
	if req.StateValue.IsNull() && !req.ConfigValue.IsNull() {
		return
	}

	var cmkEnabled types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("customer_managed_key_enabled"), &cmkEnabled)...)
	if resp.Diagnostics.HasError() || cmkEnabled.ValueBool() {
		return
	}
	resp.PlanValue = req.StateValue
}

type cidrValidator struct{}

func (v cidrValidator) Description(_ context.Context) string {
	return "value must be a valid CIDR"
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if _, _, err := net.ParseCIDR(value); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid CIDR", fmt.Sprintf("expected %q to be a valid CIDR Value, got %v: %v", req.Path, value, err))
	}
}
