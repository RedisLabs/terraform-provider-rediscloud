package pro

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// ProDatabaseDataSourceModel describes the rediscloud_database data source data model.
type ProDatabaseDataSourceModel struct {
	ID                         types.String  `tfsdk:"id"`
	SubscriptionID             types.String  `tfsdk:"subscription_id"`
	DbID                       types.Int64   `tfsdk:"db_id"`
	Name                       types.String  `tfsdk:"name"`
	Protocol                   types.String  `tfsdk:"protocol"`
	Region                     types.String  `tfsdk:"region"`
	MemoryLimitInGB            types.Float64 `tfsdk:"memory_limit_in_gb"`
	DatasetSizeInGB            types.Float64 `tfsdk:"dataset_size_in_gb"`
	RedisVersion               types.String  `tfsdk:"redis_version"`
	QueryPerformanceFactor     types.String  `tfsdk:"query_performance_factor"`
	SupportOSSClusterAPI       types.Bool    `tfsdk:"support_oss_cluster_api"`
	RespVersion                types.String  `tfsdk:"resp_version"`
	DataPersistence            types.String  `tfsdk:"data_persistence"`
	DataEviction               types.String  `tfsdk:"data_eviction"`
	Replication                types.Bool    `tfsdk:"replication"`
	ThroughputMeasurementBy    types.String  `tfsdk:"throughput_measurement_by"`
	ThroughputMeasurementValue types.Int64   `tfsdk:"throughput_measurement_value"`
	Password                   types.String  `tfsdk:"password"`
	EnablePasswordless         types.Bool    `tfsdk:"enable_passwordless"`
	PublicEndpoint             types.String  `tfsdk:"public_endpoint"`
	PrivateEndpoint            types.String  `tfsdk:"private_endpoint"`
	ReplicaOf                  types.Set     `tfsdk:"replica_of"`
	Alert                      types.Set     `tfsdk:"alert"`
	Module                     types.Set     `tfsdk:"module"`
	HashingPolicy              types.List    `tfsdk:"hashing_policy"`
	EnableDefaultUser          types.Bool    `tfsdk:"enable_default_user"`
	AutoMinorVersionUpgrade    types.Bool    `tfsdk:"auto_minor_version_upgrade"`
	LatestBackupStatus         types.Set     `tfsdk:"latest_backup_status"`
	LatestImportStatus         types.Set     `tfsdk:"latest_import_status"`
	Tags                       types.Map     `tfsdk:"tags"`
	SourceIPs                  types.Set     `tfsdk:"source_ips"`
}

// DatabaseAlertModel describes an element of the computed alert set.
type DatabaseAlertModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.Int64  `tfsdk:"value"`
}

// DatabaseModuleModel describes an element of the computed module set.
type DatabaseModuleModel struct {
	Name types.String `tfsdk:"name"`
}

// StatusErrorModel describes the error reported when a latest backup or import status lookup fails.
type StatusErrorModel struct {
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
}

// LatestBackupResponseModel describes the outcome of the latest backup.
type LatestBackupResponseModel struct {
	Status         types.String `tfsdk:"status"`
	LastBackupTime types.String `tfsdk:"last_backup_time"`
	FailureReason  types.String `tfsdk:"failure_reason"`
}

// LatestBackupStatusModel describes an element of the computed latest_backup_status set.
type LatestBackupStatusModel struct {
	Response types.Set `tfsdk:"response"`
	Error    types.Set `tfsdk:"error"`
}

// FailureReasonParamModel describes a parameter of an import failure reason.
type FailureReasonParamModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// LatestImportResponseModel describes the outcome of the latest import.
type LatestImportResponseModel struct {
	Status              types.String `tfsdk:"status"`
	LastImportTime      types.String `tfsdk:"last_import_time"`
	FailureReason       types.String `tfsdk:"failure_reason"`
	FailureReasonParams types.List   `tfsdk:"failure_reason_params"`
}

// LatestImportStatusModel describes an element of the computed latest_import_status set.
type LatestImportStatusModel struct {
	Response types.Set `tfsdk:"response"`
	Error    types.Set `tfsdk:"error"`
}

var (
	DatabaseAlertElemType  = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(DatabaseAlertModel{})}
	DatabaseModuleElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(DatabaseModuleModel{})}

	statusErrorElemType          = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(StatusErrorModel{})}
	latestBackupResponseElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(LatestBackupResponseModel{})}
	failureReasonParamElemType   = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(FailureReasonParamModel{})}
	latestImportResponseElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(LatestImportResponseModel{
		FailureReasonParams: types.ListNull(failureReasonParamElemType),
	})}

	LatestBackupStatusElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(LatestBackupStatusModel{
		Response: types.SetNull(latestBackupResponseElemType),
		Error:    types.SetNull(statusErrorElemType),
	})}
	LatestImportStatusElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(LatestImportStatusModel{
		Response: types.SetNull(latestImportResponseElemType),
		Error:    types.SetNull(statusErrorElemType),
	})}
)
