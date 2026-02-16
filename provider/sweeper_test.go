package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	fixedSubscriptions "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// testResourcePrefix is the prefix used for all test resource names.
// Set TEST_RESOURCE_PREFIX env var to override (e.g. "tf-ci-pr7-42" in CI).
// This also controls which resources the sweeper targets.
var testResourcePrefix = getTestResourcePrefix()

func getTestResourcePrefix() string {
	if prefix := os.Getenv("TEST_RESOURCE_PREFIX"); prefix != "" {
		return prefix
	}
	return "tf-test"
}

// testRandomWithPrefix is like acctest.RandomWithPrefix but uses a shorter
// random component to stay within the API's 40-char name limit.
// Defaults to 6 chars; pass an explicit length to override.
func testRandomWithPrefix(n ...int) string {
	length := 6
	if len(n) > 0 {
		length = n[0]
	}
	return testResourcePrefix + "-" + acctest.RandString(length)
}

// sweepAgeThreshold returns the minimum age a database must exceed before
// the sweeper will consider it for deletion. Controlled by SWEEP_AGE_THRESHOLD:
//
//	Not set  → 24h (local dev safety — assume someone is running tests)
//	"2h"     → 2 hours (CI pre-sweep with broad prefix — protects concurrent runs)
//	"0s"     → immediate (cleanup of a cancelled run's unique prefix)
//
// Any value accepted by time.ParseDuration is valid.
func sweepAgeThreshold() time.Duration {
	raw := os.Getenv("SWEEP_AGE_THRESHOLD")
	if raw == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Fatalf("[ERROR] Invalid SWEEP_AGE_THRESHOLD %q: %v (expected Go duration like '2h', '30m', '0s')", raw, err)
	}
	log.Printf("[INFO] Sweep age threshold set to %s via SWEEP_AGE_THRESHOLD", d)
	return d
}

// maxSweepConcurrency limits parallel sweep operations to avoid API rate limits
const maxSweepConcurrency = 5

var sweeperClients map[string]*rediscloudApi.Client

// sweepErrorCollector collects errors from concurrent sweep operations
type sweepErrorCollector struct {
	mu     sync.Mutex
	errors []error
}

func (c *sweepErrorCollector) add(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = append(c.errors, err)
}

func (c *sweepErrorCollector) result() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.errors) == 0 {
		return nil
	}
	return errors.Join(c.errors...)
}

func TestMain(m *testing.M) {
	sweeperClients = make(map[string]*rediscloudApi.Client)
	resource.TestMain(m)
}

func sharedClientForRegion(region string) (*rediscloudApi.Client, error) {
	if client, ok := sweeperClients[region]; ok {
		return client, nil
	}

	if os.Getenv(RedisCloudUrlEnvVar) == "" || os.Getenv(rediscloudApi.AccessKeyEnvVar) == "" || os.Getenv(rediscloudApi.SecretKeyEnvVar) == "" {
		return nil, fmt.Errorf("must provide environment variables %s, %s, %s", RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
	}

	client, err := rediscloudApi.NewClient(rediscloudApi.BaseURL(os.Getenv(RedisCloudUrlEnvVar)))
	if err != nil {
		return nil, err
	}

	sweeperClients[region] = client

	return client, nil
}

func init() {
	resource.AddTestSweepers("rediscloud_subscription", &resource.Sweeper{
		Name: "rediscloud_subscription",
		F:    testSweepProSubscriptions,
	})
	resource.AddTestSweepers("rediscloud_active_active_subscription", &resource.Sweeper{
		Name: "rediscloud_active_active_subscription",
		F:    testSweepActiveActiveSubscriptions,
	})
	resource.AddTestSweepers("rediscloud_essentials_subscription", &resource.Sweeper{
		Name: "rediscloud_essentials_subscription",
		F:    testSweepEssentialsSubscriptions,
	})
	resource.AddTestSweepers("rediscloud_cloud_account", &resource.Sweeper{
		Name:         "rediscloud_cloud_account",
		Dependencies: []string{"rediscloud_subscription", "rediscloud_active_active_subscription", "rediscloud_essentials_subscription"}, // in case a subscription depends on an account
		F:            testSweepCloudAccounts,
	})
	resource.AddTestSweepers("rediscloud_acl", &resource.Sweeper{
		Name: "rediscloud_acl",
		F:    testSweepAcl,
	})
}

func testSweepCloudAccounts(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return err
	}

	list, err := client.CloudAccount.List(context.TODO())
	if err != nil {
		return err
	}

	// Filter accounts to sweep
	var toSweep []*cloud_accounts.CloudAccount
	for _, account := range list {
		if redis.StringValue(account.Status) != cloud_accounts.StatusActive {
			continue
		}
		if !strings.HasPrefix(redis.StringValue(account.Name), testResourcePrefix) {
			continue
		}
		toSweep = append(toSweep, account)
	}

	if len(toSweep) == 0 {
		return nil
	}

	log.Printf("[INFO] Sweeping %d cloud accounts in parallel (max %d concurrent)", len(toSweep), maxSweepConcurrency)

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxSweepConcurrency)
	errCollector := &sweepErrorCollector{}

	for _, account := range toSweep {
		wg.Add(1)
		go func(account *cloud_accounts.CloudAccount) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			accountId := redis.IntValue(account.ID)
			accountName := redis.StringValue(account.Name)

			if err := client.CloudAccount.Delete(context.TODO(), accountId); err != nil {
				log.Printf("[ERROR] Failed to delete cloud account %d (%s): %v", accountId, accountName, err)
				errCollector.add(fmt.Errorf("cloud account %d (%s): %w", accountId, accountName, err))
				return
			}

			log.Printf("[INFO] Successfully swept cloud account %d (%s)", accountId, accountName)
		}(account)
	}

	wg.Wait()
	return errCollector.result()
}

func testSweepProSubscriptions(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return err
	}

	list, err := client.Subscription.List(context.TODO())
	if err != nil {
		return err
	}

	// Filter subscriptions to sweep
	var toSweep []*subscriptions.Subscription
	for _, sub := range list {
		if redis.StringValue(sub.Status) != subscriptions.SubscriptionStatusActive {
			continue
		}
		if !strings.HasPrefix(redis.StringValue(sub.Name), testResourcePrefix) {
			continue
		}
		if redis.StringValue(sub.DeploymentType) != subscriptions.SubscriptionDeploymentTypeSingleRegion {
			continue
		}
		toSweep = append(toSweep, sub)
	}

	if len(toSweep) == 0 {
		return nil
	}

	log.Printf("[INFO] Sweeping %d Pro subscriptions in parallel (max %d concurrent)", len(toSweep), maxSweepConcurrency)

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxSweepConcurrency)
	errCollector := &sweepErrorCollector{}

	for _, sub := range toSweep {
		wg.Add(1)
		go func(sub *subscriptions.Subscription) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			subId := redis.IntValue(sub.ID)
			subName := redis.StringValue(sub.Name)

			sweepSub, dbIds, err := testSweepReadDatabases(client, subId)
			if err != nil {
				log.Printf("[ERROR] Failed to read databases for Pro subscription %d (%s): %v", subId, subName, err)
				errCollector.add(fmt.Errorf("subscription %d (%s): %w", subId, subName, err))
				return
			}

			if !sweepSub {
				log.Printf("[INFO] Skipping Pro subscription %d (%s) - databases too recent", subId, subName)
				return
			}

			// Delete databases sequentially within this subscription
			for _, db := range dbIds {
				if err := client.Database.Delete(context.TODO(), subId, db); err != nil {
					log.Printf("[ERROR] Failed to delete database %d in Pro subscription %d (%s): %v", db, subId, subName, err)
					errCollector.add(fmt.Errorf("subscription %d (%s) database %d: %w", subId, subName, db, err))
					return
				}
			}

			if err := client.Subscription.Delete(context.TODO(), subId); err != nil {
				log.Printf("[ERROR] Failed to delete Pro subscription %d (%s): %v", subId, subName, err)
				errCollector.add(fmt.Errorf("subscription %d (%s): %w", subId, subName, err))
				return
			}

			log.Printf("[INFO] Successfully swept Pro subscription %d (%s)", subId, subName)
		}(sub)
	}

	wg.Wait()
	return errCollector.result()
}

func testSweepReadDatabases(client *rediscloudApi.Client, subId int) (bool, []int, error) {
	var dbIds []int
	list := client.Database.List(context.TODO(), subId)

	for list.Next() {
		db := list.Value()

		if !redis.TimeValue(db.ActivatedOn).Add(-sweepAgeThreshold()).Before(time.Now()) {
			// Database not old enough to sweep (controlled by SWEEP_AGE_THRESHOLD)
			return false, nil, nil
		}

		status := redis.StringValue(db.Status)
		if status != databases.StatusActive &&
			status != databases.StatusRCPActiveChangeDraft &&
			status != databases.StatusActiveChangeDraft &&
			status != databases.StatusActiveChangePending {
			// Database not in an active state, so the database can't be deleted
			log.Printf("Skipping db %d/%d as it is in status %s", subId, redis.IntValue(db.ID), status)
			continue
		}

		dbIds = append(dbIds, redis.IntValue(db.ID))
	}

	if list.Err() != nil {
		return false, nil, list.Err()
	}

	return true, dbIds, nil
}

func testSweepReadEssentialsDatabases(client *rediscloudApi.Client, subId int) (bool, []int, error) {
	var dbIds []int
	list := client.FixedDatabases.List(context.TODO(), subId)

	for list.Next() {
		db := list.Value()

		if !redis.TimeValue(db.ActivatedOn).Add(-sweepAgeThreshold()).Before(time.Now()) {
			// Database not old enough to sweep
			return false, nil, nil
		}

		status := redis.StringValue(db.Status)
		if status != databases.StatusActive &&
			status != databases.StatusRCPActiveChangeDraft &&
			status != databases.StatusActiveChangeDraft &&
			status != databases.StatusActiveChangePending {
			// Database not in an active state, so the database can't be deleted
			log.Printf("Skipping db %d/%d as it is in status %s", subId, redis.IntValue(db.DatabaseId), status)
			continue
		}

		dbIds = append(dbIds, redis.IntValue(db.DatabaseId))
	}

	if list.Err() != nil {
		return false, nil, list.Err()
	}

	return true, dbIds, nil
}

func testSweepActiveActiveSubscriptions(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return err
	}

	list, err := client.Subscription.List(context.TODO())
	if err != nil {
		return err
	}

	// Filter subscriptions to sweep
	var toSweep []*subscriptions.Subscription
	for _, sub := range list {
		if redis.StringValue(sub.Status) != subscriptions.SubscriptionStatusActive {
			continue
		}
		if !strings.HasPrefix(redis.StringValue(sub.Name), testResourcePrefix) {
			continue
		}
		if redis.StringValue(sub.DeploymentType) != subscriptions.SubscriptionDeploymentTypeActiveActive {
			continue
		}
		toSweep = append(toSweep, sub)
	}

	if len(toSweep) == 0 {
		return nil
	}

	log.Printf("[INFO] Sweeping %d Active-Active subscriptions in parallel (max %d concurrent)", len(toSweep), maxSweepConcurrency)

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxSweepConcurrency)
	errCollector := &sweepErrorCollector{}

	for _, sub := range toSweep {
		wg.Add(1)
		go func(sub *subscriptions.Subscription) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			subId := redis.IntValue(sub.ID)
			subName := redis.StringValue(sub.Name)

			sweepSub, dbIds, err := testSweepReadDatabases(client, subId)
			if err != nil {
				log.Printf("[ERROR] Failed to read databases for Active-Active subscription %d (%s): %v", subId, subName, err)
				errCollector.add(fmt.Errorf("subscription %d (%s): %w", subId, subName, err))
				return
			}

			if !sweepSub {
				log.Printf("[INFO] Skipping Active-Active subscription %d (%s) - databases too recent", subId, subName)
				return
			}

			// Delete databases sequentially within this subscription
			for _, db := range dbIds {
				if err := client.Database.Delete(context.TODO(), subId, db); err != nil {
					log.Printf("[ERROR] Failed to delete database %d in Active-Active subscription %d (%s): %v", db, subId, subName, err)
					errCollector.add(fmt.Errorf("subscription %d (%s) database %d: %w", subId, subName, db, err))
					return
				}
			}

			if err := client.Subscription.Delete(context.TODO(), subId); err != nil {
				log.Printf("[ERROR] Failed to delete Active-Active subscription %d (%s): %v", subId, subName, err)
				errCollector.add(fmt.Errorf("subscription %d (%s): %w", subId, subName, err))
				return
			}

			log.Printf("[INFO] Successfully swept Active-Active subscription %d (%s)", subId, subName)
		}(sub)
	}

	wg.Wait()
	return errCollector.result()
}

func testSweepEssentialsSubscriptions(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return err
	}

	list, err := client.FixedSubscriptions.List(context.TODO())
	if err != nil {
		return err
	}

	// Note: Only one Essentials subscription can exist per account, so no parallelisation needed.
	// We still collect errors to report all failures at the end.
	errCollector := &sweepErrorCollector{}

	for _, sub := range list {
		if redis.StringValue(sub.Status) != fixedSubscriptions.FixedSubscriptionStatusActive {
			continue
		}

		if !strings.HasPrefix(redis.StringValue(sub.Name), testResourcePrefix) {
			continue
		}

		subId := redis.IntValue(sub.ID)
		subName := redis.StringValue(sub.Name)

		sweepSub, dbIds, err := testSweepReadEssentialsDatabases(client, subId)
		if err != nil {
			log.Printf("[ERROR] Failed to read databases for Essentials subscription %d (%s): %v", subId, subName, err)
			errCollector.add(fmt.Errorf("essentials subscription %d (%s): %w", subId, subName, err))
			continue
		}

		if !sweepSub {
			log.Printf("[INFO] Skipping Essentials subscription %d (%s) - databases too recent", subId, subName)
			continue
		}

		// Delete databases sequentially
		dbDeleteFailed := false
		for _, db := range dbIds {
			if err := client.FixedDatabases.Delete(context.TODO(), subId, db); err != nil {
				log.Printf("[ERROR] Failed to delete database %d in Essentials subscription %d (%s): %v", db, subId, subName, err)
				errCollector.add(fmt.Errorf("essentials subscription %d (%s) database %d: %w", subId, subName, db, err))
				dbDeleteFailed = true
				break
			}
		}

		if dbDeleteFailed {
			continue
		}

		if err := client.FixedSubscriptions.Delete(context.TODO(), subId); err != nil {
			log.Printf("[ERROR] Failed to delete Essentials subscription %d (%s): %v", subId, subName, err)
			errCollector.add(fmt.Errorf("essentials subscription %d (%s): %w", subId, subName, err))
			continue
		}

		log.Printf("[INFO] Successfully swept Essentials subscription %d (%s)", subId, subName)
	}

	return errCollector.result()
}

func testSweepAcl(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return err
	}

	ctx := context.TODO()
	errCollector := &sweepErrorCollector{}

	// Delete users first (users depend on roles)
	users, err := client.Users.List(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		if !strings.HasPrefix(redis.StringValue(user.Name), testResourcePrefix) {
			continue
		}

		userId := redis.IntValue(user.ID)
		userName := redis.StringValue(user.Name)

		if err := client.Users.Delete(ctx, userId); err != nil {
			log.Printf("[ERROR] Failed to delete ACL user %d (%s): %v", userId, userName, err)
			errCollector.add(fmt.Errorf("ACL user %d (%s): %w", userId, userName, err))
			continue
		}
		log.Printf("[INFO] Successfully swept ACL user %d (%s)", userId, userName)
	}

	// Delete roles (roles depend on rules)
	roles, err := client.Roles.List(ctx)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if !strings.HasPrefix(redis.StringValue(role.Name), testResourcePrefix) {
			continue
		}

		roleId := redis.IntValue(role.ID)
		roleName := redis.StringValue(role.Name)

		if err := client.Roles.Delete(ctx, roleId); err != nil {
			log.Printf("[ERROR] Failed to delete ACL role %d (%s): %v", roleId, roleName, err)
			errCollector.add(fmt.Errorf("ACL role %d (%s): %w", roleId, roleName, err))
			continue
		}
		log.Printf("[INFO] Successfully swept ACL role %d (%s)", roleId, roleName)
	}

	// Delete rules last
	rules, err := client.RedisRules.List(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		// There are 3 'default' rules which can't be deleted (Read-Only, Read-Write, Full-Access)
		if redis.BoolValue(rule.IsDefault) {
			continue
		}

		if !strings.HasPrefix(redis.StringValue(rule.Name), testResourcePrefix) {
			continue
		}

		ruleId := redis.IntValue(rule.ID)
		ruleName := redis.StringValue(rule.Name)

		if err := client.RedisRules.Delete(ctx, ruleId); err != nil {
			log.Printf("[ERROR] Failed to delete ACL rule %d (%s): %v", ruleId, ruleName, err)
			errCollector.add(fmt.Errorf("ACL rule %d (%s): %w", ruleId, ruleName, err))
			continue
		}
		log.Printf("[INFO] Successfully swept ACL rule %d (%s)", ruleId, ruleName)
	}

	return errCollector.result()
}
