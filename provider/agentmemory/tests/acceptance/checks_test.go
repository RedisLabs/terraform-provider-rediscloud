package acceptance

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

type trackedResources struct {
	storeID string
	keyID   string
	secret  string
	stores  map[string]struct{}
	keys    map[string]string
}

func (r *trackedResources) remember(replaceStore, replaceKey bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		store, key := state.RootModule().Resources[storeResource], state.RootModule().Resources[keyResource]
		if store == nil || store.Primary == nil || key == nil || key.Primary == nil {
			return errors.New("Agent Memory store or API key is missing from state")
		}
		storeID, keyID, secret := store.Primary.ID, key.Primary.ID, key.Primary.Attributes["api_key"]
		if storeID == "" || keyID == "" || secret == "" {
			return errors.New("Agent Memory identity or generated API key is empty")
		}
		if r.storeID != "" && (storeID != r.storeID) != replaceStore {
			return fmt.Errorf("unexpected store replacement: expected replacement=%t", replaceStore)
		}
		if r.keyID != "" && (keyID != r.keyID) != replaceKey {
			return fmt.Errorf("unexpected API key replacement: expected replacement=%t", replaceKey)
		}
		if r.secret != "" && (secret != r.secret) != replaceKey {
			return errors.New("API key secret changed without replacement, or replacement reused its secret")
		}
		if r.stores == nil {
			r.stores, r.keys = make(map[string]struct{}), make(map[string]string)
		}
		r.stores[storeID], r.keys[keyID] = struct{}{}, storeID
		r.storeID, r.keyID, r.secret = storeID, keyID, secret
		return nil
	}
}

func checkRemoteResources(state *terraform.State) error {
	c, err := client.GetTestClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	stored, key := state.RootModule().Resources[storeResource], state.RootModule().Resources[keyResource]
	if stored == nil || stored.Primary == nil || key == nil || key.Primary == nil {
		return errors.New("Agent Memory resources are missing from state")
	}
	store, err := c.Client.AgentMemory.Get(ctx, stored.Primary.ID)
	if err != nil {
		return fmt.Errorf("read tracked Agent Memory store: %w", err)
	}
	attrs := stored.Primary.Attributes
	if redis.StringValue(store.StoreID) != stored.Primary.ID || redis.StringValue(store.Name) != attrs["name"] || redis.StringValue(store.DatabaseID) != attrs["database_id"] {
		return errors.New("Agent Memory store identity differs between state and API")
	}
	if store.ShortMemory == nil || strconv.Itoa(redis.IntValue(store.ShortMemory.TTLSeconds)) != attrs["short_term_ttl_seconds"] {
		return errors.New("Agent Memory short-term TTL differs between state and API")
	}
	if store.Endpoint == nil || redis.StringValue(store.Endpoint) == "" || redis.StringValue(store.Endpoint) != attrs["endpoint"] {
		return errors.New("Agent Memory endpoint is missing or differs between state and API")
	}
	metadata, err := c.Client.AgentMemory.GetAPIKey(ctx, stored.Primary.ID, key.Primary.ID)
	if err != nil {
		return fmt.Errorf("read tracked Agent Memory API key: %w", err)
	}
	if redis.StringValue(metadata.APIKeyID) != key.Primary.ID || redis.StringValue(metadata.KeyName) != key.Primary.Attributes["name"] {
		return errors.New("Agent Memory API key metadata differs between state and API")
	}
	return nil
}

func (r *trackedResources) checkSuperseded(_ *terraform.State) error {
	c, err := client.GetTestClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for id := range r.stores {
		if id != r.storeID {
			if err := storeAbsent(ctx, c, id); err != nil {
				return err
			}
		}
	}
	for id, storeID := range r.keys {
		if id != r.keyID {
			if err := keyAbsent(ctx, c, storeID, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *trackedResources) checkDestroyed(_ *terraform.State) error {
	c, err := client.GetTestClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for id := range r.stores {
		if err := storeAbsent(ctx, c, id); err != nil {
			return err
		}
	}
	for id, storeID := range r.keys {
		if err := keyAbsent(ctx, c, storeID, id); err != nil {
			return err
		}
	}
	return nil
}

func storeAbsent(ctx context.Context, c *client.ApiClient, id string) error {
	_, err := c.Client.AgentMemory.Get(ctx, id)
	var missing *api.NotFound
	if errors.As(err, &missing) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check deleted store %s: %w", id, err)
	}
	return fmt.Errorf("Agent Memory store %s still exists", id)
}

func keyAbsent(ctx context.Context, c *client.ApiClient, storeID, id string) error {
	_, err := c.Client.AgentMemory.GetAPIKey(ctx, storeID, id)
	var missingKey *api.APIKeyNotFound
	var missingStore *api.NotFound
	if errors.As(err, &missingKey) || errors.As(err, &missingStore) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check deleted API key %s: %w", id, err)
	}
	return fmt.Errorf("Agent Memory API key %s still exists", id)
}
