package activeactive

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ActiveActiveDatabaseModel describes the resource data model for the active-active database.
type ActiveActiveDatabaseModel struct {
	ID                               types.String  `tfsdk:"id"`
	SubscriptionID                   types.Int64   `tfsdk:"subscription_id"`
	DbID                             types.Int64   `tfsdk:"db_id"`
	Name                             types.String  `tfsdk:"name"`
	MemoryLimitInGB                  types.Float64 `tfsdk:"memory_limit_in_gb"`
	DatasetSizeInGB                  types.Float64 `tfsdk:"dataset_size_in_gb"`
	RedisVersion                     types.String  `tfsdk:"redis_version"`
	SupportOssClusterAPI             types.Bool    `tfsdk:"support_oss_cluster_api"`
	ExternalEndpointForOssClusterAPI types.Bool    `tfsdk:"external_endpoint_for_oss_cluster_api"`
	EnableTLS                        types.Bool    `tfsdk:"enable_tls"`
	ClientSSLCertificate             types.String  `tfsdk:"client_ssl_certificate"`
	ClientTLSCertificates            types.List    `tfsdk:"client_tls_certificates"`
	DataEviction                     types.String  `tfsdk:"data_eviction"`
	GlobalDataPersistence            types.String  `tfsdk:"global_data_persistence"`
	GlobalPassword                   types.String  `tfsdk:"global_password"`
	GlobalAlert                      types.Set     `tfsdk:"global_alert"`
	GlobalModules                    types.List    `tfsdk:"global_modules"`
	GlobalSourceIPs                  types.Set     `tfsdk:"global_source_ips"`
	GlobalEnableDefaultUser          types.Bool    `tfsdk:"global_enable_default_user"`
	AutoMinorVersionUpgrade          types.Bool    `tfsdk:"auto_minor_version_upgrade"`
	GlobalRespVersion                types.String  `tfsdk:"global_resp_version"`
	OverrideRegion                   types.Set     `tfsdk:"override_region"`
	PublicEndpoint                   types.Map     `tfsdk:"public_endpoint"`
	PrivateEndpoint                  types.Map     `tfsdk:"private_endpoint"`
	Port                             types.Int64   `tfsdk:"port"`
	Tags                             types.Map     `tfsdk:"tags"`
}

// AlertModel describes the global_alert nested block.
type AlertModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.Int64  `tfsdk:"value"`
}

// OverrideRegionModel describes the override_region nested block.
type OverrideRegionModel struct {
	Name                          types.String `tfsdk:"name"`
	OverrideGlobalAlert           types.Set    `tfsdk:"override_global_alert"`
	OverrideGlobalPassword        types.String `tfsdk:"override_global_password"`
	OverrideGlobalSourceIPs       types.Set    `tfsdk:"override_global_source_ips"`
	OverrideGlobalDataPersistence types.String `tfsdk:"override_global_data_persistence"`
	EnableDefaultUser             types.Bool   `tfsdk:"enable_default_user"`
	RemoteBackup                  types.List   `tfsdk:"remote_backup"`
}

// RemoteBackupModel describes the remote_backup nested block within override_region.
type RemoteBackupModel struct {
	Interval    types.String `tfsdk:"interval"`
	TimeUTC     types.String `tfsdk:"time_utc"`
	StorageType types.String `tfsdk:"storage_type"`
	StoragePath types.String `tfsdk:"storage_path"`
}
