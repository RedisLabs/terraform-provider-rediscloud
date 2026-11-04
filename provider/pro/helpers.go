package pro

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_backups"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_imports"
	"github.com/RedisLabs/rediscloud-go-api/service/tags"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DatabaseAlertsFromAPI maps database alerts to a known set, empty when the API returns none.
func DatabaseAlertsFromAPI(ctx context.Context, alerts []*databases.Alert) (types.Set, diag.Diagnostics) {
	models := make([]DatabaseAlertModel, 0, len(alerts))
	for _, a := range alerts {
		models = append(models, DatabaseAlertModel{
			Name:  types.StringValue(redis.StringValue(a.Name)),
			Value: types.Int64Value(int64(redis.IntValue(a.Value))),
		})
	}
	return types.SetValueFrom(ctx, DatabaseAlertElemType, models)
}

// DatabaseModulesFromAPI maps database modules to a known set, empty when the API returns none.
func DatabaseModulesFromAPI(ctx context.Context, modules []*databases.Module) (types.Set, diag.Diagnostics) {
	models := make([]DatabaseModuleModel, 0, len(modules))
	for _, m := range modules {
		models = append(models, DatabaseModuleModel{
			Name: types.StringValue(redis.StringValue(m.Name)),
		})
	}
	return types.SetValueFrom(ctx, DatabaseModuleElemType, models)
}

// DatabaseTagsFromAPI maps database tags to a known map, empty when the database has no tags.
func DatabaseTagsFromAPI(ctx context.Context, apiTags *tags.AllTags) (types.Map, diag.Diagnostics) {
	result := make(map[string]string)
	if apiTags != nil && apiTags.Tags != nil {
		for _, t := range *apiTags.Tags {
			result[redis.StringValue(t.Key)] = redis.StringValue(t.Value)
		}
	}
	return types.MapValueFrom(ctx, types.StringType, result)
}

func statusErrorsFromAPI(ctx context.Context, typ, description, status *string) (types.Set, diag.Diagnostics) {
	return types.SetValueFrom(ctx, statusErrorElemType, []StatusErrorModel{{
		Type:        types.StringValue(redis.StringValue(typ)),
		Description: types.StringValue(redis.StringValue(description)),
		Status:      types.StringValue(redis.StringValue(status)),
	}})
}

// LatestBackupStatusFromAPI maps the latest backup status to a single-element set.
// A nil status, as when the lookup failed, yields an empty set.
func LatestBackupStatusFromAPI(ctx context.Context, status *latest_backups.LatestBackupStatus) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	models := make([]LatestBackupStatusModel, 0, 1)
	if status == nil {
		return types.SetValueFrom(ctx, LatestBackupStatusElemType, models)
	}

	responses := make([]LatestBackupResponseModel, 0, 1)
	var apiError *latest_backups.Error
	if status.Response != nil {
		if r := status.Response.Resource; r != nil {
			lastBackupTime := types.StringNull()
			if r.LastBackupTime != nil {
				lastBackupTime = types.StringValue(r.LastBackupTime.String())
			}
			responses = append(responses, LatestBackupResponseModel{
				Status:         types.StringValue(redis.StringValue(r.Status)),
				LastBackupTime: lastBackupTime,
				FailureReason:  types.StringValue(redis.StringValue(r.FailureReason)),
			})
		}
		apiError = status.Response.Error
	}

	response, d := types.SetValueFrom(ctx, latestBackupResponseElemType, responses)
	diags.Append(d...)
	errs := types.SetValueMust(statusErrorElemType, []attr.Value{})
	if apiError != nil {
		errs, d = statusErrorsFromAPI(ctx, apiError.Type, apiError.Description, apiError.Status)
		diags.Append(d...)
	}
	if diags.HasError() {
		return types.SetNull(LatestBackupStatusElemType), diags
	}

	models = append(models, LatestBackupStatusModel{Response: response, Error: errs})
	result, d := types.SetValueFrom(ctx, LatestBackupStatusElemType, models)
	diags.Append(d...)
	return result, diags
}

// LatestImportStatusFromAPI maps the latest import status to a single-element set.
// A nil status, as when the lookup failed, yields an empty set.
func LatestImportStatusFromAPI(ctx context.Context, status *latest_imports.LatestImportStatus) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	models := make([]LatestImportStatusModel, 0, 1)
	if status == nil {
		return types.SetValueFrom(ctx, LatestImportStatusElemType, models)
	}

	responses := make([]LatestImportResponseModel, 0, 1)
	var apiError *latest_imports.Error
	if status.Response != nil {
		if r := status.Response.Resource; r != nil {
			lastImportTime := types.StringNull()
			if r.LastImportTime != nil {
				lastImportTime = types.StringValue(r.LastImportTime.String())
			}
			params := make([]FailureReasonParamModel, 0, len(r.FailureReasonParams))
			for _, p := range r.FailureReasonParams {
				params = append(params, FailureReasonParamModel{
					Key:   types.StringValue(redis.StringValue(p.Key)),
					Value: types.StringValue(redis.StringValue(p.Value)),
				})
			}
			failureReasonParams, d := types.ListValueFrom(ctx, failureReasonParamElemType, params)
			diags.Append(d...)
			responses = append(responses, LatestImportResponseModel{
				Status:              types.StringValue(redis.StringValue(r.Status)),
				LastImportTime:      lastImportTime,
				FailureReason:       types.StringValue(redis.StringValue(r.FailureReason)),
				FailureReasonParams: failureReasonParams,
			})
		}
		apiError = status.Response.Error
	}

	response, d := types.SetValueFrom(ctx, latestImportResponseElemType, responses)
	diags.Append(d...)
	errs := types.SetValueMust(statusErrorElemType, []attr.Value{})
	if apiError != nil {
		errs, d = statusErrorsFromAPI(ctx, apiError.Type, apiError.Description, apiError.Status)
		diags.Append(d...)
	}
	if diags.HasError() {
		return types.SetNull(LatestImportStatusElemType), diags
	}

	models = append(models, LatestImportStatusModel{Response: response, Error: errs})
	result, d := types.SetValueFrom(ctx, LatestImportStatusElemType, models)
	diags.Append(d...)
	return result, diags
}
