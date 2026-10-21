package acceptance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("rediscloud_agent_memory", &resource.Sweeper{
		Name: "rediscloud_agent_memory",
		F:    sweepAgentMemory,
	})
}

func sweepAgentMemory(_ string) error {
	age, err := time.ParseDuration(envDefault("SWEEP_AGE_THRESHOLD", "24h"))
	if err != nil || age < 0 {
		return fmt.Errorf("invalid SWEEP_AGE_THRESHOLD: %q", os.Getenv("SWEEP_AGE_THRESHOLD"))
	}
	c, err := client.GetTestClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	subscriptions, err := c.Client.Subscription.List(ctx)
	if err != nil {
		return err
	}
	owned := make(map[string]string)
	for _, sub := range subscriptions {
		name := redis.StringValue(sub.Name)
		if sweepOwnsSubscription(name, utils.TestResourcePrefix()) {
			owned[strconv.Itoa(redis.IntValue(sub.ID))] = name
		}
	}
	if len(owned) == 0 {
		return nil
	}
	stores, err := c.Client.AgentMemory.List(ctx)
	if err != nil {
		return err
	}
	var failures []error
	cutoff := time.Now().Add(-age)
	for _, store := range stores {
		if !sweepOwnsStore(store, owned, cutoff) {
			continue
		}
		id := redis.StringValue(store.StoreID)
		if err := c.Client.AgentMemory.Delete(ctx, id); err != nil {
			var missing *api.NotFound
			if !errors.As(err, &missing) {
				failures = append(failures, fmt.Errorf("sweep Agent Memory store %s: %w", id, err))
			}
		}
	}
	return errors.Join(failures...)
}

func sweepOwnsSubscription(name, prefix string) bool {
	return strings.HasPrefix(name, prefix+"-") && strings.HasSuffix(name, "-am")
}

func sweepOwnsStore(store api.StoreSummary, subscriptions map[string]string, cutoff time.Time) bool {
	name, owned := subscriptions[redis.StringValue(store.SubscriptionID)]
	// A matching store name alone must never authorize deletion in another subscription.
	return owned && redis.StringValue(store.StoreID) != "" && store.CreatedAt != nil &&
		!store.CreatedAt.After(cutoff) && strings.HasPrefix(redis.StringValue(store.Name), name+"-")
}

func TestSweepAgentMemoryFixtureIsolation(t *testing.T) {
	for _, tc := range []struct {
		name, prefix string
		want         bool
	}{
		{"tf-test-abcdef-am", "tf-test-abcdef", true},
		{"tf-test-ghijkl-am", "tf-test-abcdef", false},
		{"tf-test-abcdef-am", "tf-test", true},
		{"tf-testing-abcdef-am", "tf-test", false},
		{"customer-prod", "tf-test", false},
		{"tf-test-abcdef", "tf-test", false},
	} {
		t.Run(tc.name+"/"+tc.prefix, func(t *testing.T) {
			if got := sweepOwnsSubscription(tc.name, tc.prefix); got != tc.want {
				t.Fatalf("subscription ownership=%t, want %t", got, tc.want)
			}
		})
	}
}

func TestSweepAgentMemoryOwnership(t *testing.T) {
	cutoff := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	old, recent := cutoff.Add(-time.Hour), cutoff.Add(time.Hour)
	for _, tc := range []struct {
		name, subscription, storeName, id string
		created                           *time.Time
		want                              bool
	}{
		{"owned and old", "123", "tf-test-abcdef-am-memory", "store-1", &old, true},
		{"boundary age", "123", "tf-test-abcdef-am-memory", "store-1", &cutoff, true},
		{"other subscription", "456", "tf-test-abcdef-am-memory", "store-1", &old, false},
		{"other fixture", "123", "tf-test-other-am-memory", "store-1", &old, false},
		{"prefix collision", "123", "tf-test-abcdef-amother", "store-1", &old, false},
		{"missing subscription", "", "tf-test-abcdef-am-memory", "store-1", &old, false},
		{"missing ID", "123", "tf-test-abcdef-am-memory", "", &old, false},
		{"missing timestamp", "123", "tf-test-abcdef-am-memory", "store-1", nil, false},
		{"recent store", "123", "tf-test-abcdef-am-memory", "store-1", &recent, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := api.StoreSummary{StoreID: redis.String(tc.id), SubscriptionID: redis.String(tc.subscription), Name: redis.String(tc.storeName), CreatedAt: tc.created}
			if got := sweepOwnsStore(store, map[string]string{"123": "tf-test-abcdef-am"}, cutoff); got != tc.want {
				t.Fatalf("sweep ownership=%t, want %t", got, tc.want)
			}
		})
	}
}
