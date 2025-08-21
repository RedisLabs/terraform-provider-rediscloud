package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var sslCertificate = "-----BEGIN CERTIFICATE-----\\nMIIFYzCCA0ugAwIBAgIUSy/xBxWLHmzVkfsC7GeF/fYzIaUwDQYJKoZIhvcNAQEL\\nBQAwQDELMAkGA1UEBhMCR0IxDzANBgNVBAgMBkxvbmRvbjEPMA0GA1UEBwwGTG9u\\nZG9uMQ8wDQYDVQQKDAZPQyBMdGQwIBcNMjQwNzAzMTQzMjI4WhgPMzAwNDA5MDQx\\nNDMyMjhaMEAxCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNVBAcM\\nBkxvbmRvbjEPMA0GA1UECgwGT0MgTHRkMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A\\nMIICCgKCAgEAmxpumNhDKi74Q0HsJ/2/V1WlpMPwuiquYklw9MQmmvYDP1peQ6hu\\nH1dMAg/dw59r0r3S/AOWReJNT6WWQnlXbWXyHbAILJfeFXweZOh3A99ei7YKEtB1\\n4wLjWypIYmtcvFRgXTo6kayBy1pBBKPJ0sl+I8UQ+StRB/cHfMoQy07Cx0TMhJPd\\nZH3OTlyPgdIsZ+CNr5YK8T9MmyExzOuA3yFB03Gd2SZxD4M3hbQQefsX8v6HqJSp\\nDe7wEvC083K7FxpXFzckamUatuQ5TV6TQERaFCoMYXTJUchIc56boRUthOhU56Tl\\n8ozcxria0KB930tyjd6fIT97Yctzth+ZVCIzp0U16q2jBYPQhjHU3C4rtVInFaN2\\nl/NDTAt3sCo6pAxVAw6ovmdRRZWZaiDm5Gx45aNRpcz9UHw0kjkG2HPW+PiZoaeQ\\nRcUTLOfr+Z7SDIBlyBNwzNt8j6s88SDTin9tp5oNL/WnCtNLkjf9SjN+nj3nxopW\\n9s7ocjV3nbBwfODI/t0u5yVwmM/xvrI1lail7IXwqHV2v1DTnh5BELstw+8i9NI0\\nj1dhIVQhwzsu5tgwQig8iXQTg6/kVxNnMgcUQdEJckk3aonjECOCJjs9aDULAbA1\\n9mcZnx57WeugJj8eMBeixIoRzHJcYTx92Mcrr0hUHi+OmIxTpu7ydRMCAwEAAaNT\\nMFEwHQYDVR0OBBYEFNA3DgT/I4yju8Im0fKb9u08WiPUMB8GA1UdIwQYMBaAFNA3\\nDgT/I4yju8Im0fKb9u08WiPUMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEL\\nBQADggIBAIIwf2fKxLpgh7kTmMRI7zHPlzwEMo2xzgjT1JifpMTHjv+xusdrYdRZ\\nN92vLy3BZ89XcdlVuYOQYdrrniWZ9hDk/lWP5PyAvyFkVAMijmwcySW5BioddDfc\\n7augJmaP2X8Qg0CwOAJazC9RSV6x31G9ah5x89Nsh0uQc5e4udLQsV68DOD9S7Go\\nQxFB8qK/Xx8Z0OytCch9Oh/yZZzL5xBZtla5TWFG8kgoaj7m91lddhX+px04l/fC\\n48zxHRMJSjr5O8SUX3AKx189D9aZEXWLVyfdDtJ7yJmbhOVMjB60+20Jqa1fgb0h\\n1Hh6E+TP5ObDFni3ocjcmnSwwBr9Ih6PlES/z77AK4KiA7S0A+MZQVshLN6n9GVw\\nK78HS19IAHO0A8BKdxphaGBKJzye8+/S6Meyemq2hysHczNFeYWU13UKu9daWOYS\\nPlmhjikHCOii7eipK0+GtTfmkgYmL6f7OykkZ+pVjYtiq7qTU5ZIlWlW2uoPh2Oq\\ngVf//6zduKBxxEcl0i0qDHclx144uCnDnibhlnXcngqexMqNZWEn2Ld/7/mm+jYN\\nMnA37eTHAJrJ+urvEmkdonF5FFUpZtet53abyd0eYzRrVXof6iroQcetgnJA+k+I\\n4HrYoxJnDrzHJ+ycJ457/tggup254bgeqmCzalLTUeVNr9H2/lbT\\n-----END CERTIFICATE-----"
var invalidSslCertificate = "I am not a valid certificate"

// enable_tls=true, client_ssl_certificate=<valid>
func TestAccResourceRedisCloudSubscriptionTls_createWithDatabaseWithEnabledTlsAndSslCert(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	password := acctest.RandString(20)
	const subscriptionName = "rediscloud_subscription.example"
	const databaseName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			utils.TestAccPreCheck(t)
			utils.TestAccAwsPreExistingCloudAccountPreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndCert, testCloudAccountName, name, 1, password, sslCertificate),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(subscriptionName, "name", name),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(subscriptionName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestMatchResourceAttr(databaseName, "db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(databaseName, "password"),
					resource.TestCheckResourceAttr(databaseName, "name", "tf-database"),
					resource.TestCheckResourceAttr(databaseName, "dataset_size_in_gb", "1"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*utils.ApiClient)
						sub, err := client.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.Client.Database.List(context.TODO(), subId)
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
			// Ensure that SSL users can upgrade to TLS
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndTlsCert, testCloudAccountName, name, 1, password, sslCertificate),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(subscriptionName, "name", name),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(subscriptionName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestMatchResourceAttr(databaseName, "db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(databaseName, "password"),
					resource.TestCheckResourceAttr(databaseName, "name", "tf-database"),
					resource.TestCheckResourceAttr(databaseName, "dataset_size_in_gb", "1"),
				),
			},
			// And that mTLS can be switched off altogether
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndNoCert, testCloudAccountName, name, 1, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(subscriptionName, "name", name),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(subscriptionName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestMatchResourceAttr(databaseName, "db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(databaseName, "password"),
					resource.TestCheckResourceAttr(databaseName, "name", "tf-database"),
					resource.TestCheckResourceAttr(databaseName, "dataset_size_in_gb", "1"),
				),
			},
			{
				ResourceName:            databaseName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_ssl_certificate", "client_tls_certificates"},
			},
		},
	})
}

// enable_tls=true, client_ssl_certificate=""
func TestAccResourceRedisCloudSubscriptionTls_createWithDatabaseWithEnabledTlsAndEmptySslCert(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	password := acctest.RandString(20)
	const subscriptionName = "rediscloud_subscription.example"
	const databaseName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t); utils.TestAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndWithoutCert, testCloudAccountName, name, 1, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(subscriptionName, "name", name),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(subscriptionName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestMatchResourceAttr(databaseName, "db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(databaseName, "password"),
					resource.TestCheckResourceAttr(databaseName, "name", "tf-database"),
					resource.TestCheckResourceAttr(databaseName, "dataset_size_in_gb", "1"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*utils.ApiClient)
						sub, err := client.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.Client.Database.List(context.TODO(), subId)
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
				ResourceName:            subscriptionName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"creation_plan"},
			},
		},
	})
}

// enable_tls=true, client_ssl_certificate=<invalid>
func TestAccResourceRedisCloudSubscriptionTls_createWithDatabaseWithEnabledTlsAndInvalidSslCert(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			utils.TestAccPreCheck(t)
			utils.TestAccAwsPreExistingCloudAccountPreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndCert, testCloudAccountName, name, 1, password, invalidSslCertificate),
				ExpectError: regexp.MustCompile("Error: 400 BAD_REQUEST - DATABASE_INVALID_CERT: Database certificate is invalid"),
			},
		},
	})
}

// enable_tls=false, client_ssl_certificate=<invalid>
func TestAccResourceRedisCloudSubscriptionTls_createWithDatabaseAndDisabledTlsAndInvalidCert(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			utils.TestAccPreCheck(t)
			utils.TestAccAwsPreExistingCloudAccountPreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithoutEnableTlsAndWithCert, testCloudAccountName, name, 1, password, invalidSslCertificate),
				ExpectError: regexp.MustCompile("Error: 400 BAD_REQUEST - DATABASE_INVALID_CERT: Database certificate is invalid"),
			},
		},
	})
}

// enable_tls=false, client_ssl_certificate="", client_tls_certificates=["something"]
func TestAccResourceRedisCloudSubscriptionTls_createWithoutEnableTlsAndTlsCert(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			utils.TestAccPreCheck(t)
			utils.TestAccAwsPreExistingCloudAccountPreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudWithoutEnableTlsAndWithTlsCert, testCloudAccountName, name, 1, password, sslCertificate),
				ExpectError: regexp.MustCompile("TLS certificates may not be provided while enable_tls is false"),
			},
		},
	})
}

func TestAccResourceRedisCloudSubscriptionTls_createWithSslCertAndTlsCert(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			utils.TestAccPreCheck(t)
			utils.TestAccAwsPreExistingCloudAccountPreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(bothSslAndTls, testCloudAccountName, name, 1, password, sslCertificate, sslCertificate),
				ExpectError: regexp.MustCompile("Conflicting configuration arguments"),
			},
		},
	})
}

// enable_tls=true, client_ssl_certificate="", client_tls_certificates=["something"]
func TestAccResourceRedisCloudSubscriptionTls_createWithDatabaseWithEnabledTlsAndTlsCert(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	password := acctest.RandString(20)
	const subscriptionName = "rediscloud_subscription.example"
	const databaseName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			utils.TestAccPreCheck(t)
			utils.TestAccAwsPreExistingCloudAccountPreCheck(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndTlsCert, testCloudAccountName, name, 1, password, sslCertificate),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(subscriptionName, "name", name),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(subscriptionName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(subscriptionName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestMatchResourceAttr(databaseName, "db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(databaseName, "password"),
					resource.TestCheckResourceAttr(databaseName, "name", "tf-database"),
					resource.TestCheckResourceAttr(databaseName, "dataset_size_in_gb", "1"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*utils.ApiClient)
						sub, err := client.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.Client.Database.List(context.TODO(), subId)
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
				ResourceName:            databaseName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_tls_certificates"},
			},
		},
	})
}

const subscriptionBoilerplate = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
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

  creation_plan {
    dataset_size_in_gb = 1
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    modules = []
  }
}
`

const testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndCert = subscriptionBoilerplate + `

resource "rediscloud_subscription_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
  protocol = "redis"
  dataset_size_in_gb = %d
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
`

const testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndWithoutCert = subscriptionBoilerplate + `

resource "rediscloud_subscription_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
  protocol = "redis"
  dataset_size_in_gb = %d
  support_oss_cluster_api = true
  data_persistence = "none"
  replication = false
  throughput_measurement_by = "operations-per-second"
  password = "%s"
  throughput_measurement_value = 10000
  source_ips = ["10.0.0.0/8"]
  enable_tls = true
}
`

const testAccResourceRedisCloudSubscriptionOneDbWithoutEnableTlsAndWithCert = subscriptionBoilerplate + `

resource "rediscloud_subscription_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
  protocol = "redis"
  dataset_size_in_gb = %d
  support_oss_cluster_api = true
  data_persistence = "none"
  replication = false
  throughput_measurement_by = "operations-per-second"
  password = "%s"
  throughput_measurement_value = 10000
  source_ips = ["10.0.0.0/8"]
  client_ssl_certificate = "%s"
}
`

const testAccResourceRedisCloudWithoutEnableTlsAndWithTlsCert = subscriptionBoilerplate + `

resource "rediscloud_subscription_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
  protocol = "redis"
  dataset_size_in_gb = %d
  support_oss_cluster_api = true
  data_persistence = "none"
  replication = false
  throughput_measurement_by = "operations-per-second"
  password = "%s"
  throughput_measurement_value = 10000
  source_ips = ["10.0.0.0/8"]
  enable_tls = false
  client_tls_certificates = ["%s"]
}
`

const bothSslAndTls = subscriptionBoilerplate + `

resource "rediscloud_subscription_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
  protocol = "redis"
  dataset_size_in_gb = %d
  support_oss_cluster_api = true
  data_persistence = "none"
  replication = false
  throughput_measurement_by = "operations-per-second"
  password = "%s"
  throughput_measurement_value = 10000
  source_ips = ["10.0.0.0/8"]
  enable_tls = true
  client_ssl_certificate = "%s"
  client_tls_certificates = ["%s"]
}
`

const testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndTlsCert = subscriptionBoilerplate + `

resource "rediscloud_subscription_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
  protocol = "redis"
  dataset_size_in_gb = %d
  support_oss_cluster_api = true
  data_persistence = "none"
  replication = false
  throughput_measurement_by = "operations-per-second"
  password = "%s"
  throughput_measurement_value = 10000
  source_ips = ["10.0.0.0/8"]
  enable_tls = true
  client_tls_certificates = ["%s"]
}
`

const testAccResourceRedisCloudSubscriptionOneDbWithEnableTlsAndNoCert = subscriptionBoilerplate + `

resource "rediscloud_subscription_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
  protocol = "redis"
  dataset_size_in_gb = %d
  support_oss_cluster_api = true
  data_persistence = "none"
  replication = false
  throughput_measurement_by = "operations-per-second"
  password = "%s"
  throughput_measurement_value = 10000
  source_ips = ["10.0.0.0/8"]
  enable_tls = true
}
`
