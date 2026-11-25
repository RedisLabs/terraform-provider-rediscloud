package provider_test

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func init() {
	resource.AddTestSweepers("rediscloud_iris_identity_api_key", &resource.Sweeper{Name: "rediscloud_iris_identity_api_key", F: testSweepIrisIdentityAPIKeys})
	resource.AddTestSweepers("rediscloud_iris_agent_memory", &resource.Sweeper{Name: "rediscloud_iris_agent_memory", F: testSweepIrisAgentMemory, Dependencies: []string{"rediscloud_iris_identity_api_key"}})
}
func testSweepIrisAgentMemory(region string) error {
	if os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL") == "" {
		return nil
	}
	age, err := irisSweepAge()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, ids, err := irisOwnedStoreIDs(ctx, region, age)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := c.Do(ctx, "DELETE", "/iris/v1/agent-memory/stores/"+url.PathEscape(id), url.Values{"flush": {"false"}}, nil, nil); err != nil && !client.IrisNotFound(err) {
			return err
		}
	}
	return nil
}

// irisOwnedStoreIDs lists the stores that these acceptance tests created: old
// stores with the test prefix inside test-named databases of test-named
// Essentials subscriptions. Both the store and key sweepers use it, so neither
// can reach resources this run does not own. Candidates are collected before
// any deletion to keep pagination stable.
func irisOwnedStoreIDs(ctx context.Context, region string, age time.Duration) (*client.IrisServiceClient, []string, error) {
	owned, err := irisSweepDatabases(ctx, region, age)
	if err != nil || len(owned) == 0 {
		return nil, nil, err
	}
	c, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL"), Token: os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN"), APIKey: os.Getenv("REDISCLOUD_ACCESS_KEY"), SecretKey: os.Getenv("REDISCLOUD_SECRET_KEY")})
	if err != nil {
		return nil, nil, err
	}
	cursor := ""
	seen := map[string]bool{}
	var ids []string
	for {
		var page struct {
			Items []struct {
				Name      string `json:"name"`
				StoreID   string `json:"storeId"`
				CreatedAt string `json:"createdAt"`
				Database  struct {
					ID string `json:"databaseId"`
				} `json:"database"`
			} `json:"items"`
			NextCursor string `json:"nextCursor"`
		}
		query := url.Values{}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		if err := c.Do(ctx, "GET", "/iris/v1/agent-memory/stores", query, nil, &page); err != nil {
			return nil, nil, err
		}
		for _, store := range page.Items {
			created, err := time.Parse(time.RFC3339, store.CreatedAt)
			if err != nil || created.IsZero() || time.Since(created) < age || !owned[store.Database.ID] || !strings.HasPrefix(store.Name, testResourcePrefix+"-iris-acc-") {
				continue
			}
			if store.StoreID == "" {
				return nil, nil, fmt.Errorf("Iris sweeper received a matching store without an ID")
			}
			ids = append(ids, store.StoreID)
		}
		if page.NextCursor == "" {
			return c, ids, nil
		}
		if seen[page.NextCursor] {
			return nil, nil, fmt.Errorf("Iris sweeper received a repeated pagination cursor")
		}
		seen[page.NextCursor] = true
		cursor = page.NextCursor
	}
}

// Stores are swept only inside infrastructure created by these acceptance tests.
func irisSweepDatabases(ctx context.Context, region string, age time.Duration) (map[string]bool, error) {
	c, err := sharedClientForRegion(region)
	if err != nil {
		return nil, err
	}
	subscriptions, err := c.FixedSubscriptions.List(ctx)
	if err != nil {
		return nil, err
	}
	owned := map[string]bool{}
	prefix := testResourcePrefix + "-iris-acc-"
	for _, sub := range subscriptions {
		if !strings.HasPrefix(redis.StringValue(sub.Name), prefix) || sub.CreationDate == nil || sub.CreationDate.IsZero() || time.Since(*sub.CreationDate) < age {
			continue
		}
		databases := c.FixedDatabases.List(ctx, redis.IntValue(sub.ID))
		for databases.Next() {
			db := databases.Value()
			if strings.HasPrefix(redis.StringValue(db.Name), prefix) && redis.IntValue(db.DatabaseId) > 0 {
				owned[strconv.Itoa(*db.DatabaseId)] = true
			}
		}
		if err := databases.Err(); err != nil {
			return nil, err
		}
	}
	return owned, nil
}

func irisSweepAge() (time.Duration, error) {
	age := sweepAgeThreshold()
	if age < 0 || (age == 0 && os.Getenv("IRIS_SWEEP_ALLOW_ACTIVE") != "1") {
		return 0, fmt.Errorf("use a positive Iris sweep age; zero age requires IRIS_SWEEP_ALLOW_ACTIVE=1 after tests have stopped in an isolated account")
	}
	return age, nil
}

// Identity keys survive store deletion, so interrupted acceptance runs need
// their own cleanup. The key listing is global for a deployment-admin token and
// the test prefix is shared between developers, so a name alone proves nothing:
// a key is deleted only when every one of its grants points at a store that
// this run's store sweeper owns. This sweeper runs before the store sweeper, so
// those stores still exist. Keys whose stores are already gone cannot be proven
// ours and are left alone.
func testSweepIrisIdentityAPIKeys(region string) error {
	endpoint := os.Getenv("REDISCLOUD_IRIS_IDENTITY_URL")
	if endpoint == "" {
		return nil
	}
	age, err := irisSweepAge()
	if err != nil {
		return err
	}
	if os.Getenv("REDISCLOUD_IRIS_AGENT_MEMORY_URL") == "" {
		log.Printf("[WARN] Skipping the Iris Identity key sweep: without the Agent Memory endpoint, key ownership cannot be proven")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	_, storeIDs, err := irisOwnedStoreIDs(ctx, region, age)
	if err != nil {
		return err
	}
	if len(storeIDs) == 0 {
		return nil
	}
	ownedStores := map[string]bool{}
	for _, id := range storeIDs {
		ownedStores[id] = true
	}
	api, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: endpoint, Token: os.Getenv("REDISCLOUD_IRIS_IDENTITY_TOKEN"), APIKey: os.Getenv("REDISCLOUD_ACCESS_KEY"), SecretKey: os.Getenv("REDISCLOUD_SECRET_KEY"), Version: "test"})
	if err != nil {
		return err
	}
	query := url.Values{"product": {"memory"}, "resourceType": {"mem-store"}, "maxResults": {"100"}}
	seen := map[string]bool{}
	var candidates []string
	for {
		var page struct {
			APIKeys []struct {
				KeyID     string `json:"keyId"`
				Name      string `json:"name"`
				CreatedAt int64  `json:"createdAt"`
				Grants    []struct {
					Product      string `json:"product"`
					ResourceType string `json:"resourceType"`
					ResourceID   string `json:"resourceId"`
				} `json:"grants"`
			} `json:"apiKeys"`
			NextToken string `json:"nextToken"`
		}
		if err := api.Do(ctx, "GET", "/v1/api-keys", query, nil, &page); err != nil {
			return err
		}
		for _, key := range page.APIKeys {
			if key.CreatedAt <= 0 || time.Since(time.Unix(key.CreatedAt, 0)) < age || !strings.HasPrefix(key.Name, testResourcePrefix+"-iris-acc-") || len(key.Grants) == 0 {
				continue
			}
			ours := true
			for _, grant := range key.Grants {
				ours = ours && grant.Product == "memory" && grant.ResourceType == "mem-store" && ownedStores[grant.ResourceID]
			}
			if !ours {
				continue
			}
			if key.KeyID == "" {
				return fmt.Errorf("Iris Identity sweeper received a matching key without an ID")
			}
			candidates = append(candidates, key.KeyID)
		}
		if page.NextToken == "" {
			break
		}
		if seen[page.NextToken] {
			return fmt.Errorf("Iris Identity sweeper received a repeated pagination token")
		}
		seen[page.NextToken] = true
		query.Set("nextToken", page.NextToken)
	}
	for _, id := range candidates {
		if err := api.Do(ctx, "DELETE", "/v1/api-keys/"+url.PathEscape(id), nil, nil, nil); err != nil && !client.IrisNotFound(err) {
			return err
		}
	}
	return nil
}
