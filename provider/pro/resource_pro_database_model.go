package pro

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ProDatabaseModel is the framework state for rediscloud_subscription_database.
type ProDatabaseModel struct {
	ID                               types.String          `tfsdk:"id"`
	SubscriptionID                   types.Int64           `tfsdk:"subscription_id"`
	DbID                             types.Int64           `tfsdk:"db_id"`
	Name                             types.String          `tfsdk:"name"`
	Protocol                         types.String          `tfsdk:"protocol"`
	MemoryLimitInGB                  types.Float64         `tfsdk:"memory_limit_in_gb"`
	DatasetSizeInGB                  types.Float64         `tfsdk:"dataset_size_in_gb"`
	RedisVersion                     types.String          `tfsdk:"redis_version"`
	RedisVersionActual               types.String          `tfsdk:"redis_version_actual"`
	SupportOSSClusterAPI             types.Bool            `tfsdk:"support_oss_cluster_api"`
	RespVersion                      types.String          `tfsdk:"resp_version"`
	ExternalEndpointForOSSClusterAPI types.Bool            `tfsdk:"external_endpoint_for_oss_cluster_api"`
	DataPersistence                  types.String          `tfsdk:"data_persistence"`
	DataEviction                     types.String          `tfsdk:"data_eviction"`
	Replication                      types.Bool            `tfsdk:"replication"`
	ThroughputMeasurementBy          types.String          `tfsdk:"throughput_measurement_by"`
	ThroughputMeasurementValue       types.Int64           `tfsdk:"throughput_measurement_value"`
	AverageItemSizeInBytes           types.Int64           `tfsdk:"average_item_size_in_bytes"`
	RamPercentage                    types.Int64           `tfsdk:"ram_percentage"`
	Password                         types.String          `tfsdk:"password"`
	EnablePasswordless               types.Bool            `tfsdk:"enable_passwordless"`
	PublicEndpoint                   types.String          `tfsdk:"public_endpoint"`
	PrivateEndpoint                  types.String          `tfsdk:"private_endpoint"`
	ClientSSLCertificate             types.String          `tfsdk:"client_ssl_certificate"`
	ClientTLSCertificates            types.List            `tfsdk:"client_tls_certificates"`
	PeriodicBackupPath               types.String          `tfsdk:"periodic_backup_path"`
	ReplicaOf                        types.Set             `tfsdk:"replica_of"`
	Alert                            AlertSetValue         `tfsdk:"alert"`
	QueryPerformanceFactor           types.String          `tfsdk:"query_performance_factor"`
	Modules                          ModuleSetValue        `tfsdk:"modules"`
	SourceIPs                        types.Set             `tfsdk:"source_ips"`
	HashingPolicy                    types.List            `tfsdk:"hashing_policy"`
	EnableTLS                        types.Bool            `tfsdk:"enable_tls"`
	EnableDefaultUser                types.Bool            `tfsdk:"enable_default_user"`
	AutoMinorVersionUpgrade          types.Bool            `tfsdk:"auto_minor_version_upgrade"`
	Port                             types.Int64           `tfsdk:"port"`
	RemoteBackup                     RemoteBackupListValue `tfsdk:"remote_backup"`
	Tags                             types.Map             `tfsdk:"tags"`
	Timeouts                         timeouts.Value        `tfsdk:"timeouts"`
}

// AlertModel is one alert nested block.
type AlertModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.Int64  `tfsdk:"value"`
}

// ModuleModel is one modules nested block.
type ModuleModel struct {
	Name types.String `tfsdk:"name"`
}

// RemoteBackupModel is the remote_backup nested block.
type RemoteBackupModel struct {
	Interval    types.String `tfsdk:"interval"`
	TimeUTC     types.String `tfsdk:"time_utc"`
	StorageType types.String `tfsdk:"storage_type"`
	StoragePath types.String `tfsdk:"storage_path"`
}
