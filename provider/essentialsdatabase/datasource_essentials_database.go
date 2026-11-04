package essentialsdatabase

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &essentialsDatabaseDataSource{}
	_ datasource.DataSourceWithConfigure = &essentialsDatabaseDataSource{}
)

type essentialsDatabaseDataSource struct {
	client *client.ApiClient
}

// NewEssentialsDatabaseDataSource creates a new data source instance.
func NewEssentialsDatabaseDataSource() datasource.DataSource {
	return &essentialsDatabaseDataSource{}
}

// Metadata returns the data source type name.
func (d *essentialsDatabaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_essentials_database"
}

// Configure adds the provider configured client to the data source.
func (d *essentialsDatabaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *essentialsDatabaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Essentials Database data source allows access to the details of an existing database within your Redis Enterprise Cloud account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the database in the format `subscription_id/db_id`",
				Computed:    true,
			},
			"subscription_id": schema.Int64Attribute{
				Description: "Identifier of the essentials subscription",
				Required:    true,
			},
			"db_id": schema.Int64Attribute{
				Description: "Identifier of the database created",
				Computed:    true,
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the database",
				Computed:    true,
				Optional:    true,
			},
			"protocol": schema.StringAttribute{
				Description: "The protocol that will be used to access the database, (either ‘redis’, 'memcached’ or 'stack')",
				Computed:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: "The Cloud Provider hosting this database",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region within the Cloud Provider where this database is hosted",
				Computed:    true,
			},
			"redis_version": schema.StringAttribute{
				Description: "The actual Redis database version",
				Computed:    true,
			},
			"redis_version_compliance": schema.StringAttribute{
				Description: "The compliance level (redis version) of this database",
				Computed:    true,
			},
			"resp_version": schema.StringAttribute{
				Description: "RESP version must be compatible with Redis version.",
				Computed:    true,
			},
			"data_persistence": schema.StringAttribute{
				Description: "Rate of database data persistence (in persistent storage).",
				Computed:    true,
			},
			"data_eviction": schema.StringAttribute{
				Description: "The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')",
				Computed:    true,
			},
			"replication": schema.BoolAttribute{
				Description: "Database's replication",
				Computed:    true,
			},
			"activated_on": schema.StringAttribute{
				Description: "When this database was activated",
				Computed:    true,
			},
			"periodic_backup_path": schema.StringAttribute{
				Description: "Path that will be used to store database backup files",
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
			"source_ips": schema.SetAttribute{
				Description: "Set of CIDR addresses to allow access to the database",
				Computed:    true,
				ElementType: types.StringType,
			},
			// Object collections are computed attributes, not blocks, to keep the SDKv2 protocol-v5 shape:
			// SDKv2 serves computed-only nested resources as attributes.
			"replica": schema.SetAttribute{
				Description: "Details of database replication",
				Computed:    true,
				ElementType: replicaElemType,
			},
			"client_tls_certificates": schema.ListAttribute{
				Description: "",
				Computed:    true,
				ElementType: types.StringType,
			},
			"password": schema.StringAttribute{
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Computed:    true,
				Sensitive:   true,
			},
			"enable_default_user": schema.BoolAttribute{
				Description: "When 'true', enables connecting to the database with the 'default' user. Default: 'true'",
				Computed:    true,
			},
			"alert": schema.SetAttribute{
				Description: "Set of alerts to enable on the database",
				Computed:    true,
				ElementType: alertElemType,
			},
			"modules": schema.SetAttribute{
				Description: "Modules to be provisioned in the database",
				Computed:    true,
				ElementType: moduleElemType,
			},
			"latest_backup_status": schema.SetAttribute{
				Description: "Details about the last backup that took place for this database",
				Computed:    true,
				ElementType: latestBackupStatusElemType,
			},
			"latest_import_status": schema.SetAttribute{
				Description: "Details about the last import that took place for this active-active database",
				Computed:    true,
				ElementType: latestImportStatusElemType,
			},
			"memory_limit_in_gb": schema.Float64Attribute{
				Description: "Maximum memory usage for this specific database",
				Computed:    true,
			},
			"support_oss_cluster_api": schema.BoolAttribute{
				Description: "Support Redis open-source (OSS) Cluster API",
				Computed:    true,
			},
			"external_endpoint_for_oss_cluster_api": schema.BoolAttribute{
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Computed:    true,
			},
			"enable_database_clustering": schema.BoolAttribute{
				Description: "Distributes database data to different cloud instances",
				Computed:    true,
			},
			"regex_rules": schema.ListAttribute{
				Description: "Shard regex rules. Relevant only for a sharded database",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enable_tls": schema.BoolAttribute{
				Description: "Use TLS for authentication",
				Computed:    true,
			},
			"tags": schema.MapAttribute{
				Description: "Tags for database management",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}
