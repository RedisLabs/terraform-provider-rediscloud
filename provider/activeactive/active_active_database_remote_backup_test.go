package activeactive_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// defaultBackupPrincipal is the AWS principal that Redis Cloud documents for
// writing backups to a customer S3 bucket.
const defaultBackupPrincipal = "arn:aws:iam::168085023892:root"

// backupPrincipal lets a run against a non-production Redis Cloud environment
// grant bucket access to that environment's backup principal instead.
func backupPrincipal() string {
	if v := os.Getenv("REDISCLOUD_BACKUP_AWS_PRINCIPAL"); v != "" {
		return v
	}
	return defaultBackupPrincipal
}

// TestAccResourceRedisCloudActiveActiveDatabase_RemoteBackup configures a
// working S3 remote backup on one region and changes its schedule in place.
func TestAccResourceRedisCloudActiveActiveDatabase_RemoteBackup(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	subscriptionName := utils.RandomWithPrefix() + "-subscription"
	databaseName := utils.RandomWithPrefix() + "-database"
	bucketName := strings.ToLower(subscriptionName + "-backup")
	password := acctest.RandString(20)
	storagePath := "s3://" + bucketName + "/us-east-1"

	const databaseResourceName = "rediscloud_active_active_subscription_database.example"
	const subscriptionResourceName = "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, envchecks.AWSProviderCheck),
		CheckDestroy: testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/database_remote_backup.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"password":          config.StringVariable(password),
					"bucket_name":       config.StringVariable(bucketName),
					"backup_principal":  config.StringVariable(backupPrincipal()),
					"interval":          config.StringVariable("every-24-hours"),
					"time_utc":          config.StringVariable("16:00"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                         "us-east-1",
						"remote_backup.#":              "1",
						"remote_backup.0.interval":     "every-24-hours",
						"remote_backup.0.time_utc":     "16:00",
						"remote_backup.0.storage_type": "aws-s3",
						"remote_backup.0.storage_path": storagePath,
					}),
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":            "us-east-2",
						"remote_backup.#": "0",
					}),
					checkActiveActiveRegionBackup(subscriptionResourceName, databaseResourceName, "us-east-1", "every-24-hours", storagePath),
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/database_remote_backup.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"password":          config.StringVariable(password),
					"bucket_name":       config.StringVariable(bucketName),
					"backup_principal":  config.StringVariable(backupPrincipal()),
					"interval":          config.StringVariable("every-6-hours"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                         "us-east-1",
						"remote_backup.#":              "1",
						"remote_backup.0.interval":     "every-6-hours",
						"remote_backup.0.storage_path": storagePath,
					}),
					checkActiveActiveRegionBackup(subscriptionResourceName, databaseResourceName, "us-east-1", "every-6-hours", storagePath),
				),
			},
		},
	})
}

func checkActiveActiveRegionBackup(subscriptionResourceName, databaseResourceName, region, interval, storagePath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subId, dbId, err := activeActiveDatabaseIDs(s, subscriptionResourceName, databaseResourceName)
		if err != nil {
			return err
		}
		api, err := client.GetTestClient()
		if err != nil {
			return err
		}
		db, err := api.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
		if err != nil {
			return fmt.Errorf("failed to get database: %w", err)
		}
		for _, crdb := range db.CrdbDatabases {
			if redis.StringValue(crdb.Region) != region {
				continue
			}
			if crdb.Backup == nil || !redis.BoolValue(crdb.Backup.Enabled) {
				return fmt.Errorf("remote backup is not enabled in region %s", region)
			}
			if got := redis.StringValue(crdb.Backup.Interval); got != interval {
				return fmt.Errorf("region %s remote backup interval = %q, want %q", region, got, interval)
			}
			if got := redis.StringValue(crdb.Backup.Destination); got != storagePath {
				return fmt.Errorf("region %s remote backup destination = %q, want %q", region, got, storagePath)
			}
			return nil
		}
		return fmt.Errorf("region %s not found on database %d", region, dbId)
	}
}
