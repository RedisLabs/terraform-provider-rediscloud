package provider_test

import (
	"cmp"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var s3Backup = flag.Bool("s3Backup", false, "Run remote backup acceptance tests with AWS CLI v2 and AWS credentials that can manage test buckets")

const (
	backupTimeUTC               = "16:00"
	defaultBackupAWSPrincipal   = "arn:aws:iam::168085023892:root"
	remoteBackupConfigDirectory = "testdata/database_remote_backup"
)

func TestAccResourceRedisCloudProDatabase_RemoteBackup(t *testing.T) {
	requireS3Backup(t)
	const address = "rediscloud_subscription_database.example"
	const stateAddress = address + ".0"
	tags, tagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	name := utils.RandomWithPrefix(12)
	principal := cmp.Or(os.Getenv("REDISCLOUD_BACKUP_AWS_PRINCIPAL"), defaultBackupAWSPrincipal)
	bucket := strings.ToLower(name) + "-backup"
	storagePath := "s3://" + bucket + "/backup"
	fixture := config.StaticDirectory(remoteBackupConfigDirectory)
	sameDatabase := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, tagsCheck, envchecks.RedisCloudCheck, envchecks.AWSProviderCheck),
		CheckDestroy: testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigDirectory:          fixture,
				ConfigVariables: config.Variables{
					"name":             config.StringVariable(name),
					"team":             config.StringVariable(tags.Team),
					"owner":            config.StringVariable(tags.Owner),
					"backup_principal": config.StringVariable(principal),
					"interval":         config.StringVariable(databases.BackupIntervalEvery24Hours),
					"time_utc":         config.StringVariable(backupTimeUTC),
				},
				ConfigStateChecks: []statecheck.StateCheck{sameDatabase.AddStateValue(address+"[0]", tfjsonpath.New("id"))},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(stateAddress, "remote_backup.#", "1"),
					resource.TestCheckResourceAttr(stateAddress, "remote_backup.0.interval", databases.BackupIntervalEvery24Hours),
					resource.TestCheckResourceAttr(stateAddress, "remote_backup.0.time_utc", backupTimeUTC),
					resource.TestCheckResourceAttr(stateAddress, "remote_backup.0.storage_type", "aws-s3"),
					resource.TestCheckResourceAttr(stateAddress, "remote_backup.0.storage_path", storagePath),
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigDirectory:          fixture,
				ConfigVariables: config.Variables{
					"name":             config.StringVariable(name),
					"team":             config.StringVariable(tags.Team),
					"owner":            config.StringVariable(tags.Owner),
					"backup_principal": config.StringVariable(principal),
					"interval":         config.StringVariable(databases.BackupIntervalEvery6Hours),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameDatabase.AddStateValue(address+"[0]", tfjsonpath.New("id")),
					statecheck.ExpectKnownValue(address+"[0]", tfjsonpath.New("remote_backup").AtSliceIndex(0).AtMapKey("time_utc"), knownvalue.StringExact("")),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(stateAddress, "remote_backup.0.interval", databases.BackupIntervalEvery6Hours),
					runRemoteBackup(t, stateAddress, "", bucket),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveDatabase_RemoteBackup(t *testing.T) {
	requireS3Backup(t)
	const address = "rediscloud_active_active_subscription_database.example"
	const stateAddress = address + ".0"
	const backupRegion = "us-east-1"
	tags, tagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	name := utils.RandomWithPrefix(12)
	password := acctest.RandString(20)
	principal := cmp.Or(os.Getenv("REDISCLOUD_BACKUP_AWS_PRINCIPAL"), defaultBackupAWSPrincipal)
	bucket := strings.ToLower(name) + "-backup"
	storagePath := "s3://" + bucket + "/backup"
	fixture := config.StaticDirectory(remoteBackupConfigDirectory)
	sameDatabase := statecheck.CompareValue(compare.ValuesSame())
	otherRegionHasNoBackup := resource.TestCheckTypeSetElemNestedAttrs(stateAddress, "override_region.*", map[string]string{
		"name":            "us-east-2",
		"remote_backup.#": "0",
	})

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, tagsCheck, envchecks.RedisCloudCheck, envchecks.AWSProviderCheck),
		CheckDestroy: testhelpers.CheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigDirectory:          fixture,
				ConfigVariables: config.Variables{
					"active_active":    config.BoolVariable(true),
					"name":             config.StringVariable(name),
					"team":             config.StringVariable(tags.Team),
					"owner":            config.StringVariable(tags.Owner),
					"password":         config.StringVariable(password),
					"backup_principal": config.StringVariable(principal),
				},
				ConfigStateChecks: []statecheck.StateCheck{sameDatabase.AddStateValue(address+"[0]", tfjsonpath.New("id"))},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(stateAddress, "override_region.*", map[string]string{
						"name":            backupRegion,
						"remote_backup.#": "0",
					}),
					otherRegionHasNoBackup,
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigDirectory:          fixture,
				ConfigVariables: config.Variables{
					"active_active":    config.BoolVariable(true),
					"name":             config.StringVariable(name),
					"team":             config.StringVariable(tags.Team),
					"owner":            config.StringVariable(tags.Owner),
					"password":         config.StringVariable(password),
					"backup_principal": config.StringVariable(principal),
					"interval":         config.StringVariable(databases.BackupIntervalEvery24Hours),
					"time_utc":         config.StringVariable(backupTimeUTC),
				},
				ConfigStateChecks: []statecheck.StateCheck{sameDatabase.AddStateValue(address+"[0]", tfjsonpath.New("id"))},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(stateAddress, "override_region.*", map[string]string{
						"name":                         backupRegion,
						"remote_backup.#":              "1",
						"remote_backup.0.interval":     databases.BackupIntervalEvery24Hours,
						"remote_backup.0.time_utc":     backupTimeUTC,
						"remote_backup.0.storage_type": "aws-s3",
						"remote_backup.0.storage_path": storagePath,
					}),
					otherRegionHasNoBackup,
				),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigDirectory:          fixture,
				ConfigVariables: config.Variables{
					"active_active":    config.BoolVariable(true),
					"name":             config.StringVariable(name),
					"team":             config.StringVariable(tags.Team),
					"owner":            config.StringVariable(tags.Owner),
					"password":         config.StringVariable(password),
					"backup_principal": config.StringVariable(principal),
					"interval":         config.StringVariable(databases.BackupIntervalEvery6Hours),
				},
				ConfigStateChecks: []statecheck.StateCheck{sameDatabase.AddStateValue(address+"[0]", tfjsonpath.New("id"))},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(stateAddress, "override_region.*", map[string]string{
						"name":                         backupRegion,
						"remote_backup.#":              "1",
						"remote_backup.0.interval":     databases.BackupIntervalEvery6Hours,
						"remote_backup.0.storage_type": "aws-s3",
						"remote_backup.0.storage_path": storagePath,
					}),
					otherRegionHasNoBackup,
					runRemoteBackup(t, stateAddress, backupRegion, bucket),
				),
			},
		},
	})
}

func requireS3Backup(t *testing.T) {
	t.Helper()
	if !*s3Backup {
		t.Skip("Enable with -s3Backup using AWS CLI v2 and an AWS test account with bucket creation permissions")
	}
	if _, err := exec.LookPath("aws"); err != nil {
		t.Fatalf("-s3Backup requires AWS CLI v2: %v", err)
	}
}

func runRemoteBackup(t *testing.T, address, region, bucket string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		subscriptionID, databaseID, err := pro.ToDatabaseId(state.RootModule().Resources[address].Primary.ID)
		if err != nil {
			return err
		}
		api, err := client.GetTestClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
		defer cancel()
		previousFiles, err := listRemoteBackupFiles(ctx, bucket)
		if err != nil {
			return err
		}

		if region == "" {
			err = api.Client.Database.Backup(ctx, subscriptionID, databaseID)
		} else {
			err = startActiveActiveRemoteBackup(ctx, subscriptionID, databaseID, region)
		}
		if err != nil {
			return fmt.Errorf("start backup for %s: %w", address, err)
		}

		wait := retry.StateChangeConf{
			Pending:      []string{"pending"},
			Target:       []string{"uploaded"},
			Timeout:      5 * time.Minute,
			PollInterval: 10 * time.Second,
			Refresh: func() (interface{}, string, error) {
				files, err := listRemoteBackupFiles(ctx, bucket)
				if err != nil {
					return nil, "", err
				}
				for key, size := range files {
					if _, exists := previousFiles[key]; !exists {
						t.Logf("Verified RDB s3://%s/%s (%d bytes)", bucket, key, size)
						return key, "uploaded", nil
					}
				}
				return bucket, "pending", nil
			},
		}
		if _, err = wait.WaitForStateContext(ctx); err != nil {
			return fmt.Errorf("wait for new RDB in s3://%s/backup/: %w", bucket, err)
		}
		return nil
	}
}

func listRemoteBackupFiles(ctx context.Context, bucket string) (map[string]int64, error) {
	command := exec.CommandContext(ctx, "aws", "s3api", "list-objects-v2", "--region", "us-east-1",
		"--bucket", bucket, "--prefix", "backup/", "--output", "json", "--no-cli-pager")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list RDB files in s3://%s/backup/: %w", bucket, err)
	}
	var listing struct {
		Contents []struct {
			Key  string
			Size int64
		}
	}
	if err := json.Unmarshal(output, &listing); err != nil {
		return nil, fmt.Errorf("decode S3 object list: %w", err)
	}
	files := make(map[string]int64)
	for _, object := range listing.Contents {
		if object.Size > 0 && (strings.HasSuffix(object.Key, ".rdb") || strings.HasSuffix(object.Key, ".rdb.gz")) {
			files[object.Key] = object.Size
		}
	}
	return files, nil
}

func startActiveActiveRemoteBackup(ctx context.Context, subscriptionID, databaseID int, region string) error {
	base, err := url.Parse(os.Getenv(rediscloudapi.RedisCloudUrlEnvVar))
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil {
		return fmt.Errorf("%s must be a valid HTTPS URL", rediscloudapi.RedisCloudUrlEnvVar)
	}
	backupPath := base.JoinPath("subscriptions", strconv.Itoa(subscriptionID), "databases", strconv.Itoa(databaseID), "backup").Path
	api, err := rediscloudapi.NewClient(
		rediscloudapi.BaseURL(base.String()),
		rediscloudapi.Transporter(activeActiveBackupTransport{path: backupPath, region: region}),
	)
	if err != nil {
		return err
	}
	return api.Database.Backup(ctx, subscriptionID, databaseID)
}

type activeActiveBackupTransport struct {
	path   string
	region string
}

func (transport activeActiveBackupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// The SDK waits for the backup task but sends no body. Active-Active requires regionName.
	if request.Method == http.MethodPost && request.URL.Path == transport.path {
		request = request.Clone(request.Context())
		body := `{"regionName":` + strconv.Quote(transport.region) + `}`
		request.Body = io.NopCloser(strings.NewReader(body))
		request.ContentLength = int64(len(body))
		request.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultTransport.RoundTrip(request)
}
