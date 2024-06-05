Terraform Provider Redis Cloud
==================

The Redis Enterprise Cloud Terraform provider is a plugin for Terraform that allows Redis Enterprise Cloud customers to manage the full 
lifecycle of their subscriptions and related Redis databases.

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 1.x
-	[Go](https://golang.org/doc/install) >= 1.19

Quick Starts
------------

- [Using the provider](https://registry.terraform.io/providers/RedisLabs/rediscloud/latest/docs)

To use the Redis Enterprise Cloud Terraform provider you will need to set the following environment variables, 
and these are created through the Redis Enterprise Cloud console under the settings menu.

- `REDISCLOUD_ACCESS_KEY` - Account Cloud API Access Key
- `REDISCLOUD_SECRET_KEY` - Individual user Cloud API Secret Key


Developing the Provider
-----------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).
You will also need to create or have access to a [Redis Cloud Enterprise](https://redislabs.com/redis-enterprise-cloud/overview) account.

Building the Provider
---------------------

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the `make build` command: 
```sh
$ make build
```

The `make build` command will build a local provider binary into a `bin` directory at the root of the repository.

Installing the Provider
-----------------------

After the provider has been built locally it must be placed in the user plugins directory so it can be discovered by the 
Terraform CLI.  The default user plugins directory root is `~/.terraform.d/plugins`.  

Use the following make command to install the provider locally.
```sh
$ make install_local
```

The provider will now be installed in the following location ready to be used by Terraform
```
~/.terraform.d/plugins
└── registry.terraform.io
    └── RedisLabs
        └── rediscloud
            └── 99.99.99
                └── <OS>_<ARCH>
                    └── terraform-provider-rediscloud_v99.99.99
```

The provider binary is built using a version number of `99.99.99` and this will allow Terraform to use the locally 
built provider over a released version.

The terraform provider is installed and can now be discovered by Terraform through the following HCL block.

```hcl-terraform
terraform {
  required_providers {
    rediscloud = {
      source = "RedisLabs/rediscloud"
    }
  }
  required_version = ">= 0.13"
}
``` 

The following is an example of using the `rediscloud_regions` data-source to discover a list of supported regions.  It can be 
used to verify that the provider is set up and installed correctly without incurring the cost of subscriptions and databases.

```hcl-terraform
data "rediscloud_regions" "example" {
}

output "all_regions" {
  value = data.rediscloud_regions.example.regions
}
```

Testing the Provider
--------------------

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```

In order to run an individual acceptance test, the '-run' flag can be used together with a regular expression. 
The following example uses a regular expression matching single test called 'TestAccResourceRedisCloudSubscription_createWithDatabase'.

```sh
$ make testacc TESTARGS='-run=TestAccResourceRedisCloudSubscription_createWithDatabase'
```

In order to run the tests with extra debugging context, prefix the make command with TF_LOG (see the [terraform documentation](https://www.terraform.io/docs/internals/debugging.html) for details).
```sh
$ TF_LOG=trace make testacc
```

By default, the tests run with a parallelism of 3. This can be reduced if some tests are failing due to network-related 
issues, or increased if possible, to reduce the running time of the tests. Prefix the make command with TEST_PARALLELISM, 
as in the following example, to configure this.
```sh
$ TEST_PARALLELISM=2 make testacc
```

A core set of Acceptance tests are executed through the build pipeline, (considered short tests).  
Functionality that requires additional setup or environment variables can be executed using the following flags.

| Flag        | Description                                       |
|-------------|---------------------------------------------------|
| `-tls`      | Allows execution of TLS based acceptance tests    |
| `-contract` | Allows execution of contract payment method tests |

Adding Dependencies
-------------------

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up-to-date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

Releasing the Provider
----------------------

The steps to release a provider are:
1. Decide what the next version number will be. As this provider tries to follow [semantic versioning](https://semver.org/), the best strategy would be to look at the previous release number and decide whether the `MAJOR`, `MINOR` or `PATCH` version should be incremented.
2. Create a new tag on your local copy of this Git repository in the format of `vMAJOR.MINOR.PATCH`, where `MAJOR.MINOR.PATCH` is the version number you settled on in the previous step.
3. Push the tag from your local copy to GitHub. This will trigger the [release GitHub Action workflow](.github/workflows/release.yml) that will create the release for you.
4. While you are waiting for GitHub to finish building the release, update the [CHANGELOG](./CHANGELOG.md) with what has been added, fixed and changed in this release.
5. Once the release workflow has finished, the Terraform Registry will eventually spot the new version and update [the registry page](https://registry.terraform.io/providers/RedisLabs/rediscloud/latest) - this may take a few minutes.
