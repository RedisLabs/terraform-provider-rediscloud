package pro_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestHashingPolicyRequiresReplacement(t *testing.T) {
	custom := []string{"(?<tag>.*)"}
	other := []string{".*\\{(?<tag>.*)\\}.*"}

	require.False(t, pro.HashingPolicyRequiresReplacement("", nil, custom), "create is not a replacement")
	require.False(t, pro.HashingPolicyRequiresReplacement("42/7", custom, custom), "unchanged rules stay in place")
	require.False(t, pro.HashingPolicyRequiresReplacement("42/7", custom, other), "editing rules stays in place")
	require.True(t, pro.HashingPolicyRequiresReplacement("42/7", nil, custom), "adding a hashing policy replaces the database")
	require.True(t, pro.HashingPolicyRequiresReplacement("42/7", custom, nil), "removing a hashing policy replaces the database")
}

func TestProDatabaseCreate_HashingPolicySetsCustomSharding(t *testing.T) {
	const rule = "(?<tag>.*)"
	fake, createBody, updateBody := newHashingPolicyAPI(t, []any{
		map[string]any{"ordinal": 0, "pattern": rule},
	})

	res := pro.ResourceRedisCloudProDatabase()
	data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
		"subscription_id":              42,
		"name":                         "hashing-db",
		"dataset_size_in_gb":           1.0,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 1000,
		"hashing_policy":               []interface{}{rule},
	})

	diags := res.CreateContext(context.Background(), data, testhelpers.NewAPIClient(t, fake))
	require.False(t, diags.HasError(), "create diagnostics: %+v", diags)

	fake.WithHandlersPaused(func() {
		require.Equal(t, "custom-regex-rules", (*createBody)["shardingType"])
		require.Equal(t, []any{rule}, (*updateBody)["regexRules"])
	})
	require.Equal(t, rule, data.Get("hashing_policy").([]interface{})[0])
}

func TestProDatabaseCreate_WithoutHashingPolicyOmitsShardingType(t *testing.T) {
	fake, createBody, updateBody := newHashingPolicyAPI(t, []any{})

	res := pro.ResourceRedisCloudProDatabase()
	data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
		"subscription_id":              42,
		"name":                         "hashing-db",
		"dataset_size_in_gb":           1.0,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 1000,
	})

	diags := res.CreateContext(context.Background(), data, testhelpers.NewAPIClient(t, fake))
	require.False(t, diags.HasError(), "create diagnostics: %+v", diags)

	fake.WithHandlersPaused(func() {
		_, hasShardingType := (*createBody)["shardingType"]
		require.False(t, hasShardingType)
		_, hasRegexRules := (*updateBody)["regexRules"]
		require.False(t, hasRegexRules)
	})
}

func TestProDatabaseUpdate_UnchangedHashingPolicyIsNotSent(t *testing.T) {
	const rule = "(?<tag>.*)"
	fake, _, updateBody := newHashingPolicyAPI(t, []any{
		map[string]any{"ordinal": 0, "pattern": rule},
	})

	res := pro.ResourceRedisCloudProDatabase()
	original := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
		"subscription_id":              42,
		"name":                         "hashing-db",
		"dataset_size_in_gb":           1.0,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 1000,
		"hashing_policy":               []interface{}{rule},
	})
	original.SetId("42/7")
	data := res.Data(original.State())
	require.False(t, data.HasChange("hashing_policy"))

	diags := res.UpdateContext(context.Background(), data, testhelpers.NewAPIClient(t, fake))
	require.False(t, diags.HasError(), "update diagnostics: %+v", diags)

	fake.WithHandlersPaused(func() {
		_, hasRegexRules := (*updateBody)["regexRules"]
		require.False(t, hasRegexRules)
	})
}

func newHashingPolicyAPI(t *testing.T, regexRules []any) (*testhelpers.FakeAPI, *map[string]any, *map[string]any) {
	t.Helper()

	fake := testhelpers.NewFakeAPI()
	var createBody map[string]any
	var updateBody map[string]any

	fake.Handle("GET /subscriptions/42", func(r *http.Request) (any, error) {
		return map[string]any{"id": 42, "status": "active", "publicEndpointAccess": true}, nil
	})
	fake.Handle("GET /subscriptions/42/databases/7", func(r *http.Request) (any, error) {
		return hashingPolicyDatabase(regexRules), nil
	})
	fake.Handle("POST /subscriptions/42/databases", func(r *http.Request) (any, error) {
		if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
			return nil, err
		}
		return fake.RegisterTask("databaseCreateRequest", 7), nil
	})
	fake.Handle("PUT /subscriptions/42/databases/7", func(r *http.Request) (any, error) {
		if err := json.NewDecoder(r.Body).Decode(&updateBody); err != nil {
			return nil, err
		}
		return fake.RegisterTask("databaseUpdateRequest", 7), nil
	})
	fake.Handle("GET /subscriptions/42/databases/7/tags", func(r *http.Request) (any, error) {
		return map[string]any{"tags": []any{}}, nil
	})
	fake.Handle("PUT /subscriptions/42/databases/7/tags", func(r *http.Request) (any, error) {
		return map[string]any{"tags": []any{}}, nil
	})

	return fake, &createBody, &updateBody
}

func hashingPolicyDatabase(regexRules []any) map[string]any {
	return map[string]any{
		"databaseId":           7,
		"name":                 "hashing-db",
		"protocol":             "redis",
		"status":               "active",
		"datasetSizeInGb":      1,
		"supportOSSClusterApi": false,
		"dataPersistence":      "none",
		"dataEvictionPolicy":   "volatile-lru",
		"replication":          true,
		"throughputMeasurement": map[string]any{
			"by":    "operations-per-second",
			"value": 1000,
		},
		"clustering": map[string]any{
			"numberOfShards": 1,
			"regexRules":     regexRules,
		},
		"security": map[string]any{
			"enableDefaultUser":       true,
			"password":                "secret",
			"sslClientAuthentication": false,
			"tlsClientAuthentication": false,
			"enableTls":               false,
			"sourceIps":               []string{"0.0.0.0/0"},
		},
		"modules":         []any{},
		"alerts":          []any{},
		"publicEndpoint":  "example.com:10000",
		"privateEndpoint": "example.internal:10000",
	}
}
