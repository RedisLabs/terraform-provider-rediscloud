package provider

import (
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_backups"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_imports"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"sync"
	"time"
)

// This timeout is an absolute maximum used in some of the waitForStatus operations concerning creation and updating
// Subscriptions and Databases. Reads and Deletions have their own, stricter timeouts because they consistently behave
// well. The Terraform operation-level timeout should kick in way before we hit this and kill the task.
// Unfortunately there's no "time-remaining-before-timeout" utility, or we could use that in the wait blocks.
const safetyTimeout = 6 * time.Hour

func setToStringSlice(set *schema.Set) []*string {
	var ret []*string
	for _, s := range set.List() {
		ret = append(ret, redis.String(s.(string)))
	}
	return ret
}

func interfaceToStringSlice(list []interface{}) []*string {
	var ret []*string
	for _, i := range list {
		ret = append(ret, redis.String(i.(string)))
	}
	return ret
}

type perIdLock struct {
	lock  sync.Mutex
	store map[int]*sync.Mutex
}

func newPerIdLock() *perIdLock {
	return &perIdLock{
		store: map[int]*sync.Mutex{},
	}
}

func (m *perIdLock) Lock(id int) {
	m.get(id).Lock()
}

func (m *perIdLock) Unlock(id int) {
	m.get(id).Unlock()
}

func (m *perIdLock) get(id int) *sync.Mutex {
	m.lock.Lock()
	defer m.lock.Unlock()

	if v, ok := m.store[id]; ok {
		return v
	}

	mutex := &sync.Mutex{}
	m.store[id] = mutex
	return mutex
}

// IDs of any resources dependent on a subscription need to be divided by a slash. In this format: <sub id>/<resource id>.
func buildResourceId(subId int, id int) string {
	return fmt.Sprintf("%d/%d", subId, id)
}

func isTime() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics

		v, ok := i.(string)
		if !ok {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Value not a string",
				Detail:   fmt.Sprintf("Value should be a string rather than %T", i),
			})
		} else if _, err := time.Parse("15:04", v); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Value is not a time",
				Detail:   fmt.Sprintf("Value should be a valid time, got: %q: %s", i, err),
			})
		}

		return diags
	}
}

func parseLatestBackupStatus(latestBackupStatus *latest_backups.LatestBackupStatus) ([]map[string]interface{}, error) {
	lbs := map[string]interface{}{
		"response": nil,
		"error":    nil,
	}

	if latestBackupStatus.Response.Resource != nil {
		res := map[string]interface{}{
			"status":           redis.StringValue(latestBackupStatus.Response.Resource.Status),
			"last_backup_time": nil,
			"failure_reason":   redis.StringValue(latestBackupStatus.Response.Resource.FailureReason),
		}
		if latestBackupStatus.Response.Resource.LastBackupTime != nil {
			res["last_backup_time"] = latestBackupStatus.Response.Resource.LastBackupTime.String()
		}
		lbs["response"] = []map[string]interface{}{res}
	}

	if latestBackupStatus.Response.Error != nil {
		err := map[string]interface{}{
			"type":        redis.StringValue(latestBackupStatus.Response.Error.Type),
			"description": redis.StringValue(latestBackupStatus.Response.Error.Description),
			"status":      redis.StringValue(latestBackupStatus.Response.Error.Status),
		}
		lbs["error"] = []map[string]interface{}{err}
	}

	return []map[string]interface{}{lbs}, nil
}

func parseLatestImportStatus(latestImportStatus *latest_imports.LatestImportStatus) ([]map[string]interface{}, error) {
	lis := map[string]interface{}{
		"response": nil,
		"error":    nil,
	}

	if latestImportStatus.Response.Resource != nil {
		res := map[string]interface{}{
			"status":                redis.StringValue(latestImportStatus.Response.Resource.Status),
			"last_import_time":      nil,
			"failure_reason":        redis.StringValue(latestImportStatus.Response.Resource.FailureReason),
			"failure_reason_params": parseFailureReasonParams(latestImportStatus.Response.Resource.FailureReasonParams),
		}
		if latestImportStatus.Response.Resource.LastImportTime != nil {
			res["last_import_time"] = latestImportStatus.Response.Resource.LastImportTime.String()
		}
		lis["response"] = []map[string]interface{}{res}
	}

	if latestImportStatus.Response.Error != nil {
		err := map[string]interface{}{
			"type":        redis.StringValue(latestImportStatus.Response.Error.Type),
			"description": redis.StringValue(latestImportStatus.Response.Error.Description),
			"status":      redis.StringValue(latestImportStatus.Response.Error.Status),
		}
		lis["error"] = []map[string]interface{}{err}
	}

	return []map[string]interface{}{lis}, nil
}

func parseFailureReasonParams(params []*latest_imports.FailureReasonParam) []map[string]interface{} {
	writableParams := make([]map[string]interface{}, 0)
	for _, param := range params {
		writableParams = append(writableParams, map[string]interface{}{
			"key":   redis.StringValue(param.Key),
			"value": redis.StringValue(param.Value),
		})
	}
	return writableParams
}
