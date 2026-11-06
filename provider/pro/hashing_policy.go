package pro

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// HashingPolicyRequiresReplacement reports whether hashing_policy must force a
// new database.
//
// The Cloud API update accepts regexRules, and go-api UpdateDatabase exposes
// that field. Sending rules for a database whose sharding type is already
// redis-oss is rejected with DATABASE_SHARDING_TYPE_IS_IMMUTABLE. Clearing the
// rules is rejected with DATABASE_REGEX_RULES_CHANGES_BLOCKED. Editing the
// rules of a database that already has a hashing policy stays in place: that
// is a custom-regex-rules database, and the update is allowed.
//
// A new resource has no id yet, so the first create is not a replacement.
func HashingPolicyRequiresReplacement(resourceID string, oldRules, newRules []string) bool {
	if resourceID == "" || stringSlicesEqual(oldRules, newRules) {
		return false
	}
	return len(oldRules) == 0 || len(newRules) == 0
}

func hashingPolicyCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	oldRaw, newRaw := diff.GetChange("hashing_policy")
	if HashingPolicyRequiresReplacement(diff.Id(), hashingPolicyStrings(oldRaw), hashingPolicyStrings(newRaw)) {
		return diff.ForceNew("hashing_policy")
	}
	return nil
}

// hashingPolicyRulesForUpdate returns regex rules only when this apply changes
// a configured hashing policy. An omitted hashing_policy, including rules that
// were only read back into state, is left off the update so a redis-oss
// database is not asked to change its hashing policy.
func hashingPolicyRulesForUpdate(d *schema.ResourceData) []*string {
	if !d.HasChange("hashing_policy") {
		return nil
	}
	rules := configuredHashingPolicy(d)
	if len(rules) == 0 {
		return nil
	}
	return redis.StringSlice(rules...)
}

func createContextWithHashingPolicy(ctx context.Context, d *schema.ResourceData) context.Context {
	if len(configuredHashingPolicy(d)) == 0 {
		return ctx
	}
	return client.WithShardingType(ctx, client.ShardingTypeCustomRegexRules)
}

// configuredHashingPolicy returns the hashing_policy from configuration.
// A null attribute means the caller did not set it. When Terraform did not
// send a raw config, as in schema.TestResourceDataRaw, the resource data value
// is the configuration.
func configuredHashingPolicy(d *schema.ResourceData) []string {
	raw := d.GetRawConfig()
	if !raw.IsNull() && raw.IsKnown() && raw.Type().IsObjectType() && raw.Type().HasAttribute("hashing_policy") {
		attr := raw.GetAttr("hashing_policy")
		if attr.IsNull() {
			return nil
		}
		if attr.IsKnown() && attr.Type().IsListType() {
			return hashingPolicyFromCty(attr)
		}
	}
	return hashingPolicyStrings(d.Get("hashing_policy"))
}

func hashingPolicyFromCty(attr cty.Value) []string {
	rules := make([]string, 0, attr.LengthInt())
	for _, value := range attr.AsValueSlice() {
		if value.IsNull() || !value.IsKnown() || value.AsString() == "" {
			continue
		}
		rules = append(rules, value.AsString())
	}
	return rules
}

func hashingPolicyStrings(raw interface{}) []string {
	switch values := raw.(type) {
	case []string:
		rules := make([]string, 0, len(values))
		for _, rule := range values {
			if rule != "" {
				rules = append(rules, rule)
			}
		}
		return rules
	case []interface{}:
		rules := make([]string, 0, len(values))
		for _, value := range values {
			rule, ok := value.(string)
			if ok && rule != "" {
				rules = append(rules, rule)
			}
		}
		return rules
	default:
		return nil
	}
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
