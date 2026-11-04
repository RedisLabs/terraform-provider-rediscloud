package pro

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &proDatabaseDataSource{}
	_ datasource.DataSourceWithConfigure = &proDatabaseDataSource{}
)

type proDatabaseDataSource struct {
	client *client.ApiClient
}

// NewProDatabaseDataSource provides the Framework implementation for rediscloud_database.
func NewProDatabaseDataSource() datasource.DataSource {
	return &proDatabaseDataSource{}
}

func (d *proDatabaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (d *proDatabaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Object collections are computed attributes, not blocks: SDKv2 exposed these
// computed-only collections as attributes, and blocks plan as empty lists when
// the read is deferred to apply.
func (d *proDatabaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Pro Database data source allows access to the details of an existing database within your Redis Enterprise Cloud account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the database, in the format `<subscription_id>/<db_id>`",
				Computed:    true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "ID of the subscription that the database belongs to",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a number"),
				},
			},
			"db_id": schema.Int64Attribute{
				Description: "The id of the database to filter returned databases",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the database to filter returned databases",
				Optional:    true,
				Computed:    true,
			},
			"protocol": schema.StringAttribute{
				Description: "The protocol of the database to filter returned databases",
				Optional:    true,
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region of the database to filter returned databases",
				Optional:    true,
				Computed:    true,
			},
			"memory_limit_in_gb": schema.Float64Attribute{
				Description: "The maximum memory usage for the database",
				Computed:    true,
			},
			"dataset_size_in_gb": schema.Float64Attribute{
				Description: "Maximum amount of data in the dataset for this specific database in GB",
				Computed:    true,
			},
			"redis_version": schema.StringAttribute{
				Description: "The redis version of the database",
				Computed:    true,
			},
			"query_performance_factor": schema.StringAttribute{
				Description: "Query performance factor for this specific database",
				Computed:    true,
			},
			"support_oss_cluster_api": schema.BoolAttribute{
				Description: "Supports the Redis open-source (OSS) Cluster API",
				Computed:    true,
			},
			"resp_version": schema.StringAttribute{
				Description: "The database's RESP version",
				Computed:    true,
			},
			"data_persistence": schema.StringAttribute{
				Description: "The rate of database data persistence (in persistent storage)",
				Computed:    true,
			},
			"data_eviction": schema.StringAttribute{
				Description: "The data items eviction method",
				Computed:    true,
			},
			"replication": schema.BoolAttribute{
				Description: "Database replication",
				Computed:    true,
			},
			"throughput_measurement_by": schema.StringAttribute{
				Description: "The throughput measurement method",
				Computed:    true,
			},
			"throughput_measurement_value": schema.Int64Attribute{
				Description: "The throughput value",
				Computed:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password used to access the database - not present on `memcached` protocol databases",
				Computed:    true,
				Sensitive:   true,
			},
			"enable_passwordless": schema.BoolAttribute{
				Description: "Whether the database is configured without a password",
				Computed:    true,
			},
			"public_endpoint": schema.StringAttribute{
				Description: "Public endpoint to access the database",
				Computed:    true,
			},
			"private_endpoint": schema.StringAttribute{
				Description: "Private endpoint to access the database",
				Computed:    true,
			},
			"replica_of": schema.SetAttribute{
				Description: "The set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of",
				Computed:    true,
				ElementType: types.StringType,
			},
			"alert": schema.SetAttribute{
				Description: "Set of alerts to enable on the database",
				Computed:    true,
				ElementType: DatabaseAlertElemType,
			},
			"module": schema.SetAttribute{
				Description: "Redis modules that have been enabled on the database",
				Computed:    true,
				ElementType: DatabaseModuleElemType,
			},
			"hashing_policy": schema.ListAttribute{
				Description: "The list of regular expression rules the database is sharded by",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enable_default_user": schema.BoolAttribute{
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Computed:    true,
			},
			"auto_minor_version_upgrade": schema.BoolAttribute{
				Description: "Automatically upgrades the database to newer minor versions within the same major release. Applies to version 8.4 and above.",
				Computed:    true,
			},
			"latest_backup_status": schema.SetAttribute{
				Description: "Details about the last backup that took place for this database",
				Computed:    true,
				ElementType: LatestBackupStatusElemType,
			},
			"latest_import_status": schema.SetAttribute{
				Description: "Details about the last import that took place for this database",
				Computed:    true,
				ElementType: LatestImportStatusElemType,
			},
			"tags": schema.MapAttribute{
				Description: "Tags for database management",
				Computed:    true,
				ElementType: types.StringType,
			},
			"source_ips": schema.SetAttribute{
				Description: "Set of CIDR addresses to allow access to the database",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}
