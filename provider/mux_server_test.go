package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"os"
	"strconv"
	"testing"
)

func TestMuxServer(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	sdkProvider := New("dev")()
	// frameworkProvider := NewFWProvider()

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: map[string]func() (tfprotov5.ProviderServer, error){
			"rediscloud": func() (tfprotov5.ProviderServer, error) {
				// This method signature will include frameworkProvider
				muxProviderServerCreator, err := MuxProviderServerCreator(sdkProvider)
				if err != nil {
					return nil, err
				}
				return muxProviderServerCreator(), nil
			},
		},
		PreCheck: func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		CheckDestroy: func(s *terraform.State) error {
			// TODO Not happy that this method has to know what types of resource were created.
			// Will destroy any resources that the terraform-framework provider creates
			return testCheckSDKResourcesDestroyed(s, sdkProvider)
		},
		Steps: []resource.TestStep{
			// Simple sdkv2 resource
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscription, testCloudAccountName, name),
				Check:  resource.TestCheckResourceAttr(resourceName, "name", name),
			},
			// Will add a simple terraform-framework resource, make sure the destroyer cleans it up
		},
	})
}

func testCheckSDKResourcesDestroyed(s *terraform.State, sdkProvider *schema.Provider) error {
	client := sdkProvider.Meta().(*apiClient)

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := client.client.Subscription.List(context.TODO())
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
