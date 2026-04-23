---
page_title: "Guide to using Customer Managed Keys (CMKs)"
---

# Customer Managed Keys

Customer managed keys (CMKs) are a feature in Redis Cloud that lets users encrypt the data within their databases remotely.
Redis Cloud supports two encryption options: using a Redis-managed encryption key or using a customer-managed key (CMK),
where customers supply and control the key themselves.

Because of how subscriptions operate, the Terraform flow for CMKs is slightly different to Terraform flows you may have
come across before. In this case, Terraform requires applying the resource twice. The Terraform workflow for enabling
CMKs differs from standard resource provisioning due to a dependency cycle between cloud providers and the Redis Cloud API.

The Terraform flow looks like this:
- First `terraform apply` to enable CMK and get Redis service account/role info.
- Grant the Redis identity (GCP service account or AWS IAM role) access to your CMK in your cloud provider.
- Second `terraform apply` to update the encryption information in Redis Cloud.

CMKs are provided at the subscription level, not at the database level. For a multi-region database (active-active)
you will need to provide a key for each region. These can be the same key, but they must be specified for each region.

Here are examples of subscriptions which use a CMK, for both GCP and AWS:

## GCP Pro Subscription

```hcl
resource "rediscloud_subscription" "example" {
  name                         = "..."
  payment_method               = "credit-card"
  payment_method_id            = "..."
  customer_managed_key_enabled = true

  cloud_provider {
    provider = "GCP"
    region {
      region                     = "europe-west2"
      networking_deployment_cidr = "..."
    }
  }

  # only one cmk required for pro subscriptions
  customer_managed_key {
    resource_name = "projects/.../locations/.../keyRings/.../cryptoKeys/..."
  }

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    support_oss_cluster_api = false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
}

output "customer_managed_key_redis_service_account" {
  value = rediscloud_subscription.example.customer_managed_key_redis_service_account
}
```

## AWS Pro Subscription

```hcl
resource "rediscloud_subscription" "example" {
  name                         = "..."
  payment_method               = "credit-card"
  payment_method_id            = "..."
  customer_managed_key_enabled = true

  cloud_provider {
    provider = "AWS"
    region {
      region                     = "us-east-1"
      networking_deployment_cidr = "..."
    }
  }

  # only one cmk required for pro subscriptions
  customer_managed_key {
    resource_name = "arn:aws:kms:us-east-1:123456789012:key/..."
  }

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    support_oss_cluster_api = false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
}

output "customer_managed_key_aws_role_arn" {
  value = rediscloud_subscription.example.customer_managed_key_aws_role_arn
}
```

## GCP Active-Active Subscription

```hcl
resource "rediscloud_active_active_subscription" "example" {
  name                         = "..."
  payment_method               = "credit-card"
  payment_method_id            = "..."
  customer_managed_key_enabled = true
  cloud_provider               = "GCP"

  # multiple keys required for active active subscriptions
  customer_managed_key {
    resource_name = "projects/.../locations/.../keyRings/.../cryptoKeys/..."
    region        = "europe-west1"
  }

  customer_managed_key {
    resource_name = "projects/.../locations/.../keyRings/.../cryptoKeys/..."
    region        = "europe-west2"
  }

  # creation plan is to show how the regions line up with the keys
  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "europe-west1"
      networking_deployment_cidr  = "..."
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "europe-west2"
      networking_deployment_cidr  = "..."
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

output "customer_managed_key_redis_service_account" {
  value = rediscloud_active_active_subscription.example.customer_managed_key_redis_service_account
}
```

## AWS Active-Active Subscription

```hcl
resource "rediscloud_active_active_subscription" "example" {
  name                         = "..."
  payment_method               = "credit-card"
  payment_method_id            = "..."
  customer_managed_key_enabled = true
  cloud_provider               = "AWS"

  # multiple keys required for active active subscriptions
  customer_managed_key {
    resource_name = "arn:aws:kms:us-east-1:123456789012:key/..."
    region        = "us-east-1"
  }

  customer_managed_key {
    resource_name = "arn:aws:kms:us-east-2:123456789012:key/..."
    region        = "us-east-2"
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "us-east-1"
      networking_deployment_cidr  = "..."
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "us-east-2"
      networking_deployment_cidr  = "..."
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

output "customer_managed_key_aws_role_arn" {
  value = rediscloud_active_active_subscription.example.customer_managed_key_aws_role_arn
}
```


## How to activate CMK for your subscription

### 1. Create your CMK(s) in your cloud provider
This can be done either using Terraform or manually creating one in the cloud provider console.

### 2. Activate the `customer_managed_key_enabled` flag

The setup is relatively similar between the two kinds of subscriptions, and is enabled by activation of the
`customer_managed_key_enabled` flag on the subscription.

Run `terraform apply` with this flag enabled on your subscription.

The subscription is put into a `encryption_key_pending` state - and will output identity information for the next step:
- **GCP**: `customer_managed_key_redis_service_account` — the GCP service account to grant access to your Cloud KMS key
- **AWS**: `customer_managed_key_aws_role_arn` — the IAM role ARN to grant access to your AWS KMS key

### 3. Grant the Redis identity permission to use your CMK(s)

Now your Redis identity (service account or IAM role) is created with your subscription. You need to grant CMK access to this identity.

You will need to grant the CMK permissions on the cloud provider externally. This can be done by either manually using
the cloud provider console, or you can use the Terraform provider.

The exact permissions depend on the provider:

**GCP** — grant the `customer_managed_key_redis_service_account` the following roles on your Cloud KMS key:

- Cloud KMS CryptoKey Encrypter/Decrypter
- Cloud KMS Viewer

**AWS** — update your KMS key policy to grant the `customer_managed_key_aws_role_arn` the following permissions:

- `kms:Encrypt`
- `kms:Decrypt`
- `kms:ReEncrypt*`
- `kms:GenerateDataKey*`
- `kms:DescribeKey`
- `kms:CreateGrant`
- `kms:ListGrants`
- `kms:RevokeGrant`

### 4. Update the subscription

Now that you have granted access to your CMK, the subscription can be transitioned fully into an active subscription state.

Simply `terraform apply` again.

Terraform will now update the subscription with the CMK information using the `customer_managed_key` blocks. Note that
if you have an active-active subscription with multiple regions, you must provide a block for each region. This is true
even if the CMK is the same.
