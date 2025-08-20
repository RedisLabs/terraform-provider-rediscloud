package utils

import (
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func FlattenCidrs(cidrs []*attachments.Cidr) []string {
	cidrStrings := make([]string, 0)
	for _, cidr := range cidrs {
		cidrStrings = append(cidrStrings, redis.StringValue(cidr.CidrAddress))
	}
	return cidrStrings
}

// GetString safely retrieves a string value from schema.ResourceData.
func GetString(d *schema.ResourceData, key string) *string {
	if v, ok := d.GetOk(key); ok {
		return redis.String(v.(string))
	}
	return redis.String("")
}

// GetBool safely retrieves a bool value from schema.ResourceData.
func GetBool(d *schema.ResourceData, key string) *bool {
	if v, ok := d.GetOk(key); ok {
		return redis.Bool(v.(bool))
	}
	return redis.Bool(false)
}

// GetInt safely retrieves an int value from schema.ResourceData.
func GetInt(d *schema.ResourceData, key string) *int {
	if v, ok := d.GetOk(key); ok {
		return redis.Int(v.(int))
	}
	return redis.Int(0)
}

func SetStringIfNotEmpty(d *schema.ResourceData, key string, setter func(*string)) {
	if v, ok := d.GetOk(key); ok {
		if s, valid := v.(string); valid && s != "" {
			setter(redis.String(s))
		}
	}
}

func SetIntIfPositive(d *schema.ResourceData, key string, setter func(*int)) {
	if v, ok := d.GetOk(key); ok {
		if i, valid := v.(int); valid && i > 0 {
			setter(redis.Int(i))
		}
	}
}

func SetInt(d *schema.ResourceData, key string, setter func(*int)) {
	if v, ok := d.GetOk(key); ok {
		if i, valid := v.(int); valid {
			setter(redis.Int(i))
		}
	}
}

func SetFloat64(d *schema.ResourceData, key string, setter func(*float64)) {
	if v, ok := d.GetOk(key); ok {
		if f, valid := v.(float64); valid {
			setter(redis.Float64(f))
		}
	}
}

func SetBool(d *schema.ResourceData, key string, setter func(*bool)) {
	if v, ok := d.GetOk(key); ok {
		if b, valid := v.(bool); valid {
			setter(redis.Bool(b))
		}
	}
}
