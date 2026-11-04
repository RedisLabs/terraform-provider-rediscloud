package essentialsdatabase

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EssentialsDatabaseResourceModel describes the resource data model.
type EssentialsDatabaseResourceModel struct {
	ID                               types.String   `tfsdk:"id"`
	SubscriptionID                   types.Int64    `tfsdk:"subscription_id"`
	DbID                             types.Int64    `tfsdk:"db_id"`
	Name                             types.String   `tfsdk:"name"`
	Protocol                         types.String   `tfsdk:"protocol"`
	RedisVersion                     types.String   `tfsdk:"redis_version"`
	RedisVersionActual               types.String   `tfsdk:"redis_version_actual"`
	CloudProvider                    types.String   `tfsdk:"cloud_provider"`
	Region                           types.String   `tfsdk:"region"`
	RedisVersionCompliance           types.String   `tfsdk:"redis_version_compliance"`
	RespVersion                      types.String   `tfsdk:"resp_version"`
	DataPersistence                  types.String   `tfsdk:"data_persistence"`
	DataEviction                     types.String   `tfsdk:"data_eviction"`
	Replication                      types.Bool     `tfsdk:"replication"`
	ActivatedOn                      types.String   `tfsdk:"activated_on"`
	PeriodicBackupPath               types.String   `tfsdk:"periodic_backup_path"`
	PublicEndpoint                   types.String   `tfsdk:"public_endpoint"`
	PrivateEndpoint                  types.String   `tfsdk:"private_endpoint"`
	SourceIPs                        types.List     `tfsdk:"source_ips"`
	Replica                          types.List     `tfsdk:"replica"`
	ClientTLSCertificates            types.List     `tfsdk:"client_tls_certificates"`
	Password                         types.String   `tfsdk:"password"`
	EnableDefaultUser                types.Bool     `tfsdk:"enable_default_user"`
	Alert                            types.Set      `tfsdk:"alert"`
	Modules                          types.Set      `tfsdk:"modules"`
	EnablePaygFeatures               types.Bool     `tfsdk:"enable_payg_features"`
	MemoryLimitInGB                  types.Float64  `tfsdk:"memory_limit_in_gb"`
	SupportOSSClusterAPI             types.Bool     `tfsdk:"support_oss_cluster_api"`
	ExternalEndpointForOSSClusterAPI types.Bool     `tfsdk:"external_endpoint_for_oss_cluster_api"`
	EnableDatabaseClustering         types.Bool     `tfsdk:"enable_database_clustering"`
	RegexRules                       types.List     `tfsdk:"regex_rules"`
	EnableTLS                        types.Bool     `tfsdk:"enable_tls"`
	Tags                             types.Map      `tfsdk:"tags"`
	Timeouts                         timeouts.Value `tfsdk:"timeouts"`
}
