package activeactive

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &activeActiveDatabaseResource{}
	_ resource.ResourceWithConfigure   = &activeActiveDatabaseResource{}
	_ resource.ResourceWithImportState = &activeActiveDatabaseResource{}
	_ resource.ResourceWithModifyPlan  = &activeActiveDatabaseResource{}
)

// activeActiveDatabaseResource is the resource implementation.
type activeActiveDatabaseResource struct {
	client *client.ApiClient
}

// NewActiveActiveDatabaseResource returns a new resource instance.
func NewActiveActiveDatabaseResource() resource.Resource {
	return &activeActiveDatabaseResource{}
}

// Metadata returns the resource type name.
func (r *activeActiveDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_subscription_database"
}

// Configure adds the provider configured client to the resource.
func (r *activeActiveDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// useStateOnUpdateListModifier is a plan modifier that preserves the state value
// for existing resources. This implements "create-only" field behaviour in the
// Plugin Framework - the field can be set on create but changes are ignored after.
type useStateOnUpdateListModifier struct{}

var _ planmodifier.List = useStateOnUpdateListModifier{}

func (m useStateOnUpdateListModifier) Description(_ context.Context) string {
	return "Uses the prior state value for existing resources. Changes to this attribute are ignored after creation."
}

func (m useStateOnUpdateListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateOnUpdateListModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// If this is a create operation (no prior state), allow config value through
	if req.State.Raw.IsNull() {
		return
	}

	// For existing resources, always use the state value (ignoring config changes)
	resp.PlanValue = req.StateValue
}

// UseStateOnUpdate returns a plan modifier that uses state value for existing resources.
// This is the Plugin Framework equivalent of SDK v2's DiffSuppressFunc for create-only fields.
func UseStateOnUpdate() planmodifier.List {
	return useStateOnUpdateListModifier{}
}

// Schema defines the schema for the resource.
func (r *activeActiveDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Alert block schema (used in global_alert and override_global_alert)
	alertBlockSchema := schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Alert name",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(databases.AlertNameValues()...),
				},
			},
			"value": schema.Int64Attribute{
				Description: "Alert value",
				Required:    true,
			},
		},
	}

	// Remote backup block schema
	remoteBackupBlockSchema := schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"interval": schema.StringAttribute{
				Description: "Defines the frequency of the automatic backup",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(databases.BackupIntervals()...),
				},
			},
			"time_utc": schema.StringAttribute{
				Description: "Defines the hour automatic backups are made - only applicable when interval is `every-12-hours` or `every-24-hours`",
				Optional:    true,
				Validators: []validator.String{
					TimeValidator(),
				},
			},
			"storage_type": schema.StringAttribute{
				Description: "Defines the provider of the storage location",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(databases.BackupStorageTypes()...),
				},
			},
			"storage_path": schema.StringAttribute{
				Description: "Defines a URI representing the backup storage location",
				Required:    true,
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Creates database resource within an active-active subscription in your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the resource in the format `subscription_id/db_id`",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription_id": schema.Int64Attribute{
				Description: "Identifier of the subscription",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"db_id": schema.Int64Attribute{
				Description: "Identifier of the database created",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the database",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					StringLengthBetween(0, 40),
				},
			},
			"memory_limit_in_gb": schema.Float64Attribute{
				Description: "(Deprecated) Maximum memory usage for this specific database",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"dataset_size_in_gb": schema.Float64Attribute{
				Description: "Maximum amount of data in the dataset for this specific database in GB",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"redis_version": schema.StringAttribute{
				Description: "Defines the Redis database version. If omitted, the Redis version will be set to the default version",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"support_oss_cluster_api": schema.BoolAttribute{
				Description: "Support Redis open-source (OSS) Cluster API",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"external_endpoint_for_oss_cluster_api": schema.BoolAttribute{
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_tls": schema.BoolAttribute{
				Description: "Use TLS for authentication.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"client_ssl_certificate": schema.StringAttribute{
				Description: "SSL certificate to authenticate user connections.",
				Optional:    true,
				Sensitive:   true,
			},
			"client_tls_certificates": schema.ListAttribute{
				Description: "TLS certificates to authenticate user connections",
				Optional:    true,
				ElementType: types.StringType,
				Sensitive:   true,
				Validators: []validator.List{
					listvalidator.ConflictsWith(path.MatchRoot("client_ssl_certificate")),
				},
			},
			"data_eviction": schema.StringAttribute{
				Description: "Data eviction items policy",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("volatile-lru"),
			},
			"global_data_persistence": schema.StringAttribute{
				Description: "Rate of database data persistence (in persistent storage)",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"global_password": schema.StringAttribute{
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"global_source_ips": schema.SetAttribute{
				Description: "Set of CIDR addresses to allow access to the database",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"global_enable_default_user": schema.BoolAttribute{
				Description: "When 'true', enables connecting to the database with the 'default' user across all regions. Default: 'true'",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"global_resp_version": schema.StringAttribute{
				Description: "The initial RESP version for all databases provisioned under this AA database. This information is only used when creating a new database and any changes will be ignored after this.",
				Optional:    true,
				Validators: []validator.String{
					RespVersionValidator(),
				},
			},
			"global_modules": schema.ListAttribute{
				Description: "List of modules to enable on the database. This information is only used when creating a new database and any changes will be ignored after this.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					UseStateOnUpdate(),
				},
			},
			"public_endpoint": schema.MapAttribute{
				Description: "Region public endpoints to access the database",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"private_endpoint": schema.MapAttribute{
				Description: "Region private endpoints to access the database",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"port": schema.Int64Attribute{
				Description: "TCP port on which the database is available",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					PortRangeValidator(),
				},
			},
			"tags": schema.MapAttribute{
				Description: "Tags for database management",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"global_alert": schema.SetNestedBlock{
				Description:  "Set of alerts to enable on the database",
				NestedObject: alertBlockSchema,
			},
			"override_region": schema.SetNestedBlock{
				Description: "Region-specific configuration parameters to override the global configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Region name",
							Required:    true,
						},
						"override_global_password": schema.StringAttribute{
							Description: "Password used to access the database. If left empty, the password will be generated automatically",
							Optional:    true,
							Sensitive:   true,
						},
						"override_global_source_ips": schema.SetAttribute{
							Description: "Set of CIDR addresses to allow access to the database",
							Optional:    true,
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
						},
						"override_global_data_persistence": schema.StringAttribute{
							Description: "Rate of database data persistence (in persistent storage)",
							Optional:    true,
						},
						"enable_default_user": schema.BoolAttribute{
							Description: "When 'true', enables connecting to the database with the 'default' user. If not set, inherits from global_enable_default_user.",
							Optional:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"override_global_alert": schema.SetNestedBlock{
							Description:  "Set of alerts to enable on the database in this region",
							NestedObject: alertBlockSchema,
						},
						"remote_backup": schema.ListNestedBlock{
							Description:  "An object that specifies the backup options for the database in this region",
							NestedObject: remoteBackupBlockSchema,
						},
					},
				},
			},
		},
	}
}

// ModifyPlan implements custom plan modification logic.
func (r *activeActiveDatabaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If the plan is null (resource is being destroyed), skip validation
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ActiveActiveDatabaseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one of memory_limit_in_gb or dataset_size_in_gb is set
	memorySet := !plan.MemoryLimitInGB.IsNull() && !plan.MemoryLimitInGB.IsUnknown()
	datasetSet := !plan.DatasetSizeInGB.IsNull() && !plan.DatasetSizeInGB.IsUnknown()

	// Only validate on create (when ID is unknown) or when both values are explicitly set
	if plan.ID.IsUnknown() {
		if !memorySet && !datasetSet {
			resp.Diagnostics.AddError(
				"Missing required attribute",
				"One of 'memory_limit_in_gb' or 'dataset_size_in_gb' must be specified",
			)
		}
		if memorySet && datasetSet {
			resp.Diagnostics.AddError(
				"Conflicting attributes",
				"Only one of 'memory_limit_in_gb' or 'dataset_size_in_gb' may be specified",
			)
		}
	}

	// Validate that client_ssl_certificate and client_tls_certificates are not both set
	sslCertSet := !plan.ClientSSLCertificate.IsNull() && plan.ClientSSLCertificate.ValueString() != ""
	tlsCertsSet := !plan.ClientTLSCertificates.IsNull() && len(plan.ClientTLSCertificates.Elements()) > 0

	if sslCertSet && tlsCertsSet {
		resp.Diagnostics.AddError(
			"Conflicting attributes",
			"'client_ssl_certificate' and 'client_tls_certificates' cannot both be specified",
		)
	}

	// Validate backup interval and time_utc combinations in override_region blocks
	if !plan.OverrideRegion.IsNull() && !plan.OverrideRegion.IsUnknown() {
		var regions []OverrideRegionModel
		diags := plan.OverrideRegion.ElementsAs(ctx, &regions, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, region := range regions {
			if region.RemoteBackup.IsNull() || region.RemoteBackup.IsUnknown() {
				continue
			}

			var backups []RemoteBackupModel
			diags := region.RemoteBackup.ElementsAs(ctx, &backups, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			for _, backup := range backups {
				if !backup.TimeUTC.IsNull() && backup.TimeUTC.ValueString() != "" {
					interval := backup.Interval.ValueString()
					if interval != "every-12-hours" && interval != "every-24-hours" {
						resp.Diagnostics.AddError(
							"Invalid backup configuration",
							fmt.Sprintf("'time_utc' can only be set when 'interval' is 'every-12-hours' or 'every-24-hours', got: %s in region %s", interval, region.Name.ValueString()),
						)
					}
				}
			}
		}
	}

	// Suppress diff for global_modules after creation (only used on create)
	if !plan.ID.IsUnknown() && !plan.ID.IsNull() {
		var state ActiveActiveDatabaseModel
		diags := req.State.Get(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Note: UseStateOnUpdate() plan modifier ensures state value is preserved in plan
		// for global_modules - changes after creation are silently ignored as documented

		// Keep the state value for global_resp_version (changes are ignored after creation)
		if !state.GlobalRespVersion.IsNull() && !state.GlobalRespVersion.IsUnknown() {
			plan.GlobalRespVersion = state.GlobalRespVersion
		}

		// Mark public_endpoint as unknown if:
		// 1. Public endpoints are currently empty (public_endpoint_access was false), AND
		// 2. User is making config changes (to avoid perpetual drift on re-plans)
		//
		// Note: Only mark public_endpoint, not private_endpoint, since private endpoints
		// have real values even when public_endpoint_access=false.
		if !state.PublicEndpoint.IsNull() {
			var publicEndpoints map[string]string
			diags = state.PublicEndpoint.ElementsAs(ctx, &publicEndpoints, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			hasEmptyPublicEndpoints := false
			for _, v := range publicEndpoints {
				if v == "" {
					hasEmptyPublicEndpoints = true
					break
				}
			}

			if hasEmptyPublicEndpoints {
				// Check if user is making config changes by comparing config to state
				// for user-configurable fields. If config differs, this is a real update.
				var config ActiveActiveDatabaseModel
				diags = req.Config.Get(ctx, &config)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Detect config changes: check if global_modules in config differs from state
				configHasModules := !config.GlobalModules.IsNull() && len(config.GlobalModules.Elements()) > 0
				stateHasModules := !state.GlobalModules.IsNull() && len(state.GlobalModules.Elements()) > 0

				// If config has modules but state doesn't (or vice versa), user is making changes
				isConfigChange := configHasModules != stateHasModules

				if isConfigChange {
					// Real update - public endpoints might change, mark as unknown
					plan.PublicEndpoint = types.MapUnknown(types.StringType)
				}
				// If no config change, keep the state values (UseStateForUnknown behaviour)
			}
		}

		// Set the updated plan
		diags = resp.Plan.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
	}
}

// ImportState imports an existing resource.
func (r *activeActiveDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID (expected format: subscription_id/db_id)
	subId, dbId, err := parseResourceId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in format 'subscription_id/db_id', got: %s. Error: %s", req.ID, err.Error()),
		)
		return
	}

	// Set the ID and required fields
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), buildResourceId(subId, dbId))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subscription_id"), int64(subId))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("db_id"), int64(dbId))...)
}

// Create implements resource creation.
func (r *activeActiveDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ActiveActiveDatabaseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the CRUD implementation
	r.createDatabase(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read implements resource reading.
func (r *activeActiveDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ActiveActiveDatabaseModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the CRUD implementation
	removed := r.readDatabase(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if removed {
		resp.State.RemoveResource(ctx)
		return
	}

	// Set the state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update implements resource updating.
func (r *activeActiveDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ActiveActiveDatabaseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ActiveActiveDatabaseModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the ID from state
	plan.ID = state.ID
	plan.DbID = state.DbID

	// Call the CRUD implementation
	r.updateDatabase(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the state to get computed values
	r.readDatabase(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete implements resource deletion.
func (r *activeActiveDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ActiveActiveDatabaseModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the CRUD implementation
	r.deleteDatabase(ctx, &state, &resp.Diagnostics)
}

// timeouts returns the resource timeouts.
func (r *activeActiveDatabaseResource) timeouts() map[string]time.Duration {
	return map[string]time.Duration{
		"create": 30 * time.Minute,
		"read":   10 * time.Minute,
		"update": 30 * time.Minute,
		"delete": 10 * time.Minute,
	}
}

// getAlertAttrTypes returns the attribute types for alert objects.
func getAlertAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":  types.StringType,
		"value": types.Int64Type,
	}
}

// getRemoteBackupAttrTypes returns the attribute types for remote_backup objects.
func getRemoteBackupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interval":     types.StringType,
		"time_utc":     types.StringType,
		"storage_type": types.StringType,
		"storage_path": types.StringType,
	}
}

// overrideRegionHasConfig checks if an override region block has any configuration set
// beyond just the name (which is always required).
func overrideRegionHasConfig(region OverrideRegionModel) bool {
	if !region.OverrideGlobalAlert.IsNull() && len(region.OverrideGlobalAlert.Elements()) > 0 {
		return true
	}
	if !region.OverrideGlobalPassword.IsNull() && region.OverrideGlobalPassword.ValueString() != "" {
		return true
	}
	if !region.OverrideGlobalSourceIPs.IsNull() && len(region.OverrideGlobalSourceIPs.Elements()) > 0 {
		return true
	}
	if !region.OverrideGlobalDataPersistence.IsNull() && region.OverrideGlobalDataPersistence.ValueString() != "" {
		return true
	}
	if !region.RemoteBackup.IsNull() && len(region.RemoteBackup.Elements()) > 0 {
		return true
	}
	return false
}

// findOverrideRegion finds a region by name in the override_region set.
func findOverrideRegion(ctx context.Context, overrideRegionSet types.Set, regionName string) (*OverrideRegionModel, error) {
	if overrideRegionSet.IsNull() || overrideRegionSet.IsUnknown() {
		return nil, nil
	}

	var regions []OverrideRegionModel
	diags := overrideRegionSet.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		return nil, fmt.Errorf("error reading override_region: %v", diags)
	}

	for _, region := range regions {
		if strings.EqualFold(region.Name.ValueString(), regionName) {
			return &region, nil
		}
	}

	return nil, nil
}
