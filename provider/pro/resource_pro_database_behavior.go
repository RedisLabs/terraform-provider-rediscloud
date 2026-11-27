package pro

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// Default RFC1918 private IP ranges used when public_endpoint_access is false.
var defaultPrivateIPRanges = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"}

// isDefaultSourceIPs reports whether source IPs are unset, the public default
// 0.0.0.0/0, or the RFC1918 private ranges used when public access is blocked.
func isDefaultSourceIPs(sourceIPs []*string) bool {
	if len(sourceIPs) == 0 {
		return true
	}

	if len(sourceIPs) == 1 && redis.StringValue(sourceIPs[0]) == "0.0.0.0/0" {
		return true
	}

	if len(sourceIPs) == len(defaultPrivateIPRanges) {
		privateIPSet := make(map[string]bool, len(defaultPrivateIPRanges))
		for _, ip := range defaultPrivateIPRanges {
			privateIPSet[ip] = true
		}
		for _, ip := range sourceIPs {
			if !privateIPSet[redis.StringValue(ip)] {
				return false
			}
		}
		return true
	}

	return false
}

func isDefaultSourceIPStrings(sourceIPs []string) bool {
	pointers := make([]*string, len(sourceIPs))
	for i, ip := range sourceIPs {
		pointers[i] = redis.String(ip)
	}
	return isDefaultSourceIPs(pointers)
}

func defaultSourceIPsForAccess(publicEndpointAccess *bool) []string {
	if publicEndpointAccess != nil && !*publicEndpointAccess {
		copied := make([]string, len(defaultPrivateIPRanges))
		copy(copied, defaultPrivateIPRanges)
		return copied
	}
	return []string{"0.0.0.0/0"}
}

// resolveReadSourceIPs applies the SDKv2 source_ips refresh rules.
// useSubscriptionDefault is true when the API value is empty or a default and
// state does not already hold a custom list.
func resolveReadSourceIPs(apiIPs, stateIPs []string) (ips []string, useSubscriptionDefault bool) {
	apiDefault := len(apiIPs) == 0 || isDefaultSourceIPStrings(apiIPs)
	stateHasCustom := !isDefaultSourceIPStrings(stateIPs)
	if apiDefault && stateHasCustom {
		return append([]string(nil), stateIPs...), false
	}
	if apiDefault {
		return nil, true
	}
	return append([]string(nil), apiIPs...), false
}

// ShouldRejectPasswordlessWithPassword errors if passwordless is enabled and a
// non-empty password is actively being set. passwordChanged means the password
// is in config, not only carried from state.
func ShouldRejectPasswordlessWithPassword(enablePasswordless bool, passwordChanged bool, newPassword string) error {
	if !enablePasswordless {
		return nil
	}
	if passwordChanged && newPassword != "" {
		return fmt.Errorf(`"enable_passwordless" cannot be true when "password" is set`)
	}
	return nil
}

// ShouldWarnRedis8Modules reports whether modules on Redis 8 or newer should warn.
func ShouldWarnRedis8Modules(version string, hasModules bool) bool {
	if !hasModules {
		return false
	}
	if len(version) > 0 {
		majorVersionStr := strings.Split(version, ".")[0]
		if majorVersion, err := strconv.Atoi(majorVersionStr); err == nil {
			return majorVersion >= 8
		}
	}
	return false
}

// ShouldSuppressModuleDiffsForRedis8 reports whether module changes are ignored
// because Redis 8 and newer bundle modules.
func ShouldSuppressModuleDiffsForRedis8(version string) bool {
	if len(version) == 0 {
		return false
	}
	majorVersionStr := strings.Split(version, ".")[0]
	if majorVersion, err := strconv.Atoi(majorVersionStr); err == nil {
		return majorVersion >= 8
	}
	return false
}

func redis8ModulesWarning(version string) diag.Diagnostic {
	return diag.NewWarningDiagnostic(
		"Modules are bundled by default in Redis 8.0+",
		fmt.Sprintf("The 'modules' block is deprecated for Redis %s and later versions, as modules (RediSearch, RedisJSON, RedisBloom, RedisTimeSeries) are bundled by default. You should remove the 'modules' block from your configuration.", version),
	)
}

func logRedis8ModulesWarning(version string) {
	log.Printf("[WARN] Modules are bundled by default in Redis %s and later versions. The 'modules' block is deprecated for Redis 8.0+ as modules (RediSearch, RedisJSON, RedisBloom, RedisTimeSeries) are bundled by default. You should remove the 'modules' block from your configuration.", version)
}

// backupTimesEquivalent reports the SDKv2 time_utc diff suppression for a
// 12-hour backup interval. The API returns the morning time when the
// configuration uses the afternoon time twelve hours later.
func backupTimesEquivalent(interval, oldValue, newValue string) bool {
	if interval != databases.BackupIntervalEvery12Hours {
		return false
	}

	oldTime, err := time.Parse("15:04", oldValue)
	if err != nil {
		return false
	}
	newTime, err := time.Parse("15:04", newValue)
	if err != nil {
		return false
	}

	return oldTime.Minute() == newTime.Minute() && oldTime.Add(12*time.Hour).Hour() == newTime.Hour()
}

func remoteBackupIntervalError() string {
	return fmt.Sprintf(
		"unexpected value at remote_backup.0.time_utc - time_utc can only be set when interval is either %s or %s",
		databases.BackupIntervalEvery24Hours,
		databases.BackupIntervalEvery12Hours,
	)
}

// shouldUpgradeRedisVersion is the SDKv2 upgrade guard. Blank versions never
// upgrade, and a target that already matches redis_version_actual does not
// upgrade. The actual comparison stays until redis_version stops being written
// from Read.
//
// TODO(TF3.0) once redis_version goes write-only-by-convention (Computed dropped, not written from Read), drop the actualRedisVersion guard — DSF already makes it unreachable in normal operation. Kept today as belt-and-suspenders against DSF regressions.
func shouldUpgradeRedisVersion(priorVersion, plannedVersion, actualVersion types.String) bool {
	if priorVersion.IsNull() || priorVersion.IsUnknown() || priorVersion.ValueString() == "" {
		return false
	}
	if plannedVersion.IsNull() || plannedVersion.IsUnknown() || plannedVersion.ValueString() == "" {
		return false
	}
	if priorVersion.ValueString() == plannedVersion.ValueString() {
		return false
	}
	actual := ""
	if !actualVersion.IsNull() && !actualVersion.IsUnknown() {
		actual = actualVersion.ValueString()
	}
	return actual != plannedVersion.ValueString()
}

func configuredRedisVersion(config, state ProDatabaseModel, hasState bool) string {
	if !config.RedisVersion.IsNull() && !config.RedisVersion.IsUnknown() && config.RedisVersion.ValueString() != "" {
		return config.RedisVersion.ValueString()
	}
	if hasState && !state.RedisVersion.IsNull() && !state.RedisVersion.IsUnknown() {
		return state.RedisVersion.ValueString()
	}
	return ""
}

// adjustRedisVersionPlan keeps the prior redis_version when the request is
// already satisfied by redis_version_actual. Empty config is satisfied so the
// computed state value stays put.
func adjustRedisVersionPlan(config, plan, state ProDatabaseModel, hasState bool) types.String {
	if !hasState || config.RedisVersion.IsUnknown() {
		return plan.RedisVersion
	}
	requested := ""
	if !config.RedisVersion.IsNull() {
		requested = config.RedisVersion.ValueString()
	}
	actual := ""
	if !state.RedisVersionActual.IsNull() && !state.RedisVersionActual.IsUnknown() {
		actual = state.RedisVersionActual.ValueString()
	}
	if utils.RedisVersionRequestSatisfied(requested, actual) {
		return state.RedisVersion
	}
	if config.RedisVersion.IsNull() {
		return plan.RedisVersion
	}
	return config.RedisVersion
}

type certificateState struct {
	sslCertificate  types.String
	tlsCertificates types.List
}

// applyCertificateHints mirrors utils.ApplyCertificateHints for framework values.
func applyCertificateHints(tlsAuthEnabled bool, ssl types.String, tlsCerts types.List) (certificateState, diag.Diagnostics) {
	var diags diag.Diagnostics
	sslValue := ""
	if !ssl.IsNull() && !ssl.IsUnknown() {
		sslValue = ssl.ValueString()
	}
	certs, certDiags := stringList(context.Background(), tlsCerts)
	diags.Append(certDiags...)
	if diags.HasError() {
		return certificateState{}, diags
	}

	result := certificateState{
		sslCertificate:  ssl,
		tlsCertificates: tlsCerts,
	}
	if ssl.IsUnknown() {
		result.sslCertificate = types.StringNull()
	}
	if tlsCerts.IsNull() || tlsCerts.IsUnknown() {
		result.tlsCertificates = types.ListNull(types.StringType)
	}

	if tlsAuthEnabled {
		if sslValue == "" && len(certs) == 0 {
			list, listDiags := types.ListValueFrom(context.Background(), types.StringType, []string{"Unknown certificate"})
			diags.Append(listDiags...)
			result.tlsCertificates = list
		}
		return result, diags
	}

	if sslValue != "" {
		//TODO(TF3.0) make client_ssl_certificate nullable
		result.sslCertificate = types.StringValue("")
	}
	empty, emptyDiags := types.ListValueFrom(context.Background(), types.StringType, []string{})
	diags.Append(emptyDiags...)
	result.tlsCertificates = empty
	return result, diags
}

type timeValidator struct{}

func (v timeValidator) Description(context.Context) string {
	return "value must be a valid time in HH:MM format (24-hour)"
}

func (v timeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v timeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() == "" {
		return
	}
	if _, err := time.Parse("15:04", req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid time format", fmt.Sprintf("Value must be a valid time in HH:MM format (24-hour), got: %q", req.ConfigValue.ValueString()))
	}
}

type cidrValidator struct{}

func (v cidrValidator) Description(context.Context) string { return "value must be a valid CIDR" }

func (v cidrValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v cidrValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, _, err := net.ParseCIDR(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid CIDR", fmt.Sprintf("Value must be a valid CIDR, got: %q", req.ConfigValue.ValueString()))
	}
}

type redisURLValidator struct{}

func (v redisURLValidator) Description(context.Context) string {
	return "value must be a redis:// URL"
}

func (v redisURLValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v redisURLValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	parsed, err := url.Parse(req.ConfigValue.ValueString())
	if err != nil || parsed.Scheme != "redis" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid replica URI", fmt.Sprintf("Value must be a URL with scheme redis, got: %q", req.ConfigValue.ValueString()))
	}
}

type lowercaseTagsValidator struct{}

func (v lowercaseTagsValidator) Description(context.Context) string {
	return "tag keys and values must be lower case"
}

func (v lowercaseTagsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v lowercaseTagsValidator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	tags := map[string]string{}
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	invalid := make([]string, 0)
	for key, value := range tags {
		if key != strings.ToLower(key) {
			invalid = append(invalid, key)
		}
		if value != strings.ToLower(value) {
			invalid = append(invalid, value)
		}
	}
	if len(invalid) > 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid tags",
			fmt.Sprintf("tag keys and values must be lower case, invalid entries: %s", strings.Join(invalid, ", ")),
		)
	}
}

// moduleSetValidator keeps the SDKv2 requirement that each module has a name.
// The set is an object attribute, so the object itself has no required fields.
type moduleSetValidator struct{}

func (v moduleSetValidator) Description(context.Context) string {
	return "each module must set name"
}

func (v moduleSetValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v moduleSetValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var modules []ModuleModel
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &modules, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, module := range modules {
		if module.Name.IsNull() || module.Name.IsUnknown() || module.Name.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid module", "each module must set name")
			return
		}
	}
}
