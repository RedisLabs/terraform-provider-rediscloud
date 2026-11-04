package pro_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
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

// TestAccResourceRedisCloudProDatabase_RemoteBackup configures a working S3
// remote backup, changes its schedule in place, then runs a manual backup and
// checks that the backup files reached the bucket.
func TestAccResourceRedisCloudProDatabase_RemoteBackup(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	subscriptionName := utils.RandomWithPrefix()
	bucketName := strings.ToLower(subscriptionName + "-backup")
	storagePath := "s3://" + bucketName + "/pro"

	const databaseResourceName = "rediscloud_subscription_database.example"
	const backupsDataSourceName = "data.aws_s3_objects.backups.0"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, envchecks.AWSProviderCheck),
		CheckDestroy: testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/pro_database_remote_backup.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"bucket_name":       config.StringVariable(bucketName),
					"backup_principal":  config.StringVariable(backupPrincipal()),
					"interval":          config.StringVariable("every-24-hours"),
					"time_utc":          config.StringVariable("16:00"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.0.interval", "every-24-hours"),
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.0.time_utc", "16:00"),
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.0.storage_type", "aws-s3"),
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.0.storage_path", storagePath),
					checkProDatabaseBackupConfig(databaseResourceName, "every-24-hours", storagePath),
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/pro_database_remote_backup.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"bucket_name":       config.StringVariable(bucketName),
					"backup_principal":  config.StringVariable(backupPrincipal()),
					"interval":          config.StringVariable("every-6-hours"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.0.interval", "every-6-hours"),
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.0.time_utc", ""),
					resource.TestCheckResourceAttr(databaseResourceName, "remote_backup.0.storage_path", storagePath),
					checkProDatabaseBackupConfig(databaseResourceName, "every-6-hours", storagePath),
					runProDatabaseBackup(databaseResourceName),
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/pro_database_remote_backup.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"bucket_name":       config.StringVariable(bucketName),
					"backup_principal":  config.StringVariable(backupPrincipal()),
					"interval":          config.StringVariable("every-6-hours"),
					"list_backups":      config.BoolVariable(true),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(backupsDataSourceName, "keys.#", func(value string) error {
						n, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						if n == 0 {
							return fmt.Errorf("expected backup files under %s, found none", storagePath)
						}
						return nil
					}),
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/pro_database_remote_backup.tf"),
				ConfigVariables: config.Variables{
					"team":              config.StringVariable(resourceTags.Team),
					"owner":             config.StringVariable(resourceTags.Owner),
					"subscription_name": config.StringVariable(subscriptionName),
					"bucket_name":       config.StringVariable(bucketName),
					"backup_principal":  config.StringVariable(backupPrincipal()),
					"interval":          config.StringVariable("every-6-hours"),
				},
				ResourceName:      databaseResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The API does not return the storage type.
				ImportStateVerifyIgnore: []string{"remote_backup.0.storage_type"},
			},
		},
	})
}

func proDatabaseIDs(s *terraform.State, resourceName string) (int, int, error) {
	r, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return 0, 0, fmt.Errorf("resource %s not found in state", resourceName)
	}
	subId, err := strconv.Atoi(r.Primary.Attributes["subscription_id"])
	if err != nil {
		return 0, 0, fmt.Errorf("couldn't parse subscription_id %q: %w", r.Primary.Attributes["subscription_id"], err)
	}
	dbId, err := strconv.Atoi(r.Primary.Attributes["db_id"])
	if err != nil {
		return 0, 0, fmt.Errorf("couldn't parse db_id %q: %w", r.Primary.Attributes["db_id"], err)
	}
	return subId, dbId, nil
}

func checkProDatabaseBackupConfig(resourceName, interval, storagePath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subId, dbId, err := proDatabaseIDs(s, resourceName)
		if err != nil {
			return err
		}
		api, err := client.GetTestClient()
		if err != nil {
			return err
		}
		db, err := api.Client.Database.Get(context.TODO(), subId, dbId)
		if err != nil {
			return err
		}
		if db.Backup == nil || !redis.BoolValue(db.Backup.Enabled) {
			return fmt.Errorf("remote backup is not enabled on database %d", dbId)
		}
		if got := redis.StringValue(db.Backup.Interval); got != interval {
			return fmt.Errorf("remote backup interval = %q, want %q", got, interval)
		}
		if got := redis.StringValue(db.Backup.Destination); got != storagePath {
			return fmt.Errorf("remote backup destination = %q, want %q", got, storagePath)
		}
		return nil
	}
}

func runProDatabaseBackup(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subId, dbId, err := proDatabaseIDs(s, resourceName)
		if err != nil {
			return err
		}
		api, err := client.GetTestClient()
		if err != nil {
			return err
		}
		if err := api.Client.Database.Backup(context.TODO(), subId, dbId); err != nil {
			return fmt.Errorf("manual backup of database %d failed: %w", dbId, err)
		}
		return nil
	}
}
