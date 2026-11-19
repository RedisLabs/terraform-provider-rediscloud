package essentialsdatabase

import (
	"context"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_backups"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_imports"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
)

// defaultSourceIP is what the API reports when a database has no source IP restriction.
const defaultSourceIP = "0.0.0.0/0"

type databaseFilter func(db *fixedDatabases.FixedDatabase) bool

func filterFixedDatabases(list *fixedDatabases.ListFixedDatabase, filters []databaseFilter) ([]*fixedDatabases.FixedDatabase, error) {
	var filtered []*fixedDatabases.FixedDatabase
	for list.Next() {
		if matchesAll(list.Value(), filters) {
			filtered = append(filtered, list.Value())
		}
	}
	if list.Err() != nil {
		return nil, list.Err()
	}
	return filtered, nil
}

func matchesAll(db *fixedDatabases.FixedDatabase, filters []databaseFilter) bool {
	for _, filter := range filters {
		if !filter(db) {
			return false
		}
	}
	return true
}

// sourceIPsFromAPI returns no entries when the API reports only the implicit allow-all rule,
// because the API stores an empty request as that rule.
func sourceIPsFromAPI(security *fixedDatabases.Security) []string {
	if security == nil {
		return []string{}
	}
	if len(security.SourceIPs) == 1 && redis.StringValue(security.SourceIPs[0]) == defaultSourceIP {
		return []string{}
	}
	return redis.StringSliceValue(security.SourceIPs...)
}

func syncSourceModelsFromAPI(replica *fixedDatabases.ReplicaOf) []SyncSourceModel {
	syncSources := make([]SyncSourceModel, 0, len(replica.SyncSources))
	for _, ss := range replica.SyncSources {
		syncSources = append(syncSources, SyncSourceModel{
			Endpoint:   types.StringValue(redis.StringValue(ss.Endpoint)),
			Encryption: types.BoolValue(redis.BoolValue(ss.Encryption)),
			ServerCert: types.StringValue(redis.StringValue(ss.ServerCert)),
		})
	}
	return syncSources
}

func replicaModelsFromAPI(ctx context.Context, replica *fixedDatabases.ReplicaOf) ([]ReplicaModel, diag.Diagnostics) {
	if replica == nil {
		return []ReplicaModel{}, nil
	}
	syncSourceSet, diags := types.SetValueFrom(ctx, syncSourceElemType, syncSourceModelsFromAPI(replica))
	if diags.HasError() {
		return nil, diags
	}
	return []ReplicaModel{{SyncSource: syncSourceSet}}, diags
}

// AlertsFromAPI converts database alerts into a Terraform set.
func AlertsFromAPI(ctx context.Context, alerts *[]*databases.Alert) (types.Set, diag.Diagnostics) {
	models := make([]AlertModel, 0)
	if alerts != nil {
		for _, alert := range *alerts {
			models = append(models, AlertModel{
				Name:  types.StringValue(redis.StringValue(alert.Name)),
				Value: types.Int64Value(int64(redis.IntValue(alert.Value))),
			})
		}
	}
	return types.SetValueFrom(ctx, alertElemType, models)
}

// ModulesFromAPI converts database modules into a Terraform set.
func ModulesFromAPI(ctx context.Context, modules *[]*databases.Module) (types.Set, diag.Diagnostics) {
	models := make([]ModuleModel, 0)
	if modules != nil {
		for _, module := range *modules {
			models = append(models, ModuleModel{Name: types.StringValue(redis.StringValue(module.Name))})
		}
	}
	return types.SetValueFrom(ctx, moduleElemType, models)
}

func regexRulesFromAPI(ctx context.Context, clustering *fixedDatabases.Clustering) (types.List, diag.Diagnostics) {
	rules := []string{}
	if clustering != nil {
		rules = pro.FlattenRegexRules(clustering.RegexRules)
	}
	return types.ListValueFrom(ctx, types.StringType, rules)
}

func tagsFromAPI(ctx context.Context, api *client.ApiClient, subId, dbId int) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	tagResponse, err := api.Client.Tags.GetFixed(ctx, subId, dbId)
	if err != nil {
		diags.AddError("Unable to Read Essentials Database Tags", err.Error())
		return types.MapNull(types.StringType), diags
	}
	tags := make(map[string]string)
	if tagResponse.Tags != nil {
		for _, tag := range *tagResponse.Tags {
			tags[redis.StringValue(tag.Key)] = redis.StringValue(tag.Value)
		}
	}
	return types.MapValueFrom(ctx, types.StringType, tags)
}

func timeStringValue(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.String())
}

func latestBackupStatusFromAPI(ctx context.Context, status *latest_backups.LatestBackupStatus) (types.Set, diag.Diagnostics) {
	models := make([]LatestBackupStatusModel, 0, 1)
	if status != nil {
		responses := make([]BackupStatusResponseModel, 0, 1)
		errs := make([]StatusErrorModel, 0, 1)
		if status.Response != nil {
			if res := status.Response.Resource; res != nil {
				responses = append(responses, BackupStatusResponseModel{
					Status:         types.StringValue(redis.StringValue(res.Status)),
					LastBackupTime: timeStringValue(res.LastBackupTime),
					FailureReason:  types.StringValue(redis.StringValue(res.FailureReason)),
				})
			}
			if e := status.Response.Error; e != nil {
				errs = append(errs, StatusErrorModel{
					Type:        types.StringValue(redis.StringValue(e.Type)),
					Description: types.StringValue(redis.StringValue(e.Description)),
					Status:      types.StringValue(redis.StringValue(e.Status)),
				})
			}
		}
		responseSet, diags := types.SetValueFrom(ctx, backupStatusResponseElemType, responses)
		if diags.HasError() {
			return types.SetNull(latestBackupStatusElemType), diags
		}
		errorSet, diags := types.SetValueFrom(ctx, statusErrorElemType, errs)
		if diags.HasError() {
			return types.SetNull(latestBackupStatusElemType), diags
		}
		models = append(models, LatestBackupStatusModel{Response: responseSet, Error: errorSet})
	}
	return types.SetValueFrom(ctx, latestBackupStatusElemType, models)
}

func latestImportStatusFromAPI(ctx context.Context, status *latest_imports.LatestImportStatus) (types.Set, diag.Diagnostics) {
	models := make([]LatestImportStatusModel, 0, 1)
	if status != nil {
		responses := make([]ImportStatusResponseModel, 0, 1)
		errs := make([]StatusErrorModel, 0, 1)
		if status.Response != nil {
			if res := status.Response.Resource; res != nil {
				params := make([]FailureReasonParamModel, 0, len(res.FailureReasonParams))
				for _, p := range res.FailureReasonParams {
					params = append(params, FailureReasonParamModel{
						Key:   types.StringValue(redis.StringValue(p.Key)),
						Value: types.StringValue(redis.StringValue(p.Value)),
					})
				}
				paramList, diags := types.ListValueFrom(ctx, failureReasonParamElemType, params)
				if diags.HasError() {
					return types.SetNull(latestImportStatusElemType), diags
				}
				responses = append(responses, ImportStatusResponseModel{
					Status:              types.StringValue(redis.StringValue(res.Status)),
					LastImportTime:      timeStringValue(res.LastImportTime),
					FailureReason:       types.StringValue(redis.StringValue(res.FailureReason)),
					FailureReasonParams: paramList,
				})
			}
			if e := status.Response.Error; e != nil {
				errs = append(errs, StatusErrorModel{
					Type:        types.StringValue(redis.StringValue(e.Type)),
					Description: types.StringValue(redis.StringValue(e.Description)),
					Status:      types.StringValue(redis.StringValue(e.Status)),
				})
			}
		}
		responseSet, diags := types.SetValueFrom(ctx, importStatusResponseElemType, responses)
		if diags.HasError() {
			return types.SetNull(latestImportStatusElemType), diags
		}
		errorSet, diags := types.SetValueFrom(ctx, statusErrorElemType, errs)
		if diags.HasError() {
			return types.SetNull(latestImportStatusElemType), diags
		}
		models = append(models, LatestImportStatusModel{Response: responseSet, Error: errorSet})
	}
	return types.SetValueFrom(ctx, latestImportStatusElemType, models)
}
