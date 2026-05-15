package activeactive_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var protoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"rediscloud": func() (tfprotov5.ProviderServer, error) {
		muxServer, err := provider.MuxProviderServerCreator(
			provider.NewSdkProvider("dev")(),
			provider.NewFrameworkProvider("dev")(),
		)
		if err != nil {
			return nil, err
		}
		return muxServer(), nil
	},
}

func testAccPreCheck(t *testing.T) {
	for _, name := range []string{rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar} {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

func testAccAwsPreExistingCloudAccountPreCheck(t *testing.T) {
	if _, ok := os.LookupEnv("AWS_TEST_CLOUD_ACCOUNT_NAME"); !ok {
		t.Fatalf("Missing `AWS_TEST_CLOUD_ACCOUNT_NAME` environment variable")
	}
}

var (
	sharedClient     *client.ApiClient
	sharedClientOnce sync.Once
	sharedClientErr  error
)

func getTestClient() (*client.ApiClient, error) {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = client.NewClient()
	})
	return sharedClient, sharedClientErr
}

func testRandomWithPrefix(n ...int) string {
	length := 6
	if len(n) > 0 {
		length = n[0]
	}
	prefix := os.Getenv("TEST_RESOURCE_PREFIX")
	if prefix == "" {
		prefix = "tf-test"
	}
	return prefix + "-" + acctest.RandString(length)
}

// checkAASubscriptionDestroy verifies that all rediscloud_active_active_subscription
// resources have been destroyed. Uses terraform-plugin-testing's terraform.State
// (required for ConfigFile/ConfigVariables gold standard test pattern).
func checkAASubscriptionDestroy(s *terraform.State) error {
	apiClient, err := getTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_active_active_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := apiClient.Client.Subscription.List(context.TODO())
		if err != nil {
			return err
		}

		for _, sub := range subs {
			if redis.IntValue(sub.ID) == subId {
				return fmt.Errorf("subscription %d still exists", subId)
			}
		}
	}

	return nil
}
