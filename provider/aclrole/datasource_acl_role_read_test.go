package aclrole_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestDataSourceRedisCloudAclRole_emptyName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/acl/roles" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"roles":[{"id":42,"name":"only-role"}]}`); err != nil {
			t.Errorf("write roles response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv(rediscloudApi.RedisCloudUrlEnvVar, server.URL)
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "unused")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "unused")

	const dataSourceName = "data.rediscloud_acl_role.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
data "rediscloud_acl_role" "test" {
  name = ""
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "42"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "only-role"),
				),
			},
		},
	})
}
