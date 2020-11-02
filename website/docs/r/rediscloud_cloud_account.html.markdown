---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_cloud_account"
sidebar_current: "docs-rediscloud-cloud-account"
description: |-
  Cloud Account resource in the Terraform provider RedisCloud.
---

# rediscloud_cloud_account

Cloud Account resource in the Terraform provider RedisCloud.

## Example Usage

```hcl
resource "rediscloud_cloud_account" "example" {
  access_key_id     = "abcdefg"
  access_secret_key = "9876543"
  console_username  = "username"
  console_password  = "password"
  name              = "Example account"
  provider_type     = "AWS"
  sign_in_login_url = "https://1234567890.signin.aws.amazon.com/console"
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
Note that drift cannot currently be detected for this.

* `sign_in_login_url` - (Required) Cloud provider management console login URL.
Note that drift cannot currently be detected for this.

## Attribute Reference

`status` is set to the current status of the account - `draft`, `pending` or `active`.
