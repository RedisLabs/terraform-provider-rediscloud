package pro

import (
	"context"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	redisTags "github.com/RedisLabs/rediscloud-go-api/service/tags"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// proDatabaseAPI is the Cloud API surface the pro database resource uses.
// The live adapter calls rediscloud-go-api. Tests substitute a fake.
type proDatabaseAPI interface {
	Create(ctx context.Context, subscriptionID int, create databases.CreateDatabase) (int, error)
	Get(ctx context.Context, subscriptionID int, databaseID int) (*databases.Database, error)
	Update(ctx context.Context, subscriptionID int, databaseID int, update databases.UpdateDatabase) error
	Delete(ctx context.Context, subscriptionID int, databaseID int) error
	UpgradeRedisVersion(ctx context.Context, subscriptionID int, databaseID int, upgrade databases.UpgradeRedisVersion) error
	SubscriptionPublicEndpointAccess(ctx context.Context, subscriptionID int) (*bool, error)
	GetTags(ctx context.Context, subscriptionID int, databaseID int) (map[string]string, error)
	PutTags(ctx context.Context, subscriptionID int, databaseID int, tags map[string]string) error
}

type liveProDatabaseAPI struct {
	client *client.ApiClient
}

func (a liveProDatabaseAPI) Create(ctx context.Context, subscriptionID int, create databases.CreateDatabase) (int, error) {
	return a.client.Client.Database.Create(ctx, subscriptionID, create)
}

func (a liveProDatabaseAPI) Get(ctx context.Context, subscriptionID int, databaseID int) (*databases.Database, error) {
	return a.client.Client.Database.Get(ctx, subscriptionID, databaseID)
}

func (a liveProDatabaseAPI) Update(ctx context.Context, subscriptionID int, databaseID int, update databases.UpdateDatabase) error {
	return a.client.Client.Database.Update(ctx, subscriptionID, databaseID, update)
}

func (a liveProDatabaseAPI) Delete(ctx context.Context, subscriptionID int, databaseID int) error {
	return a.client.Client.Database.Delete(ctx, subscriptionID, databaseID)
}

func (a liveProDatabaseAPI) UpgradeRedisVersion(ctx context.Context, subscriptionID int, databaseID int, upgrade databases.UpgradeRedisVersion) error {
	return a.client.Client.Database.UpgradeRedisVersion(ctx, subscriptionID, databaseID, upgrade)
}

func (a liveProDatabaseAPI) SubscriptionPublicEndpointAccess(ctx context.Context, subscriptionID int) (*bool, error) {
	subscription, err := a.client.Client.Subscription.Get(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return nil, nil
	}
	return subscription.PublicEndpointAccess, nil
}

func (a liveProDatabaseAPI) GetTags(ctx context.Context, subscriptionID int, databaseID int) (map[string]string, error) {
	tags := map[string]string{}
	response, err := a.client.Client.Tags.Get(ctx, subscriptionID, databaseID)
	if err != nil {
		return nil, err
	}
	if response != nil && response.Tags != nil {
		for _, tag := range *response.Tags {
			tags[redis.StringValue(tag.Key)] = redis.StringValue(tag.Value)
		}
	}
	return tags, nil
}

func (a liveProDatabaseAPI) PutTags(ctx context.Context, subscriptionID int, databaseID int, tags map[string]string) error {
	payload := make([]*redisTags.Tag, 0, len(tags))
	for key, value := range tags {
		payload = append(payload, &redisTags.Tag{Key: redis.String(key), Value: redis.String(value)})
	}
	return a.client.Client.Tags.Put(ctx, subscriptionID, databaseID, redisTags.AllTags{Tags: &payload})
}

func (r *proDatabaseResource) waitForSubscription(ctx context.Context, subscriptionID int, timeout time.Duration) error {
	if r.waitForSubscriptionActive != nil {
		return r.waitForSubscriptionActive(ctx, subscriptionID, timeout)
	}
	return utils.WaitForSubscriptionToBeActive(ctx, subscriptionID, r.client, timeout)
}

func (r *proDatabaseResource) waitForDatabase(ctx context.Context, subscriptionID, databaseID int, timeout time.Duration) error {
	if r.waitForDatabaseActive != nil {
		return r.waitForDatabaseActive(ctx, subscriptionID, databaseID, timeout)
	}
	return utils.WaitForDatabaseToBeActive(ctx, subscriptionID, databaseID, r.client, timeout)
}
