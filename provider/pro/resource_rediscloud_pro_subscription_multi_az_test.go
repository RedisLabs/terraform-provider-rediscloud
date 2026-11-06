package pro_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestMultipleAvailabilityZonesChangePlansReplacement(t *testing.T) {
	resource := pro.ResourceRedisCloudProSubscription()

	t.Run("true to false", func(t *testing.T) {
		diff := planSubscriptionChange(t, resource, subscriptionConfig(true), subscriptionConfig(false))
		requireReplacementForMultipleAvailabilityZones(t, diff)
	})

	t.Run("false to true", func(t *testing.T) {
		diff := planSubscriptionChange(t, resource, subscriptionConfig(false), subscriptionConfig(true))
		requireReplacementForMultipleAvailabilityZones(t, diff)
	})

	t.Run("name stays an in-place update", func(t *testing.T) {
		next := subscriptionConfig(true)
		next["name"] = "renamed"
		diff := planSubscriptionChange(t, resource, subscriptionConfig(true), next)
		require.False(t, diff.RequiresNew(), "name change attributes: %#v", diff.Attributes)
		name, ok := diff.Attributes["name"]
		require.True(t, ok, "expected an in-place name change, got %#v", diff.Attributes)
		require.NotEqual(t, name.Old, name.New)
		require.False(t, name.RequiresNew)
	})
}

func planSubscriptionChange(t *testing.T, resource *schema.Resource, previous map[string]interface{}, next map[string]interface{}) *terraform.InstanceDiff {
	t.Helper()

	state := schema.TestResourceDataRaw(t, resource.Schema, previous)
	state.SetId("123")

	fake := testhelpers.NewFakeAPI()
	fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
		return map[string]any{"id": 123, "status": "active"}, nil
	})

	diff, err := resource.Diff(
		context.Background(),
		state.State(),
		terraform.NewResourceConfigRaw(next),
		testhelpers.NewAPIClient(t, fake),
	)
	require.NoError(t, err)
	require.NotNil(t, diff)
	return diff
}

func requireReplacementForMultipleAvailabilityZones(t *testing.T, diff *terraform.InstanceDiff) {
	t.Helper()
	require.True(t, diff.RequiresNew(), "planned an in-place update: %#v", diff.Attributes)

	for key, attr := range diff.Attributes {
		if attr != nil && attr.RequiresNew && strings.Contains(key, "multiple_availability_zones") {
			return
		}
	}
	t.Fatalf("replacement was not caused by multiple_availability_zones: %#v", diff.Attributes)
}

func subscriptionConfig(multipleAvailabilityZones bool) map[string]interface{} {
	return map[string]interface{}{
		"name": "example",
		"cloud_provider": []interface{}{
			map[string]interface{}{
				"provider": "AWS",
				"region": []interface{}{
					map[string]interface{}{
						"region":                       "us-east-1",
						"multiple_availability_zones":  multipleAvailabilityZones,
						"networking_deployment_cidr":   "10.0.0.0/24",
						"preferred_availability_zones": []interface{}{"use1-az1", "use1-az2", "use1-az3"},
					},
				},
			},
		},
		"creation_plan": []interface{}{
			map[string]interface{}{
				"dataset_size_in_gb":           1.0,
				"quantity":                     1,
				"replication":                  true,
				"throughput_measurement_by":    "operations-per-second",
				"throughput_measurement_value": 1000,
			},
		},
	}
}
