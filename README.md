Terraform Provider RedisCloud
==================

The RedisCloud Terraform provider is a plugin for Terraform that allows Redis Cloud Enterprise customers to manage the full 
lifecyle of their enterprise subscriptions and related redis databases.

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.12.x
-	[Go](https://golang.org/doc/install) >= 1.12

Building The Provider
---------------------

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `make install_macos` command: 
```sh
$ make install_macos
```

Adding Dependencies
---------------------

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.


Using the provider
----------------------

The RedisCloud Terraform provider is distributed through the HashiCorp managed [Terraform Registry](https://registry.terraform.io) 
and is discovered through the use of a `required_providers`hcl block as shown in the following example.

```hcl-terraform
terraform {
  required_providers {
    rediscloud = {
      source = "registry.redislabs.com/redislabs/rediscloud"
    }
  }
  required_version = ">= 0.13"
}
```

After the `required_providers` block has been configured Terraform will download the latest version of the RedisCloud provider.  
An optional and recommended `version` parameter can be applied to use a specific version of the provider.

To use the RedisCloud Terraform provider you will need to set the following environment variables 
and these are created through the Redis Cloud console under the settings menu.

* REDISCLOUD_ACCESS_KEY - Account Cloud API Access Key
* REDISCLOUD_SECRET_KEY - Individual user Cloud API Secret Key

After this initial setup the provider can be used against your account.  The following example can be used to validate 
your account access through the provider and will return a list of regions without incurring a cost.  

```hcl-terraform
data "rediscloud_regions" "example" {
}

output "all_regions" {
  value = data.rediscloud_regions.example.regions
}
```

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).
You will also need to create or have access to a [Redis Cloud Enterprise](https://redislabs.com/redis-enterprise-cloud/overview) account.

To compile the provider, run `make install_macos`. This will build the provider and place the binary 
in the `~/Library/Application\ Support/io.terraform/plugins` directory.

The provider binary is installed with a version number of `99.99.99` and this will allow terraform to use the locally 
built provider over a released version.

Next configure a `required_versions` hcl block to reference the provider, (see [Using the provider](#using-the-provider)) and set the appropriate 
access and secret keys linked to your RedisCloud Enterprise account.  The provider can now be used through the Terraform CLI.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
