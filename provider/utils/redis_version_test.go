package utils

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
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
			got := SuppressIfRedisVersionSatisfied("redis_version", "", tc.newVal, d)
			assert.Equal(t, tc.suppress, got)
		})
	}
}
