package acceptance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAccResourceIrisIdentityAPIKey_CRUDI(t *testing.T) {
	api := newIdentityClient(t)
	f := newFixture(t, "identity_api_key.tf")
	const keyAddress = "rediscloud_iris_identity_api_key.test"
	var keyID, token, otherStoreID string
	expiry := time.Now().Add(time.Hour).Unix()
	t.Cleanup(func() {
		if keyID != "" {
			cleanupIdentity(t, api, "/v1/api-keys/"+url.PathEscape(keyID))
		}
	})
	checkKey := func(name string, grants int, expiresAt int64) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			if err := f.remember(state); err != nil {
				return err
			}
			key := state.RootModule().Resources[keyAddress]
			keyID = key.Primary.ID
			otherStoreID = state.RootModule().Resources["rediscloud_iris_agent_memory.other"].Primary.ID
			if token == "" {
				token = key.Primary.Attributes["token"]
			}
			if token == "" || key.Primary.Attributes["token"] != token {
				return fmt.Errorf("the create-only key token was missing or changed during update")
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			var current struct {
				Name      string `json:"name"`
				ExpiresAt int64  `json:"expiresAt"`
				TenantID  string `json:"tenantId"`
				Grants    []struct {
					Tenant     string   `json:"tenant"`
					ResourceID string   `json:"resourceId"`
					Actions    []string `json:"actions"`
				} `json:"grants"`
			}
			if err := api.Do(ctx, http.MethodGet, "/v1/api-keys/"+url.PathEscape(keyID), nil, nil, &current); err != nil {
				return err
			}
			if current.Name != name || current.TenantID == "" || len(current.Grants) != grants {
				return fmt.Errorf("remote key metadata or grant count did not match configuration")
			}
			if current.ExpiresAt != expiresAt || key.Primary.Attributes["expires_at"] != strconv.FormatInt(expiresAt, 10) {
				return fmt.Errorf("remote key expiry did not match its Terraform configuration")
			}
			seen := map[string]bool{}
			for _, grant := range current.Grants {
				if grant.Tenant != current.TenantID || len(grant.Actions) != 1 || grant.Actions[0] != "read" {
					return fmt.Errorf("remote grant did not retain its tenant and read permission")
				}
				seen[grant.ResourceID] = true
			}
			if !seen[f.id] || (grants == 2 && !seen[otherStoreID]) {
				return fmt.Errorf("remote key grants did not reference the test stores")
			}
			return nil
		}
	}
	testCase := f.testCase(t, nil)
	testCase.CheckDestroy = resource.ComposeTestCheckFunc(testCase.CheckDestroy, func(*terraform.State) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := f.api.Do(ctx, http.MethodGet, basePath+otherStoreID, nil, nil, nil); !client.IrisNotFound(err) {
			if err != nil {
				return fmt.Errorf("secondary test store was not deleted: %w", err)
			}
			return fmt.Errorf("secondary test store still exists")
		}
		return identityAbsent(api, "/v1/api-keys/"+url.PathEscape(keyID))
	})
	testCase.Steps = []resource.TestStep{
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "key_name": config.StringVariable(f.name), "tenant": config.StringVariable(os.Getenv("REDISCLOUD_IRIS_IDENTITY_TENANT"))}, Check: checkKey(f.name, 1, 0)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "key_name": config.StringVariable(f.name), "tenant": config.StringVariable(os.Getenv("REDISCLOUD_IRIS_IDENTITY_TENANT")), "expires_at": config.IntegerVariable(expiry)}, Check: checkKey(f.name, 1, expiry)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "key_name": config.StringVariable(f.name), "tenant": config.StringVariable(os.Getenv("REDISCLOUD_IRIS_IDENTITY_TENANT"))}, Check: checkKey(f.name, 1, 0)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "key_name": config.StringVariable(f.name + "-renamed"), "tenant": config.StringVariable(os.Getenv("REDISCLOUD_IRIS_IDENTITY_TENANT"))}, Check: checkKey(f.name+"-renamed", 1, 0)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "key_name": config.StringVariable(f.name + "-renamed"), "tenant": config.StringVariable(os.Getenv("REDISCLOUD_IRIS_IDENTITY_TENANT")), "extra_grant": config.BoolVariable(true)}, Check: checkKey(f.name+"-renamed", 2, 0)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "key_name": config.StringVariable(f.name + "-renamed"), "tenant": config.StringVariable(os.Getenv("REDISCLOUD_IRIS_IDENTITY_TENANT")), "extra_grant": config.BoolVariable(true)}, PlanOnly: true},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "key_name": config.StringVariable(f.name + "-renamed"), "tenant": config.StringVariable(os.Getenv("REDISCLOUD_IRIS_IDENTITY_TENANT"))}, Check: checkKey(f.name+"-renamed", 1, 0)},
	}
	resource.Test(t, testCase)
}
