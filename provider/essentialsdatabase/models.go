package essentialsdatabase

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// SyncSourceModel describes one replica sync source.
type SyncSourceModel struct {
	Endpoint   types.String `tfsdk:"endpoint"`
	Encryption types.Bool   `tfsdk:"encryption"`
	ServerCert types.String `tfsdk:"server_cert"`
}

var syncSourceElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(SyncSourceModel{})}

// ReplicaModel describes the database replication settings.
type ReplicaModel struct {
	SyncSource types.Set `tfsdk:"sync_source"`
}

var replicaElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ReplicaModel{
	SyncSource: types.SetNull(syncSourceElemType),
})}

// AlertModel describes one database alert.
type AlertModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.Int64  `tfsdk:"value"`
}

var alertElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AlertModel{})}

// ModuleModel describes one database module.
type ModuleModel struct {
	Name types.String `tfsdk:"name"`
}

var moduleElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ModuleModel{})}

// StatusErrorModel describes an error returned while looking up a backup or import status.
type StatusErrorModel struct {
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
}

var statusErrorElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(StatusErrorModel{})}

// BackupStatusResponseModel describes the last backup operation.
type BackupStatusResponseModel struct {
	Status         types.String `tfsdk:"status"`
	LastBackupTime types.String `tfsdk:"last_backup_time"`
	FailureReason  types.String `tfsdk:"failure_reason"`
}

var backupStatusResponseElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(BackupStatusResponseModel{})}

// LatestBackupStatusModel describes the latest backup status lookup.
type LatestBackupStatusModel struct {
	Response types.Set `tfsdk:"response"`
	Error    types.Set `tfsdk:"error"`
}

var latestBackupStatusElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(LatestBackupStatusModel{
	Response: types.SetNull(backupStatusResponseElemType),
	Error:    types.SetNull(statusErrorElemType),
})}

// FailureReasonParamModel describes one parameter of an import failure.
type FailureReasonParamModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

var failureReasonParamElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(FailureReasonParamModel{})}

// ImportStatusResponseModel describes the last import operation.
type ImportStatusResponseModel struct {
	Status              types.String `tfsdk:"status"`
	LastImportTime      types.String `tfsdk:"last_import_time"`
	FailureReason       types.String `tfsdk:"failure_reason"`
	FailureReasonParams types.List   `tfsdk:"failure_reason_params"`
}

var importStatusResponseElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ImportStatusResponseModel{
	FailureReasonParams: types.ListNull(failureReasonParamElemType),
})}

// LatestImportStatusModel describes the latest import status lookup.
type LatestImportStatusModel struct {
	Response types.Set `tfsdk:"response"`
	Error    types.Set `tfsdk:"error"`
}

var latestImportStatusElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(LatestImportStatusModel{
	Response: types.SetNull(importStatusResponseElemType),
	Error:    types.SetNull(statusErrorElemType),
})}
