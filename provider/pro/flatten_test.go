package pro_test

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
)

func TestFlattenCloudDetails_WithResourceTags(t *testing.T) {
	details := []*subscriptions.CloudDetail{
		{
			Provider:       redis.String("AWS"),
			CloudAccountID: redis.Int(2),
			ResourceTags: []*subscriptions.ResourceTag{
				{Key: redis.String("environment"), Value: redis.String("production")},
				{Key: redis.String("team"), Value: redis.String("platform")},
			},
		},
	}

	flattened := pro.FlattenCloudDetails(details, true)

	assert.Len(t, flattened, 1)
	tags, ok := flattened[0]["resource_tags"].(map[string]string)
	assert.True(t, ok, "resource_tags should be a map[string]string")
	assert.Equal(t, map[string]string{
		"environment": "production",
		"team":        "platform",
	}, tags)
}

func TestFlattenCloudDetails_ResourceTagsNil(t *testing.T) {
	details := []*subscriptions.CloudDetail{
		{
			Provider:       redis.String("AWS"),
			CloudAccountID: redis.Int(2),
			ResourceTags:   nil,
		},
	}

	flattened := pro.FlattenCloudDetails(details, true)

	assert.Len(t, flattened, 1)
	tags, ok := flattened[0]["resource_tags"].(map[string]string)
	assert.True(t, ok, "resource_tags should be a non-nil map[string]string even when API returns nil")
	assert.Empty(t, tags)
}

func TestFlattenCloudDetails_ResourceTagsEmpty(t *testing.T) {
	details := []*subscriptions.CloudDetail{
		{
			Provider:       redis.String("AWS"),
			CloudAccountID: redis.Int(2),
			ResourceTags:   []*subscriptions.ResourceTag{},
		},
	}

	flattened := pro.FlattenCloudDetails(details, true)

	assert.Len(t, flattened, 1)
	tags, ok := flattened[0]["resource_tags"].(map[string]string)
	assert.True(t, ok)
	assert.Empty(t, tags)
}

// TestFlattenCloudDetails_DataSourceIncludesTags verifies the data source flatten
// path (isResource=false) also exposes resource_tags, since the data source schema
// declares it as Computed.
func TestFlattenCloudDetails_DataSourceIncludesTags(t *testing.T) {
	details := []*subscriptions.CloudDetail{
		{
			Provider:       redis.String("AWS"),
			CloudAccountID: redis.Int(2),
			ResourceTags: []*subscriptions.ResourceTag{
				{Key: redis.String("environment"), Value: redis.String("staging")},
			},
		},
	}

	flattened := pro.FlattenCloudDetails(details, false)

	assert.Len(t, flattened, 1)
	assert.Equal(t, map[string]string{"environment": "staging"}, flattened[0]["resource_tags"])
}
