Terraform Provider Redis Cloud
==================

- Website: [terraform.io](https://www.terraform.io)
- Tutorials: [learn.hashicorp.com](https://learn.hashicorp.com/terraform?track=getting-started#getting-started)
- Redis Cloud:  [redislabs.com/redis-enterprise-cloud](https://redislabs.com/redis-enterprise-cloud)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

The Redis Enterprise Cloud Terraform provider is a plugin for Terraform that allows RediD Enterprise Cloud Pro customers to manage the full 
lifecyle of their subscriptions and related Redis databases.

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.12.x
-	[Go](https://golang.org/doc/install) >= 1.12

Quick Starts
------------

- [Using the provider](https://www.terraform.io/docs/providers/RedisLabs/rediscloud/index.html)

To use the Redis Enterprise Cloud Terraform provider you will need to set the following environment variables, 
and these are created through the Redis Enterprise Cloud console under the settings menu.

- REDISCLOUD_ACCESS_KEY - Account Cloud API Access Key
- REDISCLOUD_SECRET_KEY - Individual user Cloud API Secret Key


Developing the Provider
-----------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).
You will also need to create or have access to a [Redis Cloud Enterprise](https://redislabs.com/redis-enterprise-cloud/overview) account.

Building the Provider
---------------------

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the `make build` command: 
```sh
$ make build
```

The `make build` command will build a local provider binary into a `bin` directory at the root of the repository.

Installing the Provider
-----------------------

After the provider has been built locally it most be placed the user plugins directory so it can be discovered by the 
Terraform CLI.  The default user plugins directory root is `~/.terraform.d/plugins`.  

Use the following make command to install the provider locally.
```sh
$ make install_local
```

The provider will now be installed that the following location ready to be used by Terraform
```
~/.terraform.d/plugins
└── registry.terraform.io
    └── RedisLabs
        └── rediscloud
            └── 99.99.99
                └── <OS>_<ARCH>
                    └── terraform-provider-rediscloud_v99.99.99
```

The provider binary is built using a version number of `99.99.99` and this will allow terraform to use the locally 
built provider over a released version.

The terraform provider is now installed and now can be discovered by Terraform through the following HCL block.

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

The following is an example of using the rediscloud_regions data-source to discover a list of supported regions.  It can be 
used to verify that the provider is setup and installed correctly without incurring the cost of subscriptions and databases.

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

Adding Dependencies
-------------------

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.
