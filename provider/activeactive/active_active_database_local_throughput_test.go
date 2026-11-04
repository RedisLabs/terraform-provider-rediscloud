package activeactive_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudActiveActiveDatabase_LocalThroughput(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()

	subscriptionName := utils.RandomWithPrefix() + "-subscription"
	databaseName := utils.RandomWithPrefix() + "-database"
	password := acctest.RandString(20)

	const databaseResourceName = "rediscloud_active_active_subscription_database.example"
	const subscriptionResourceName = "rediscloud_active_active_subscription.example"

	var subId, dbId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/local_throughput_create.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                              "us-east-1",
						"local_read_operations_per_second":  "2000",
						"local_write_operations_per_second": "1500",
					}),
					func(s *terraform.State) error {
						var err error
						subId, dbId, err = activeActiveDatabaseIDs(s, subscriptionResourceName, databaseResourceName)
						if err != nil {
							return err
						}
						return checkLocalThroughput(t, subId, dbId, map[string][2]int{
							"us-east-1": {2000, 1500},
							"us-east-2": {1000, 1000},
						})
					},
				),
			},
			{
				ConfigFile: config.StaticFile("./testdata/local_throughput_update.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                              "us-east-1",
						"local_read_operations_per_second":  "3000",
						"local_write_operations_per_second": "2500",
					}),
					func(s *terraform.State) error {
						currentSubID, currentDbID, err := activeActiveDatabaseIDs(s, subscriptionResourceName, databaseResourceName)
						if err != nil {
							return err
						}
						if currentDbID != dbId {
							return fmt.Errorf("database id changed from %d to %d", dbId, currentDbID)
						}
						return checkLocalThroughput(t, currentSubID, currentDbID, map[string][2]int{
							"us-east-1": {3000, 2500},
							"us-east-2": {1000, 1000},
						})
					},
				),
			},
		},
	})
}

func activeActiveDatabaseIDs(s *terraform.State, subscriptionResourceName, databaseResourceName string) (int, int, error) {
	subscription := s.RootModule().Resources[subscriptionResourceName]
	subID, err := strconv.Atoi(subscription.Primary.ID)
	if err != nil {
		return 0, 0, fmt.Errorf("couldn't parse subscription ID: %s", subscription.Primary.ID)
	}
	database := s.RootModule().Resources[databaseResourceName]
	dbID, err := strconv.Atoi(database.Primary.Attributes["db_id"])
	if err != nil {
		return 0, 0, fmt.Errorf("couldn't parse database ID: %s", database.Primary.Attributes["db_id"])
	}
	return subID, dbID, nil
}

func checkLocalThroughput(t *testing.T, subID, dbID int, want map[string][2]int) error {
	t.Helper()
	apiClient := client.MustTestClient(t)
	db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subID, dbID)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	for region, ops := range want {
		read, write, ok := crdbRegionOps(db, region)
		if !ok {
			return fmt.Errorf("region %s has no local throughput on the API response", region)
		}
		if read != ops[0] || write != ops[1] {
			return fmt.Errorf("region %s read/write = %d/%d, want %d/%d", region, read, write, ops[0], ops[1])
		}
	}
	return nil
}

func crdbRegionOps(db *databases.ActiveActiveDatabase, region string) (int, int, bool) {
	for _, crdb := range db.CrdbDatabases {
		if redis.StringValue(crdb.Region) != region {
			continue
		}
		if crdb.ReadOperationsPerSecond == nil || crdb.WriteOperationsPerSecond == nil {
			return 0, 0, false
		}
		return *crdb.ReadOperationsPerSecond, *crdb.WriteOperationsPerSecond, true
	}
	return 0, 0, false
}
