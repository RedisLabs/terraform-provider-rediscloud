package utils

import (
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	redisTags "github.com/RedisLabs/rediscloud-go-api/service/tags"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
	"strconv"
	"strings"
)

import (
	"fmt"
	"sync"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/latest_backups"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_imports"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

const RedisCloudUrlEnvVar = "REDISCLOUD_URL"

func SetToStringSlice(set *schema.Set) []*string {
	var ret []*string
	for _, s := range set.List() {
		ret = append(ret, redis.String(s.(string)))
	}
	return ret
}

func InterfaceToStringSlice(list []interface{}) []*string {
	var ret []*string
	for _, i := range list {
		if i == nil {
			// The user probably entered "" (string's zero-value) but gets read in as nil (interface{}'s zero-value)
			ret = append(ret, redis.String(""))
		} else {
			ret = append(ret, redis.String(i.(string)))
		}
	}
	return ret
}

type perIdLock struct {
	lock  sync.Mutex
	store map[int]*sync.Mutex
}

func NewPerIdLock() *perIdLock {
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
func BuildResourceId(subId int, id int) string {
	return fmt.Sprintf("%d/%d", subId, id)
}

func IsTime() schema.SchemaValidateDiagFunc {
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

func ParseLatestBackupStatus(latestBackupStatus *latest_backups.LatestBackupStatus) ([]map[string]interface{}, error) {
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

func ParseLatestImportStatus(latestImportStatus *latest_imports.LatestImportStatus) ([]map[string]interface{}, error) {
	lis := map[string]interface{}{
		"response": nil,
		"error":    nil,
	}

	if latestImportStatus.Response.Resource != nil {
		res := map[string]interface{}{
			"status":                redis.StringValue(latestImportStatus.Response.Resource.Status),
			"last_import_time":      nil,
			"failure_reason":        redis.StringValue(latestImportStatus.Response.Resource.FailureReason),
			"failure_reason_params": ParseFailureReasonParams(latestImportStatus.Response.Resource.FailureReasonParams),
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

func ParseFailureReasonParams(params []*latest_imports.FailureReasonParam) []map[string]interface{} {
	writableParams := make([]map[string]interface{}, 0)
	for _, param := range params {
		writableParams = append(writableParams, map[string]interface{}{
			"key":   redis.StringValue(param.Key),
			"value": redis.StringValue(param.Value),
		})
	}
	return writableParams
}

func ApplyCertificateHints(tlsAuthEnabled bool, d *schema.ResourceData) error {
	sslCertificate := d.Get("client_ssl_certificate").(string)
	tlsCertificates := InterfaceToStringSlice(d.Get("client_tls_certificates").([]interface{}))
	if tlsAuthEnabled {
		if sslCertificate == "" && len(tlsCertificates) == 0 {
			// The resource does have SSL/TLS auth enabled, but it was not certified by this template.
			if err := d.Set("client_tls_certificates", []interface{}{"Unknown certificate"}); err != nil {
				return err
			}
		}
	} else {
		if sslCertificate != "" {
			// The resource does not have SSL/TLS auth enabled, but this template provides an SSL certificate
			if err := d.Set("client_ssl_certificate", ""); err != nil {
				return err
			}
		}
		if len(tlsCertificates) >= 0 {
			// The resource does not have SSL/TLS auth enabled, but this template provides TLS certificates.
			if err := d.Set("client_tls_certificates", []interface{}{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateTagsfunc(tagsRaw interface{}, _ cty.Path) diag.Diagnostics {
	tags := tagsRaw.(map[string]interface{})
	invalid := make([]string, 0)
	for k, v := range tags {
		if k != strings.ToLower(k) {
			invalid = append(invalid, k)
		}
		vStr := v.(string)
		if vStr != strings.ToLower(vStr) {
			invalid = append(invalid, vStr)
		}
	}

	if len(invalid) > 0 {
		return diag.Errorf("tag keys and values must be lower case, invalid entries: %s", strings.Join(invalid, ", "))
	}
	return nil
}

func ToDatabaseId(id string) (int, int, error) {
	parts := strings.Split(id, "/")

	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid id: %s", id)
	}

	if len(parts) == 1 {
		dbId, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		return 0, dbId, nil
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	dbId, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return subId, dbId, nil
}

func ReadTags(ctx context.Context, api *ApiClient, subId int, databaseId int, d *schema.ResourceData) error {
	tags := make(map[string]string)
	tagResponse, err := api.Client.Tags.Get(ctx, subId, databaseId)
	if err != nil {
		return err
	}
	if tagResponse.Tags != nil {
		for _, t := range *tagResponse.Tags {
			tags[redis.StringValue(t.Key)] = redis.StringValue(t.Value)
		}
	}
	return d.Set("tags", tags)
}

func ReadPaymentMethodID(d *schema.ResourceData) (*int, error) {
	pmID := d.Get("payment_method_id").(string)
	if pmID != "" {
		pmID, err := strconv.Atoi(pmID)
		if err != nil {
			return nil, err
		}
		return redis.Int(pmID), nil
	}
	return nil, nil
}

func RemoteBackupIntervalSetCorrectly(key string) schema.CustomizeDiffFunc {
	// Validate multiple attributes - https://github.com/hashicorp/terraform-plugin-sdk/issues/233

	return func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
		if v, ok := diff.GetOk(key); ok {
			backups := v.([]interface{})
			if len(backups) == 1 {
				v := backups[0].(map[string]interface{})

				interval := v["interval"].(string)
				timeUtc := v["time_utc"].(string)

				if interval != databases.BackupIntervalEvery12Hours && interval != databases.BackupIntervalEvery24Hours && timeUtc != "" {
					return fmt.Errorf("unexpected value at %s.0.time_utc - time_utc can only be set when interval is either %s or %s", key, databases.BackupIntervalEvery24Hours, databases.BackupIntervalEvery12Hours)
				}
			}
		}
		return nil
	}

}

func WriteTags(ctx context.Context, api *ApiClient, subId int, databaseId int, d *schema.ResourceData) error {
	tags := make([]*redisTags.Tag, 0)
	tState := d.Get("tags").(map[string]interface{})
	for k, v := range tState {
		tags = append(tags, &redisTags.Tag{
			Key:   redis.String(k),
			Value: redis.String(v.(string)),
		})
	}
	return api.Client.Tags.Put(ctx, subId, databaseId, redisTags.AllTags{Tags: &tags})
}

func BuildBackupPlan(data interface{}, periodicBackupPath interface{}) *databases.DatabaseBackupConfig {
	var d map[string]interface{}

	switch v := data.(type) {
	case []interface{}:
		if len(v) != 1 {
			if periodicBackupPath == nil {
				return &databases.DatabaseBackupConfig{Active: redis.Bool(false)}
			} else {
				return nil
			}
		}
		d = v[0].(map[string]interface{})
	default:
		d = v.(map[string]interface{})
	}

	config := databases.DatabaseBackupConfig{
		Active:      redis.Bool(true),
		Interval:    redis.String(d["interval"].(string)),
		StorageType: redis.String(d["storage_type"].(string)),
		StoragePath: redis.String(d["storage_path"].(string)),
	}

	if v := d["time_utc"].(string); v != "" {
		config.TimeUTC = redis.String(v)
	}

	return &config
}

func FlattenPricing(pricing []*pricing.Pricing) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)
	for _, p := range pricing {

		tf := map[string]interface{}{
			"database_name":        p.DatabaseName,
			"type":                 p.Type,
			"type_details":         p.TypeDetails,
			"quantity":             p.Quantity,
			"quantity_measurement": p.QuantityMeasurement,
			"price_per_unit":       p.PricePerUnit,
			"price_currency":       p.PriceCurrency,
			"price_period":         p.PricePeriod,
			"region":               p.Region,
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func FilterSubscriptions(subs []*subscriptions.Subscription, filters []func(sub *subscriptions.Subscription) bool) []*subscriptions.Subscription {
	var filteredSubs []*subscriptions.Subscription
	for _, sub := range subs {
		if filterSub(sub, filters) {
			filteredSubs = append(filteredSubs, sub)
		}
	}

	return filteredSubs
}

func filterSub(method *subscriptions.Subscription, filters []func(method *subscriptions.Subscription) bool) bool {
	for _, f := range filters {
		if !f(method) {
			return false
		}
	}
	return true
}

func BuildPrivateServiceConnectActiveActiveId(subId int, regionId int, pscServiceId int) string {
	return fmt.Sprintf("%d/%d/%d", subId, regionId, pscServiceId)
}
