package provider

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var tlsFlag = flag.Bool("tls", false,
	"Add this flag '-tls' to run tests for subscriptions and databases that use TLS")

func testAccTLSValidCertificatePreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "SSL_CERTIFICATE")
}

func testAccTLSInvalidCertificatePreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "SSL_CERTIFICATE_INVALID")
}

// enable_tls=true, client_ssl_certificate=<valid>
func TestAccResourceRedisCloudSubscription_createWithDatabaseWithEnabledTlsAndSslCert(t *testing.T) {

	if !*tlsFlag {
		t.Skip("The '-tls' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	clientSslCertificate := os.Getenv("SSL_CERTIFICATE")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsPreExistingCloudAccountPreCheck(t)
			testAccTLSValidCertificatePreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndCert, testCloudAccountName, name, 1, password, clientSslCertificate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(resourceName, "database.0.password"),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr(resourceName, "database.0.memory_limit_in_gb", "1"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*apiClient)
						sub, err := client.client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.client.Database.List(context.TODO(), subId)
						if listDb.Next() != true {
							return fmt.Errorf("no database found: %s", listDb.Err())
						}

						if listDb.Err() != nil {
							return listDb.Err()
						}

						database := listDb.Value()
						if *database.Security.SSLClientAuthentication != true {
							return fmt.Errorf("database SSL Authentication is not enabled: %v", *database.Security.SSLClientAuthentication)
						}

						return nil
					},
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"database.0.client_ssl_certificate"},
			},
		},
	})
}

// enable_tls=true, client_ssl_certificate=""
func TestAccResourceRedisCloudSubscription_createWithDatabaseWithEnabledTlsAndEmptySslCert(t *testing.T) {

	if !*tlsFlag {
		t.Skip("The '-tls' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndWithoutCert, testCloudAccountName, name, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(resourceName, "database.0.password"),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr(resourceName, "database.0.memory_limit_in_gb", "1"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*apiClient)
						sub, err := client.client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.client.Database.List(context.TODO(), subId)
						if listDb.Next() != true {
							return fmt.Errorf("no database found: %s", listDb.Err())
						}

						if listDb.Err() != nil {
							return listDb.Err()
						}

						database := listDb.Value()
						if *database.Security.EnableTls != true {
							return fmt.Errorf("database Tls flag is not enabled: %v", *database.Security.SSLClientAuthentication)
						}
						if *database.Security.SSLClientAuthentication != false {
							return fmt.Errorf("database SSL client Authentication is enabled: %v", *database.Security.SSLClientAuthentication)
						}

						return nil
					},
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// enable_tls=true, client_ssl_certificate=<invalid>
func TestAccResourceRedisCloudSubscription_createWithDatabaseWithEnabledTlsAndInvalidSslCert(t *testing.T) {

	if !*tlsFlag {
		t.Skip("The '-tls' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	invalidClientSslCertificate := os.Getenv("SSL_CERTIFICATE_INVALID")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsPreExistingCloudAccountPreCheck(t)
			testAccTLSInvalidCertificatePreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndCert, testCloudAccountName, name, 1, password, invalidClientSslCertificate),
				ExpectError: regexp.MustCompile("Error: 400 BAD_REQUEST - DATABASE_INVALID_CERT: Database certificate is invalid"),
			},
		},
	})
}

// enable_tls=false, client_ssl_certificate=<invalid>
func TestAccResourceRedisCloudSubscription_createWithDatabaseAndDisabledTlsAndInvalidCert(t *testing.T) {

	if !*tlsFlag {
		t.Skip("The '-tls' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	invalidClientSslCertificate := os.Getenv("SSL_CERTIFICATE_INVALID")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsPreExistingCloudAccountPreCheck(t)
			testAccTLSInvalidCertificatePreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithoutEnableTlsAndWithCert, testCloudAccountName, name, 1, password, invalidClientSslCertificate),
				ExpectError: regexp.MustCompile("Error: 400 BAD_REQUEST - DATABASE_INVALID_CERT: Database certificate is invalid"),
			},
		},
	})
}

const testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndCert = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
	security_group_ids = []
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = %d
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]
	enable_tls = true
	client_ssl_certificate = "%s"
  }
}
`

const testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndWithoutCert = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
	security_group_ids = []
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = %d
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]
	enable_tls = true
  }
}
`

const testAccResourceRedisCloudSubscriptionOneDbWithoutEnableTlsAndWithCert = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
	security_group_ids = []
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = %d
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]
	client_ssl_certificate = "%s"
  }
}
`
