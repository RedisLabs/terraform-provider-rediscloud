package activeactive_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/activeactive"
)

func hasModifier(t *testing.T, resp resource.SchemaResponse, attr string, want planmodifier.String) {
	t.Helper()

	a, ok := resp.Schema.Attributes[attr]
	if !assert.True(t, ok, "attribute %q missing", attr) {
		return
	}

	sa, ok := a.(schema.StringAttribute)
	if !assert.True(t, ok, "attribute %q is not a StringAttribute", attr) {
		return
	}

	found := false
	for _, m := range sa.PlanModifiers {
		if reflect.TypeOf(m) == reflect.TypeOf(want) {
			found = true
		}
	}
	if !assert.True(t, found, "attribute %q missing expected plan modifier %T", attr, want) {
		return
	}
}

// Wiring: prove the REAL Active-Active database resource attaches these modifiers to the version
// attributes (the behaviour tests above run on a test resource, not this wiring).
func TestUnitRedisVersionModifiersWired(t *testing.T) {
	var resp resource.SchemaResponse
	activeactive.NewActiveActiveDatabaseResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	assert.False(t, resp.Diagnostics.HasError())

	hasModifier(t, resp, "redis_version", activeactive.RedisVersion())
	hasModifier(t, resp, "redis_version_actual", activeactive.RedisVersionActual())
}
