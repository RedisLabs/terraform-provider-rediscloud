package utils_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestSuppressIfRedisVersionSatisfied(t *testing.T) {
	tests := []struct {
		name     string
		newVal   string
		actual   string
		suppress bool
	}{
		{name: "empty config value suppresses (Computed fallback)", newVal: "", actual: "8.6", suppress: true},
		{name: "missing actual does not suppress (e.g. create)", newVal: "8.4", actual: "", suppress: false},
		{name: "actual ahead within same major suppresses", newVal: "8.4", actual: "8.6", suppress: true},
		{name: "actual equal suppresses", newVal: "8.6", actual: "8.6", suppress: true},
		{name: "actual behind does not suppress (real upgrade)", newVal: "8.6", actual: "8.4", suppress: false},
		{name: "patch-level actual ahead suppresses", newVal: "7.4", actual: "7.4.2", suppress: true},
		{name: "actual ahead across major does not suppress", newVal: "7.4", actual: "8.0", suppress: false},
		{name: "actual behind across major does not suppress", newVal: "8.0", actual: "7.4", suppress: false},
		{name: "malformed new does not suppress", newVal: "not-a-version", actual: "8.6", suppress: false},
		{name: "malformed actual does not suppress", newVal: "8.4", actual: "not-a-version", suppress: false},
	}

	rsrc := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"redis_version_actual": {Type: schema.TypeString, Optional: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]interface{}{}
			if tc.actual != "" {
				raw["redis_version_actual"] = tc.actual
			}
			d := schema.TestResourceDataRaw(t, rsrc.Schema, raw)
			got := utils.SuppressIfRedisVersionSatisfied("redis_version", "", tc.newVal, d)
			assert.Equal(t, tc.suppress, got)
		})
	}
}

func TestRedisVersionSatisfied(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		actual    string
		satisfied bool
	}{
		{name: "running version ahead within major (bg auto-upgrade past request)", requested: "8.2", actual: "8.6", satisfied: true},
		{name: "running version behind (genuine upgrade needed)", requested: "8.4", actual: "8.2", satisfied: false},
		{name: "running version equals request", requested: "8.4", actual: "8.4", satisfied: true},
		{name: "unchanged version (e.g. non-version update)", requested: "7.4", actual: "7.4", satisfied: true},
		{name: "patch-level actual ahead satisfies", requested: "7.4", actual: "7.4.2", satisfied: true},
		{name: "higher requested minor not satisfied", requested: "8.10", actual: "8.2", satisfied: false},
		{name: "lower requested minor satisfied", requested: "8.2", actual: "8.10", satisfied: true},
		{name: "cross-major upgrade not satisfied", requested: "9.0", actual: "8.6", satisfied: false},
		{name: "cross-major downgrade not satisfied", requested: "7.0", actual: "8.6", satisfied: false},
		{name: "unparseable requested not satisfied", requested: "not-a-version", actual: "8.6", satisfied: false},
		{name: "unparseable actual not satisfied", requested: "8.4", actual: "not-a-version", satisfied: false},
		{name: "empty requested not satisfied", requested: "", actual: "8.6", satisfied: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.satisfied, utils.RedisVersionSatisfied(tc.requested, tc.actual))
		})
	}
}
