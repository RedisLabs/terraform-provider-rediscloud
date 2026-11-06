package activeactive

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

const (
	activeActiveSubscriptionCreateDefaultTimeout = 30 * time.Minute
	activeActiveSubscriptionReadDefaultTimeout   = 10 * time.Minute
	activeActiveSubscriptionUpdateDefaultTimeout = 30 * time.Minute
	activeActiveSubscriptionDeleteDefaultTimeout = 10 * time.Minute
)

var (
	_ resource.Resource                = &activeActiveSubscriptionResource{}
	_ resource.ResourceWithConfigure   = &activeActiveSubscriptionResource{}
	_ resource.ResourceWithImportState = &activeActiveSubscriptionResource{}
	_ resource.ResourceWithModifyPlan  = &activeActiveSubscriptionResource{}
)

// activeActiveSubscriptionResource is the Framework implementation of
// rediscloud_active_active_subscription.
type activeActiveSubscriptionResource struct {
	client *client.ApiClient
}

// NewActiveActiveSubscriptionResource returns a new resource instance.
func NewActiveActiveSubscriptionResource() resource.Resource {
	return &activeActiveSubscriptionResource{}
}

func (r *activeActiveSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_subscription"
}

func (r *activeActiveSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// useStateOnUpdateStringModifier ignores configuration changes after create.
// The attribute must be Computed so the framework allows the planned value to
// be replaced with state. This is the string form of UseStateOnUpdate.
type useStateOnUpdateStringModifier struct{}

var _ planmodifier.String = useStateOnUpdateStringModifier{}

func (m useStateOnUpdateStringModifier) Description(_ context.Context) string {
	return "Uses the prior state value for existing resources. Changes to this attribute are ignored after creation."
}

func (m useStateOnUpdateStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateOnUpdateStringModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

func useStateOnUpdateString() planmodifier.String {
	return useStateOnUpdateStringModifier{}
}

// redisVersionModifier keeps the SDKv2 diff suppression: removing the deprecated
// attribute does not recreate the subscription, and a different non-empty value does.
type redisVersionModifier struct{}

var _ planmodifier.String = redisVersionModifier{}

func (m redisVersionModifier) Description(_ context.Context) string {
	return "Recreates the subscription only when redis_version changes to another non-empty value. Removing the deprecated attribute keeps the stored value."
}

func (m redisVersionModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// keepStateWhenUnsetListModifier preserves a stored list when configuration
// omits it. SDKv2 Optional+Computed and Computed lists behave that way.
// UseStateForUnknown does not, because an omitted block is null rather than unknown.
type keepStateWhenUnsetListModifier struct{}

var _ planmodifier.List = keepStateWhenUnsetListModifier{}

func (m keepStateWhenUnsetListModifier) Description(_ context.Context) string {
	return "Keeps the stored list when configuration omits it."
}

func (m keepStateWhenUnsetListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m keepStateWhenUnsetListModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if req.ConfigValue.IsNull() || req.PlanValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}

func keepStateWhenUnsetList() planmodifier.List {
	return keepStateWhenUnsetListModifier{}
}

func (m redisVersionModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	configValue := ""
	if !req.ConfigValue.IsNull() {
		configValue = req.ConfigValue.ValueString()
	}
	if configValue == "" {
		if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
			resp.PlanValue = req.StateValue
		}
		return
	}

	stateValue := ""
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		stateValue = req.StateValue.ValueString()
	}
	if configValue != stateValue {
		resp.RequiresReplace = true
	}
}

func (r *activeActiveSubscriptionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates an Active-Active Subscription within your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the subscription",
				Computed:    true,
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
					stringvalidator.OneOf("credit-card", "marketplace"),
				},
				PlanModifiers: []planmodifier.String{
					useStateOnUpdateString(),
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
			"cloud_provider": schema.StringAttribute{
				Description: "A cloud provider string either GCP or AWS",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("AWS"),
				Validators: []validator.String{
					stringvalidator.OneOf("GCP", "AWS"),
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
			"redis_version": schema.StringAttribute{
				Description:        "Version of Redis to create",
				Optional:           true,
				Computed:           true,
				DeprecationMessage: "This attribute is deprecated on Active-Active subscriptions. Please specify `redis_version` on databases directly instead.",
				PlanModifiers: []planmodifier.String{
					redisVersionModifier{},
				},
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
			"resource_tags": schema.MapAttribute{
				Description: "A string/string map of tags to assign to the cloud resources created by this subscription.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"creation_plan": schema.ListNestedBlock{
				Description: "Information about the planned databases used to optimise the database infrastructure. This information is only used when creating a new subscription and any changes will be ignored after this.",
				CustomType:  NewCreationPlanListType(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				PlanModifiers: []planmodifier.List{
					UseStateOnUpdate(),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"memory_limit_in_gb": schema.Float64Attribute{
							Description: "(Deprecated) Maximum memory usage for this specific database",
							Optional:    true,
							Validators: []validator.Float64{
								float64validator.ConflictsWith(
									path.MatchRelative().AtParent().AtName("dataset_size_in_gb"),
								),
							},
						},
						"dataset_size_in_gb": schema.Float64Attribute{
							Description: "Maximum amount of data in the dataset for this specific database in GB",
							Optional:    true,
							Validators: []validator.Float64{
								float64validator.ConflictsWith(
									path.MatchRelative().AtParent().AtName("memory_limit_in_gb"),
								),
							},
						},
						"quantity": schema.Int64Attribute{
							Description: "The planned number of databases",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"modules": schema.ListAttribute{
							Description: "Modules that will be used by the databases in this subscription.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"region": schema.SetNestedBlock{
							Description: "Cloud networking details, per region (multiple regions for Active-Active cluster)",
							CustomType:  NewCreationPlanRegionSetType(),
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"region": schema.StringAttribute{
										Description: "Deployment region as defined by cloud provider",
										Required:    true,
									},
									"networking_deployment_cidr": schema.StringAttribute{
										Description: "Deployment CIDR mask",
										Required:    true,
										Validators: []validator.String{
											CIDRValidator(),
										},
									},
									"write_operations_per_second": schema.Int64Attribute{
										Description: "Write operations per second for creation plan databases",
										Required:    true,
									},
									"read_operations_per_second": schema.Int64Attribute{
										Description: "Read operations per second for creation plan databases",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"maintenance_windows": schema.ListNestedBlock{
				Description: "Specify the subscription's maintenance windows",
				CustomType:  customtypes.NewMaintenanceListType(),
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.List{
					keepStateWhenUnsetList(),
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
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
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
			"pricing": schema.ListNestedBlock{
				Description: "Pricing details totalled over this Subscription",
				PlanModifiers: []planmodifier.List{
					keepStateWhenUnsetList(),
				},
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
							Description: "Self-explanatory",
							Computed:    true,
						},
						"quantity_measurement": schema.StringAttribute{
							Description: "Self-explanatory",
							Computed:    true,
						},
						"price_per_unit": schema.Float64Attribute{
							Description: "Self-explanatory",
							Computed:    true,
						},
						"price_currency": schema.StringAttribute{
							Description: "Self-explanatory e.g. 'USD'",
							Computed:    true,
						},
						"price_period": schema.StringAttribute{
							Description: "Self-explanatory e.g. 'hour'",
							Computed:    true,
						},
						"region": schema.StringAttribute{
							Description: "Self-explanatory, if the cost is associated with a particular region",
							Computed:    true,
						},
					},
				},
			},
			"customer_managed_key": schema.ListNestedBlock{
				Description: "CMK resources used to encrypt the databases in this subscription. Ignored if `customer_managed_key_enabled` set to false. See documentation for CMK flow.",
				CustomType:  NewCustomerManagedKeyListType(),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"resource_name": schema.StringAttribute{
							Description: "Resource name of the customer managed key as defined by the cloud provider, e.g. projects/PROJECT_ID/locations/LOCATION/keyRings/KEY_RING/cryptoKeys/KEY_NAME",
							Required:    true,
						},
						"region": schema.StringAttribute{
							Description: "Name of region for the customer managed key as defined by the cloud provider.",
							Required:    true,
						},
					},
				},
			},
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// ModifyPlan requires creation_plan while the subscription is being created and
// keeps the SDKv2 grace-period diff suppression for state written before that
// attribute existed.
func (r *activeActiveSubscriptionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ActiveActiveSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if req.State.Raw.IsNull() {
		if plan.CreationPlan.IsNull() || (!plan.CreationPlan.IsUnknown() && len(plan.CreationPlan.Elements()) == 0) {
			resp.Diagnostics.AddError(
				`the "creation_plan" block is required`,
				`the "creation_plan" block is required`,
			)
			return
		}
		if !plan.CreationPlan.IsUnknown() {
			plans, diags := plan.CreationPlan.AsModels(ctx)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			for _, creationPlan := range plans {
				if creationPlan.Region.IsNull() || (!creationPlan.Region.IsUnknown() && len(creationPlan.Region.Elements()) == 0) {
					resp.Diagnostics.AddError(
						`the "creation_plan.region" block is required`,
						`the "creation_plan.region" block is required`,
					)
					return
				}
			}
		}
		return
	}

	var state ActiveActiveSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if gracePeriodUpgradeSuppressed(state, plan) {
		plan.CustomerManagedKeyDeletionGracePeriod = state.CustomerManagedKeyDeletionGracePeriod
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func gracePeriodUpgradeSuppressed(state, plan ActiveActiveSubscriptionResourceModel) bool {
	grace := state.CustomerManagedKeyDeletionGracePeriod
	if grace.IsNull() || grace.IsUnknown() || grace.ValueString() != "" {
		return false
	}
	planned := plan.CustomerManagedKeyDeletionGracePeriod
	if planned.IsNull() || planned.IsUnknown() || planned.ValueString() != "immediate" {
		return false
	}
	enabled := plan.CustomerManagedKeyEnabled
	if enabled.IsUnknown() {
		return false
	}
	return enabled.IsNull() || !enabled.ValueBool()
}

func (r *activeActiveSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
