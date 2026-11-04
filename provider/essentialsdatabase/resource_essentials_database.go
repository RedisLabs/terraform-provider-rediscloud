package essentialsdatabase

import (
	"context"
	"fmt"

	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &essentialsDatabaseResource{}
	_ resource.ResourceWithConfigure   = &essentialsDatabaseResource{}
	_ resource.ResourceWithImportState = &essentialsDatabaseResource{}
)

type essentialsDatabaseResource struct {
	client *client.ApiClient
}

// NewEssentialsDatabaseResource returns a new resource instance.
func NewEssentialsDatabaseResource() resource.Resource {
	return &essentialsDatabaseResource{}
}

// Metadata returns the resource type name.
func (r *essentialsDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_essentials_database"
}

// Configure adds the provider configured client to the resource.
func (r *essentialsDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema defines the schema for the resource.
//
// SDKv2 suppressed diffs on the Pay-As-You-Go attributes while enable_payg_features was false. Terraform requires
// a configured value to be planned as is, so these attributes are Optional+Computed and Read keeps their prior value
// while Pay-As-You-Go features are disabled.
func (r *essentialsDatabaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates database resource within an essentials subscription in your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the database in the format `subscription_id/db_id`",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			//TODO(TF3.0) refactor subscription_id to a StringAttribute
			"subscription_id": schema.Int64Attribute{
				Description: "Identifier of the essentials subscription",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			//TODO(TF3.0) refactor db_id to a StringAttribute
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
				Description: "The protocol that will be used to access the database, (either 'redis', 'memcached' or 'stack')",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(fixedDatabases.ProtocolValues()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"redis_version": schema.StringAttribute{
				Description: "Defines the Redis database version. If omitted, the Redis version will be set to the default version",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"redis_version_actual": schema.StringAttribute{
				Description: "The actual Redis database version",
				Computed:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: "The Cloud Provider hosting this database",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The region within the Cloud Provider where this database is hosted",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"redis_version_compliance": schema.StringAttribute{
				Description: "The compliance level (redis version) of this database",
				Computed:    true,
			},
			"resp_version": schema.StringAttribute{
				Description: "RESP version must be compatible with Redis version.",
				Optional:    true,
				Computed:    true,
			},
			"data_persistence": schema.StringAttribute{
				Description: "Rate of database data persistence (in persistent storage).",
				Required:    true,
			},
			"data_eviction": schema.StringAttribute{
				Description: "The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("volatile-lru"),
			},
			"replication": schema.BoolAttribute{
				Description: "Database's replication",
				Required:    true,
			},
			"activated_on": schema.StringAttribute{
				Description: "When this database was activated",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"periodic_backup_path": schema.StringAttribute{
				Description: "Path that will be used to store database backup files",
				Optional:    true,
			},
			"public_endpoint": schema.StringAttribute{
				Description: "Public endpoint to access the database",
				Computed:    true,
			},
			"private_endpoint": schema.StringAttribute{
				Description: "Private endpoint to access the database",
				Computed:    true,
			},
			"source_ips": schema.ListAttribute{
				Description: "Set of CIDR addresses to allow access to the database.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(cidrValidator{}),
				},
				PlanModifiers: []planmodifier.List{
					emptyListUnlessConfigured(),
				},
			},
			"client_tls_certificates": schema.ListAttribute{
				Description: "",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					emptyListUnlessConfigured(),
				},
			},
			"password": schema.StringAttribute{
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
			},
			"enable_default_user": schema.BoolAttribute{
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			// SDKv2 used ConfigMode: SchemaConfigModeAttr, so modules is configured with attribute syntax.
			"modules": schema.SetAttribute{
				Description: "Modules to be provisioned in the database. Note: Not supported for Redis 8.0 and higher as modules are bundled by default.",
				Optional:    true,
				Computed:    true,
				ElementType: moduleElemType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
					setplanmodifier.RequiresReplace(),
				},
			},
			"enable_payg_features": schema.BoolAttribute{
				Description: "Enable features for PAYG databases",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"memory_limit_in_gb": schema.Float64Attribute{
				Description: "Maximum memory usage for this specific database",
				Optional:    true,
				Computed:    true,
			},
			"support_oss_cluster_api": schema.BoolAttribute{
				Description: "Support Redis open-source (OSS) Cluster API",
				Optional:    true,
				Computed:    true,
			},
			"external_endpoint_for_oss_cluster_api": schema.BoolAttribute{
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Optional:    true,
				Computed:    true,
			},
			"enable_database_clustering": schema.BoolAttribute{
				Description: "Distributes database data to different cloud instances",
				Optional:    true,
				Computed:    true,
			},
			// Regex rules are not validated: the service expects patterns such as `(?<tag>.*)`, which are not valid Go regexes.
			"regex_rules": schema.ListAttribute{
				Description: "Shard regex rules. Relevant only for a sharded database. Supported only for 'Pay-As-You-Go' subscriptions",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"enable_tls": schema.BoolAttribute{
				Description: "Use TLS for authentication",
				Optional:    true,
				Computed:    true,
			},
			"tags": schema.MapAttribute{
				Description: "Tags for database management",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					lowerCaseTagsValidator{},
				},
				PlanModifiers: []planmodifier.Map{
					emptyMapUnlessConfigured(),
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
			"replica": schema.ListNestedBlock{
				Description: "Details of database replication",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"sync_source": schema.SetNestedBlock{
							Description: "A source database to replicate here",
							Validators: []validator.Set{
								setvalidator.IsRequired(),
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"endpoint": schema.StringAttribute{
										Description: "A Redis URI (sample format: 'redis://user:password@host:port)'. If the URI provided is Redis Cloud instance, only host and port should be provided (using the format: ['redis://endpoint1:6379', 'redis://endpoint2:6380'])",
										Required:    true,
									},
									"encryption": schema.BoolAttribute{
										Description: "Defines if encryption should be used to connect to the sync source. If not provided and if the source is a Redis Cloud instance, it will automatically detect if the source uses encryption",
										Optional:    true,
									},
									"server_cert": schema.StringAttribute{
										Description: "TLS/SSL certificate chain of the sync source. If left null and if the source is a Redis Cloud instance, it will automatically detect the certificate to use",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
			"alert": schema.SetNestedBlock{
				Description: "Set of alerts to enable on the database",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Alert name",
							Required:    true,
						},
						"value": schema.Int64Attribute{
							Description: "Alert value",
							Required:    true,
						},
					},
				},
			},
		},
	}
}
