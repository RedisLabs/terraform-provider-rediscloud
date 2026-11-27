package pro

import (
	"context"
	"fmt"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/databases"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var (
	_ resource.Resource                = &proDatabaseResource{}
	_ resource.ResourceWithConfigure   = &proDatabaseResource{}
	_ resource.ResourceWithImportState = &proDatabaseResource{}
	_ resource.ResourceWithModifyPlan  = &proDatabaseResource{}
)

const (
	proDatabaseCreateDefaultTimeout = 30 * time.Minute
	proDatabaseReadDefaultTimeout   = 10 * time.Minute
	proDatabaseUpdateDefaultTimeout = 30 * time.Minute
	proDatabaseDeleteDefaultTimeout = 10 * time.Minute
)

// proDatabaseResource is the framework implementation of rediscloud_subscription_database.
type proDatabaseResource struct {
	client                    *client.ApiClient
	api                       proDatabaseAPI
	waitForSubscriptionActive func(ctx context.Context, subscriptionID int, timeout time.Duration) error
	waitForDatabaseActive     func(ctx context.Context, subscriptionID, databaseID int, timeout time.Duration) error
}

// NewProDatabaseResource returns the framework pro database resource.
func NewProDatabaseResource() resource.Resource {
	return &proDatabaseResource{}
}

func (r *proDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription_database"
}

func (r *proDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.api = liveProDatabaseAPI{client: apiClient}
}

func (r *proDatabaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates database resource within a pro subscription in your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the resource in the format `subscription_id/db_id`",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription_id": schema.Int64Attribute{
				Description: "Identifier of the pro subscription",
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
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 40),
				},
			},
			"protocol": schema.StringAttribute{
				Description: "The protocol that will be used to access the database (either ‘redis’ or 'memcached’)",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("redis"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(databases.ProtocolValues()...),
				},
			},
			"memory_limit_in_gb": schema.Float64Attribute{
				Description: "(Deprecated) Maximum memory usage for this specific database",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Float64{
					float64validator.ExactlyOneOf(path.MatchRoot("dataset_size_in_gb")),
				},
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"dataset_size_in_gb": schema.Float64Attribute{
				Description: "Maximum amount of data in the dataset for this specific database in GB",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Float64{
					float64validator.ExactlyOneOf(path.MatchRoot("memory_limit_in_gb")),
				},
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"redis_version": schema.StringAttribute{
				Description: "Defines the requested Redis database version. If omitted, the Redis version will be set to the default version. Use `redis_version_actual` to get the version on which the database is running.",
				Optional:    true,
				//TODO(TF3.0) drop Computed and stop writing redis_version from Read — makes the field write-only-by-convention (no plugin-framework migration), so state only ever holds what the user wrote in config and auto-upgrades can't drift state. DSF stays to heal state poisoned by older provider versions.
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"redis_version_actual": schema.StringAttribute{
				Description: "The actual Redis database version",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"support_oss_cluster_api": schema.BoolAttribute{
				Description: "Support Redis open-source (OSS) Cluster API",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"resp_version": schema.StringAttribute{
				Description: "The database's RESP version",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_endpoint_for_oss_cluster_api": schema.BoolAttribute{
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"data_persistence": schema.StringAttribute{
				Description: "Rate of database data persistence (in persistent storage)",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("none"),
			},
			"data_eviction": schema.StringAttribute{
				Description: "(Optional) The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("volatile-lru"),
			},
			"replication": schema.BoolAttribute{
				Description: "Databases replication",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"throughput_measurement_by": schema.StringAttribute{
				Description: "Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("number-of-shards", "operations-per-second"),
				},
			},
			"throughput_measurement_value": schema.Int64Attribute{
				Description: "Throughput value (as applies to selected measurement method)",
				Required:    true,
			},
			"average_item_size_in_bytes": schema.Int64Attribute{
				Description:        "Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database",
				Optional:           true,
				Computed:           true,
				Default:            int64default.StaticInt64(0),
				DeprecationMessage: "Configure `ram_percentage` instead. This attribute will be removed in the next major version of the provider.",
				Validators: []validator.Int64{
					int64validator.ConflictsWith(path.MatchRoot("ram_percentage")),
				},
			},
			"ram_percentage": schema.Int64Attribute{
				Description: "Relevant only to ram-and-flash subscriptions. The percentage of data to be stored in RAM",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
					int64validator.ConflictsWith(path.MatchRoot("average_item_size_in_bytes")),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Description: "Password used to access the database. If omitted (and enable_passwordless is false), a random password will be generated automatically. Cannot be used together with enable_passwordless",
				Optional:    true,
				Sensitive:   true,
				Computed:    true,
			},
			"enable_passwordless": schema.BoolAttribute{
				Description: "When 'true', the database is configured without a password. Only valid when the subscription has public_endpoint_access disabled. Cannot be used together with password. Default: 'false'",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"public_endpoint": schema.StringAttribute{
				Description: "Public endpoint to access the database",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_endpoint": schema.StringAttribute{
				Description: "Private endpoint to access the database",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_ssl_certificate": schema.StringAttribute{
				Description: "SSL certificate to authenticate user connections",
				Optional:    true,
				//TODO(TF3.0) make client_ssl_certificate nullable
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("client_tls_certificates")),
				},
			},
			"client_tls_certificates": schema.ListAttribute{
				Description: "TLS certificates to authenticate user connections",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ConflictsWith(path.MatchRoot("client_ssl_certificate")),
				},
			},
			"periodic_backup_path": schema.StringAttribute{
				Description:        "Path that will be used to store database backup files",
				Optional:           true,
				DeprecationMessage: "Use `remote_backup` block instead",
				//TODO(TF3.0) make periodic_backup_path nullable
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"replica_of": schema.SetAttribute{
				Description: "Set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(redisURLValidator{}),
				},
			},
			"query_performance_factor": schema.StringAttribute{
				Description: "Query performance factor for this specific database",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_ips": schema.SetAttribute{
				Description: "Set of CIDR addresses to allow access to the database",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(cidrValidator{}),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"hashing_policy": schema.ListAttribute{
				Description: "List of regular expression rules to shard the database by. See the documentation on clustering for more information on the hashing policy - https://docs.redislabs.com/latest/rc/concepts/clustering/",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"enable_tls": schema.BoolAttribute{
				Description: "Use TLS for authentication",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_default_user": schema.BoolAttribute{
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"auto_minor_version_upgrade": schema.BoolAttribute{
				Description: "Automatically upgrades the database to newer minor versions within the same major release. Applies to version 8.4 and above. Default: 'true'",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"port": schema.Int64Attribute{
				Description: "TCP port on which the database is available",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.Between(10000, 19999),
				},
			},
			"tags": schema.MapAttribute{
				Description: "Tags for database management",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					lowercaseTagsValidator{},
				},
			},
			// SDKv2 stored modules as a set attribute (ConfigModeAttr), not a
			// block. A nested block would change configuration syntax and the
			// protocol-v5 state type. A nested attribute needs protocol v6.
			"modules": schema.SetAttribute{
				Description: "Modules to be provisioned in the database. Note: NOT supported for Redis 8.0 and higher as modules are bundled by default.",
				Optional:    true,
				ElementType: moduleElementType,
				CustomType:  NewModuleSetType(),
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					moduleSetValidator{},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
			"alert": schema.SetNestedBlock{
				Description: "Set of alerts to enable on the database",
				CustomType:  NewAlertSetType(),
				NestedObject: schema.NestedBlockObject{
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
				},
			},
			"remote_backup": schema.ListNestedBlock{
				Description: "An object that specifies the backup options for the database",
				CustomType:  NewRemoteBackupListType(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
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
								timeValidator{},
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
				},
			},
		},
	}
}

func (r *proDatabaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan ProDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var config ProDatabaseModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasState := !req.State.Raw.IsNull()
	var state ProDatabaseModel
	if hasState {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	adjusted := adjustProDatabasePlan(ctx, config, plan, state, hasState)
	resp.Diagnostics.Append(adjusted.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.RequiresReplace = append(resp.RequiresReplace, adjusted.RequiresReplace...)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &adjusted.Plan)...)
}

func (r *proDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	subID, dbID, err := ToDatabaseId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected import ID in format 'subscription_id/db_id', got: %s. Error: %s", req.ID, err.Error()))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), utils.BuildResourceId(subID, dbID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subscription_id"), int64(subID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("db_id"), int64(dbID))...)
}

func (r *proDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diags := plan.Timeouts.Create(ctx, proDatabaseCreateDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	r.createDatabase(ctx, &plan, &resp.Diagnostics, createTimeout)
	// A known id means the database exists. Persist the whole plan, including
	// when a later wait fails, so Terraform keeps the remote object.
	if resp.Diagnostics.HasError() {
		if !plan.ID.IsNull() && !plan.ID.IsUnknown() {
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *proDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readTimeout, diags := state.Timeouts.Read(ctx, proDatabaseReadDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	removed := r.readDatabase(ctx, &state, &resp.Diagnostics, readTimeout)
	if resp.Diagnostics.HasError() {
		return
	}
	if removed {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *proDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state ProDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	plan.DbID = state.DbID

	updateTimeout, diags := plan.Timeouts.Update(ctx, proDatabaseUpdateDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	r.updateDatabase(ctx, &plan, &state, &resp.Diagnostics, updateTimeout)
	if resp.Diagnostics.HasError() {
		if !plan.ID.IsNull() && !plan.ID.IsUnknown() {
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
		return
	}
	r.readDatabase(ctx, &plan, &resp.Diagnostics, updateTimeout)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *proDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteTimeout, diags := state.Timeouts.Delete(ctx, proDatabaseDeleteDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()
	r.deleteDatabase(ctx, &state, &resp.Diagnostics, deleteTimeout)
}
