package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	fixedSubscriptions "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var sweeperClients map[string]*rediscloudApi.Client

func TestMain(m *testing.M) {
	sweeperClients = make(map[string]*rediscloudApi.Client)
	resource.TestMain(m)
}

func sharedClientForRegion(region string) (*rediscloudApi.Client, error) {
	if client, ok := sweeperClients[region]; ok {
		return client, nil
	}

	if os.Getenv(utils.RedisCloudUrlEnvVar) == "" || os.Getenv(rediscloudApi.AccessKeyEnvVar) == "" || os.Getenv(rediscloudApi.SecretKeyEnvVar) == "" {
		return nil, fmt.Errorf("must provide environment variables %s, %s, %s", utils.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
	}

	client, err := rediscloudApi.NewClient(rediscloudApi.BaseURL(os.Getenv(utils.RedisCloudUrlEnvVar)))
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

	for _, account := range list {
		if redis.StringValue(account.Status) != cloud_accounts.StatusActive {
			continue
		}

		if !strings.HasPrefix(redis.StringValue(account.Name), utils.TestResourcePrefix) {
			continue
		}

		err := client.CloudAccount.Delete(context.TODO(), redis.IntValue(account.ID))
		if err != nil {
			return err
		}
	}

	return nil
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

	for _, sub := range list {
		if redis.StringValue(sub.Status) != subscriptions.SubscriptionStatusActive {
			continue
		}

		if !strings.HasPrefix(redis.StringValue(sub.Name), utils.TestResourcePrefix) {
			continue
		}

		if redis.StringValue(sub.DeploymentType) != subscriptions.SubscriptionDeploymentTypeSingleRegion {
			continue
		}

		subId := redis.IntValue(sub.ID)
		sweepSub, dbIds, err := testSweepReadDatabases(client, subId)
		if err != nil {
			return err
		}

		if !sweepSub {
			continue
		}

		for _, db := range dbIds {
			err := client.Database.Delete(context.TODO(), subId, db)
			if err != nil {
				return err
			}
		}

		err = client.Subscription.Delete(context.TODO(), subId)
		if err != nil {
			return err
		}
	}

	return nil
}

func testSweepReadDatabases(client *rediscloudApi.Client, subId int) (bool, []int, error) {
	var dbIds []int
	list := client.Database.List(context.TODO(), subId)

	for list.Next() {
		db := list.Value()

		if !redis.TimeValue(db.ActivatedOn).Add(24 * -1 * time.Hour).Before(time.Now()) {
			// Subscription _probably_ created within the last day, so assume someone might be
			// currently running the tests
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

		if !redis.TimeValue(db.ActivatedOn).Add(24 * -1 * time.Hour).Before(time.Now()) {
			// Subscription _probably_ created within the last day, so assume someone might be
			// currently running the tests
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

	for _, sub := range list {
		if redis.StringValue(sub.Status) != subscriptions.SubscriptionStatusActive {
			continue
		}

		if !strings.HasPrefix(redis.StringValue(sub.Name), utils.TestResourcePrefix) {
			continue
		}

		if redis.StringValue(sub.DeploymentType) != subscriptions.SubscriptionDeploymentTypeActiveActive {
			continue
		}

		subId := redis.IntValue(sub.ID)
		sweepSub, dbIds, err := testSweepReadDatabases(client, subId)
		if err != nil {
			return err
		}

		if !sweepSub {
			continue
		}

		for _, db := range dbIds {
			err := client.Database.Delete(context.TODO(), subId, db)
			if err != nil {
				return err
			}
		}

		err = client.Subscription.Delete(context.TODO(), subId)
		if err != nil {
			return err
		}
	}

	return nil
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

	for _, sub := range list {
		if redis.StringValue(sub.Status) != fixedSubscriptions.FixedSubscriptionStatusActive {
			continue
		}

		if !strings.HasPrefix(redis.StringValue(sub.Name), utils.TestResourcePrefix) {
			continue
		}

		subId := redis.IntValue(sub.ID)
		sweepSub, dbIds, err := testSweepReadEssentialsDatabases(client, subId)
		if err != nil {
			return err
		}

		if !sweepSub {
			continue
		}

		for _, db := range dbIds {
			err := client.FixedDatabases.Delete(context.TODO(), subId, db)
			if err != nil {
				return err
			}
		}

		err = client.FixedSubscriptions.Delete(context.TODO(), subId)
		if err != nil {
			return err
		}
	}

	return nil
}

func testSweepAcl(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return err
	}

	ctx := context.TODO()

	users, err := client.Users.List(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		if !strings.HasPrefix(redis.StringValue(user.Name), utils.TestResourcePrefix) {
			continue
		}

		if client.Users.Delete(ctx, redis.IntValue(user.ID)) != nil {
			return err
		}
	}

	roles, err := client.Roles.List(ctx)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if !strings.HasPrefix(redis.StringValue(role.Name), utils.TestResourcePrefix) {
			continue
		}

		if client.Roles.Delete(ctx, redis.IntValue(role.ID)) != nil {
			return err
		}
	}

	rules, err := client.RedisRules.List(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		// There are 3 'default' rules which can't be deleted (Read-Only, Read-Write, Full-Access)
		if redis.BoolValue(rule.IsDefault) {
			continue
		}

		if !strings.HasPrefix(redis.StringValue(rule.Name), utils.TestResourcePrefix) {
			continue
		}

		if client.RedisRules.Delete(ctx, redis.IntValue(rule.ID)) != nil {
			return err
		}
	}

	return nil
}
