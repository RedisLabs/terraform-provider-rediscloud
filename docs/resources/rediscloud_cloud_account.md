---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_cloud_account"
description: |-
  Cloud Account resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_cloud_account

Creates a Cloud Account resource representing the access credentials to a cloud provider account, (`AWS`).
Redis Enterprise Cloud uses these credentials to provision databases within your infrastructure. 

## Example Usage

The following example defines a new AWS Cloud Account that is then used with a Subscription.

```hcl-terraform
resource "rediscloud_cloud_account" "example" {
  access_key_id     = "abcdefg"
  access_secret_key = "9876543"
  console_username  = "username"
  console_password  = "password"
  name              = "Example account"
  provider_type     = "AWS"
  sign_in_login_url = "https://1234567890.signin.aws.amazon.com/console"
}

resource "rediscloud_subscription" "example" {
  name                          = "My Example Subscription"
  payment_method_id             = data.rediscloud_payment_method.card.id
  memory_storage                = "ram"

  cloud_provider {
    provider         = data.rediscloud_cloud_account.example.provider_type
    cloud_account_id = data.rediscloud_cloud_account.example.id
    # ...
  }
  # ...
}

```

## Argument Reference

The following arguments are supported:

* `access_key_id` - (Required) Cloud provider access key.

* `access_secret_key` - (Required) Cloud provider secret key.
Note that drift cannot currently be detected for this.

* `console_username` - (Required) Cloud provider management console username.
Note that drift cannot currently be detected for this.

* `console_password` - (Required) Cloud provider management console password.
Note that drift cannot currently be detected for this.

* `name` - (Required) Display name of the account.

* `provider_type` - (Required) Cloud provider type - either `AWS` or `GCP`.
Note that drift cannot currently be detected for this. **Modifying this attribute will force creation of a new resource.**

* `sign_in_login_url` - (Required) Cloud provider management console login URL.
Note that drift cannot currently be detected for this.

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 5 mins) Used when creating the Cloud Account
* `update` - (Defaults to 5 mins) Used when updating the Cloud Account
* `delete` - (Defaults to 5 mins) Used when destroying the Cloud Account

## Attribute Reference

`status` is set to the current status of the account - `draft`, `pending` or `active`.

## Import

`rediscloud_cloud_account` can be imported using the ID of the Cloud Account, e.g.

```
$ terraform import rediscloud_cloud_account.example 12345678
```
