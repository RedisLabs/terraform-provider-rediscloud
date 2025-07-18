# Terraform Provider RedisCloud Development Guide

## Table of Contents
1. [Project Overview](#project-overview)
2. [Repository Access Methods](#repository-access-methods)
3. [Architecture Analysis](#architecture-analysis)
4. [Available Controllers](#available-controllers)
5. [Step-by-Step Implementation Guide](#step-by-step-implementation-guide)
6. [Documentation Standards](#documentation-standards)
7. [Concrete Examples](#concrete-examples)
8. [Best Practices and Considerations](#best-practices-and-considerations)

## Project Overview

The `terraform-provider-rediscloud` is a Terraform provider that enables infrastructure-as-code management of Redis Enterprise Cloud resources. It acts as a bridge between Terraform configurations and the Redis Cloud API.

### Key Relationships

- **terraform-provider-rediscloud**: The Terraform provider repository
- **sm-cloud-api**: The backend REST API service (Java Spring Boot application)
- **rediscloud-go-api**: Go client library that wraps the sm-cloud-api REST endpoints

```
Terraform Config → terraform-provider-rediscloud → rediscloud-go-api → sm-cloud-api → Redis Cloud Infrastructure
```

### API Documentation
- Swagger UI: https://api.redislabs.com/v1/swagger-ui/index.html
- API Docs: https://api.redislabs.com/v1/cloud-api-docs

## Repository Access Methods

### Local Access
The sm-cloud-api repository is cloned locally at:
```
~/dev/git/sm-cloud-api/
```

### GitHub API Access
Repository: `redislabsdev/sm-cloud-api`
- **URL**: https://github.com/redislabsdev/sm-cloud-api
- **Access**: Via GitHub API for browsing structure and files
- **Branch**: `develop` (default)

### Directory Structure
```
sm-cloud-api/
├── sm-redislabs-api/
│   └── src/main/java/com/redislabs/api/rest/controllers/
│       ├── SubscriptionsController.java
│       ├── DatabasesController.java
│       ├── UsersController.java
│       └── [other controllers...]
└── [other modules...]
```

## Architecture Analysis

### Current Project Structure

```
terraform-provider-rediscloud/
├── provider/
│   ├── provider.go                           # Main provider registration
│   ├── resource_rediscloud_pro_subscription.go  # Maps to SubscriptionsController
│   ├── resource_rediscloud_pro_database.go      # Maps to DatabasesController
│   ├── resource_rediscloud_acl_user.go          # Maps to ACLController
│   ├── datasource_rediscloud_*.go               # Data sources
│   └── [other resources...]
├── go.mod                                    # Dependencies including rediscloud-go-api v0.22.0
└── [other files...]
```

### Resource to Controller Mapping

| Terraform Resource | sm-cloud-api Controller | Go API Service |
|-------------------|------------------------|----------------|
| `resource_rediscloud_pro_subscription.go` | `SubscriptionsController.java` | `service/subscriptions` |
| `resource_rediscloud_pro_database.go` | `DatabasesController.java` | `service/databases` |
| `resource_rediscloud_acl_user.go` | `ACLController.java` | `service/access_control_lists/users` |
| `resource_rediscloud_cloud_account.go` | `CloudAccountsController.java` | `service/cloud_accounts` |

### Provider Registration Pattern

In `provider/provider.go`:
```go
ResourcesMap: map[string]*schema.Resource{
    "rediscloud_subscription":              resourceRedisCloudProSubscription(),
    "rediscloud_subscription_database":     resourceRedisCloudProDatabase(),
    "rediscloud_acl_user":                 resourceRedisCloudAclUser(),
    // ... other resources
},
```

### Import Structure Pattern

```go
import (
    "github.com/RedisLabs/rediscloud-go-api/redis"
    "github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
    "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)
```

## Available Controllers

Based on analysis of sm-cloud-api, the following controllers are available for potential terraform resources:

### Already Implemented
- ✅ `SubscriptionsController.java` → `rediscloud_subscription`, `rediscloud_active_active_subscription`
- ✅ `DatabasesController.java` → `rediscloud_subscription_database`, `rediscloud_active_active_database`
- ✅ `ACLController.java` → `rediscloud_acl_user`, `rediscloud_acl_role`, `rediscloud_acl_rule`
- ✅ `CloudAccountsController.java` → `rediscloud_cloud_account`
- ✅ `FixedSubscriptionsController.java` → `rediscloud_essentials_subscription`
- ✅ `FixedDatabasesController.java` → `rediscloud_essentials_database`
- ✅ `ModuleController.java` → `rediscloud_database_modules` (data source)
- ✅ `DataPersistenceController.java` → `rediscloud_data_persistence` (data source)
- ✅ `PlanController.java` → `rediscloud_essentials_plan` (data source)
- ✅ `SubscriptionsConnectivityController.java` → `rediscloud_private_service_connect`, `rediscloud_transit_gateway_attachment`, `rediscloud_active_active_private_service_connect`, `rediscloud_active_active_transit_gateway_attachment`
- ✅ `RegionController.java` → `rediscloud_regions` (data source)
- ✅ `AccountController.java` → `rediscloud_payment_method` (data source)

### Available for Implementation
- 🔄 `UsersController.java` → `rediscloud_user` (user management)
- 🔄 `DedicatedInstancesController.java` → `rediscloud_dedicated_instance`
- 🔄 `DedicatedSubscriptionsController.java` → `rediscloud_dedicated_subscription`

### Used Internally (Not for Direct Resource Implementation)
- 🔧 `TasksController.java` - Used internally for asynchronous operation polling and status tracking
- 🔧 `MetricsController.java` - Internal metrics collection (not user-configurable)
- 🔧 `MonitoringController.java` - Internal monitoring services (system monitoring)
- 🔧 `SearchScalingFactorController.java` - Internal search scaling configuration (system optimization)

### Helper/Utility Controllers
- `BaseController.java` - Base functionality
- `ControllerHelper.java` - Helper utilities
- `DatabaseControllerHelper.java` - Database-specific helpers
- `ControllerHateoasLinksHelper.java` - HATEOAS link generation utilities
- `FixedProviderBinder.java` - Fixed subscription provider binding utilities
- `ProviderBinder.java` - General provider binding utilities

## Asynchronous Operations and TasksController Usage

### Understanding TasksController

The `TasksController.java` is **NOT** intended to be exposed as a direct terraform resource (`rediscloud_task`). Instead, it serves as an internal mechanism for handling asynchronous operations in the Redis Cloud API.

### How Asynchronous Operations Work

When the terraform provider performs long-running operations (like creating subscriptions or databases), the flow typically follows this pattern:

1. **API Call**: Provider calls a resource endpoint (e.g., `POST /subscriptions`)
2. **Task Creation**: The API returns a task ID for the asynchronous operation
3. **Internal Polling**: The provider internally polls the TasksController to check operation status
4. **Status Monitoring**: Provider waits for the task to complete before proceeding
5. **Resource Ready**: Once the task completes, the resource is considered ready

### Current Implementation Pattern

The terraform provider uses **direct resource status polling** rather than task-based polling. Here's how it works:

#### Example: Subscription Creation Flow

```go
// From resource_rediscloud_pro_subscription.go
func resourceRedisCloudProSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    // 1. Create the subscription
    subId, err := api.client.Subscription.Create(ctx, createSubscription)
    if err != nil {
        return diag.FromErr(err)
    }

    d.SetId(strconv.Itoa(subId))

    // 2. Wait for subscription to become active (internal polling)
    err = waitForSubscriptionToBeActive(ctx, subId, api)
    if err != nil {
        return append(diags, diag.FromErr(err)...)
    }

    // 3. Continue with additional setup...
}

// Internal wait function using resource status polling
func waitForSubscriptionToBeActive(ctx context.Context, id int, api *apiClient) error {
    wait := &retry.StateChangeConf{
        Pending:      []string{subscriptions.SubscriptionStatusPending},
        Target:       []string{subscriptions.SubscriptionStatusActive},
        Timeout:      safetyTimeout,
        Delay:        10 * time.Second,
        PollInterval: 30 * time.Second,
        Refresh: func() (interface{}, string, error) {
            // Poll the resource directly, not the TasksController
            subscription, err := api.client.Subscription.Get(ctx, id)
            if err != nil {
                return nil, "", err
            }
            return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
        },
    }
    if _, err := wait.WaitForStateContext(ctx); err != nil {
        return err
    }
    return nil
}
```

#### Example: Database Creation Flow

```go
// From resource_rediscloud_pro_database.go
func resourceRedisCloudProDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    // 1. Ensure subscription is ready
    if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
        return diag.FromErr(err)
    }

    // 2. Create the database
    dbId, err := api.client.Database.Create(ctx, subId, createDatabase)
    if err != nil {
        return diag.FromErr(err)
    }

    // 3. Wait for both database and subscription to be active
    if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
        return diag.FromErr(err)
    }
    if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
        return diag.FromErr(err)
    }

    // 4. Continue with configuration...
}
```

### Why TasksController is Not a Direct Resource

1. **Implementation Detail**: TasksController is an internal API mechanism, not a user-facing resource
2. **Abstraction Level**: Terraform users care about resources (subscriptions, databases), not the tasks that create them
3. **Lifecycle Management**: Tasks are ephemeral - they exist only during operations and are not managed long-term
4. **Provider Responsibility**: The provider handles task polling internally to provide a clean, synchronous interface to users

### Alternative Async Patterns

Some APIs use task-based polling, but the Redis Cloud terraform provider uses **resource status polling**:

```go
// Task-based pattern (NOT used in Redis Cloud provider)
taskId, err := api.client.Resource.CreateAsync(ctx, request)
err = waitForTaskToComplete(ctx, taskId, api) // Poll TasksController

// Resource status pattern (USED in Redis Cloud provider)
resourceId, err := api.client.Resource.Create(ctx, request)
err = waitForResourceToBeActive(ctx, resourceId, api) // Poll resource directly
```

## Step-by-Step Implementation Guide

### Step 1: Research the API Controller

1. **Examine the Controller**: Check the sm-cloud-api controller to understand available endpoints
2. **Review API Documentation**: Check the OpenAPI spec at https://api.redislabs.com/v1/cloud-api-docs
3. **Verify Go Client Support**: Ensure `rediscloud-go-api` has the corresponding service

### Step 2: Create the Resource File

Create `provider/resource_rediscloud_[resource_name].go`:

```go
package provider

import (
    "context"
    "strconv"
    "time"

    "github.com/RedisLabs/rediscloud-go-api/redis"
    "github.com/RedisLabs/rediscloud-go-api/service/[service_name]"
    "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRedisCloud[ResourceName]() *schema.Resource {
    return &schema.Resource{
        Description:   "Manages a [resource] within your Redis Enterprise Cloud Account.",
        CreateContext: resourceRedisCloud[ResourceName]Create,
        ReadContext:   resourceRedisCloud[ResourceName]Read,
        UpdateContext: resourceRedisCloud[ResourceName]Update,
        DeleteContext: resourceRedisCloud[ResourceName]Delete,

        Importer: &schema.ResourceImporter{
            StateContext: schema.ImportStatePassthroughContext,
        },

        Timeouts: &schema.ResourceTimeout{
            Create: schema.DefaultTimeout(5 * time.Minute),
            Read:   schema.DefaultTimeout(3 * time.Minute),
            Update: schema.DefaultTimeout(5 * time.Minute),
            Delete: schema.DefaultTimeout(5 * time.Minute),
        },

        Schema: map[string]*schema.Schema{
            // Define schema fields based on API model
        },
    }
}
```

### Step 3: Implement CRUD Operations

```go
func resourceRedisCloud[ResourceName]Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    api := meta.(*apiClient)
    
    // Extract values from schema
    // Create API request object
    // Call API client
    // Set resource ID
    // Return read operation
}

func resourceRedisCloud[ResourceName]Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    api := meta.(*apiClient)
    var diags diag.Diagnostics
    
    // Get resource ID
    // Call API to fetch resource
    // Handle not found case
    // Set schema values from API response
    
    return diags
}

// Implement Update and Delete similarly
```

### Step 4: Register the Resource

Add to `provider/provider.go`:
```go
ResourcesMap: map[string]*schema.Resource{
    // ... existing resources ...
    "rediscloud_[resource_name]": resourceRedisCloud[ResourceName](),
},
```

### Step 5: Create Data Source (Optional)

Create `provider/datasource_rediscloud_[resource_name].go` for read-only access.

### Step 6: Create Documentation

Create documentation files following the established patterns:
- `docs/resources/rediscloud_[resource_name].md` for resource documentation
- `docs/data-sources/rediscloud_[resource_name].md` for data source documentation (if applicable)

### Step 7: Add Tests

Create `provider/resource_rediscloud_[resource_name]_test.go` with acceptance tests.

## Documentation Standards

### Terraform Registry Integration

The terraform-provider-rediscloud documentation is published to the **Terraform Registry** at:
- **Main Provider Page**: https://registry.terraform.io/providers/RedisLabs/rediscloud/latest
- **Resource Documentation**: https://registry.terraform.io/providers/RedisLabs/rediscloud/latest/docs/resources/[resource_name]
- **Data Source Documentation**: https://registry.terraform.io/providers/RedisLabs/rediscloud/latest/docs/data-sources/[data_source_name]

### Documentation Structure

The provider follows the standard Terraform documentation structure:

```
docs/
├── index.md                    # Provider overview and configuration
├── guides/                     # Migration guides and tutorials
│   ├── migration-guide-v1.0.0.md
│   └── migration-guide-v2.0.0.md
├── resources/                  # Resource documentation
│   ├── rediscloud_subscription.md
│   ├── rediscloud_acl_user.md
│   └── [other resources...]
└── data-sources/              # Data source documentation
    ├── rediscloud_subscription.md
    ├── rediscloud_acl_user.md
    └── [other data sources...]
```

### Documentation Format Standards

#### Resource Documentation Template

Each resource documentation file follows this structure:

```markdown
---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_[resource_name]"
description: |-
  [Resource description] resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_[resource_name]

[Detailed description of what the resource manages]

[Optional: Notes about specific behaviors, limitations, or requirements]

## Example Usage

```hcl
resource "rediscloud_[resource_name]" "example" {
  # Configuration example
}
```

## Argument Reference

The following arguments are supported:

* `argument_name` - (Required/Optional) Description of the argument
* `another_argument` - (Required/Optional, change forces recreation) Description

[For nested blocks:]
The `block_name` block supports:

* `nested_field` - (Required/Optional) Description

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to X mins) Used when creating the resource
* `update` - (Defaults to X mins) Used when updating the resource
* `delete` - (Defaults to X mins) Used when destroying the resource

## Attribute Reference

[List of computed attributes]

## Import

`rediscloud_[resource_name]` can be imported using the ID, e.g.

```
$ terraform import rediscloud_[resource_name].example 12345
```

[Optional: Notes about import behavior]
```

#### Data Source Documentation Template

```markdown
---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_[data_source_name]"
description: |-
  [Data source description] data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_[data_source_name]

[Description of what data the source provides]

## Example Usage

```hcl
data "rediscloud_[data_source_name]" "example" {
  # Filter criteria
}

output "result" {
  value = data.rediscloud_[data_source_name].example.id
}
```

## Argument Reference

* `filter_field` - (Optional) Description of filter

## Attributes Reference

* `id` - The ID of the found resource
* `attribute_name` - Description of attribute
```

### Documentation Best Practices

#### Content Guidelines

1. **Clear Descriptions**: Use concise, clear language that explains what the resource does
2. **Argument Documentation**:
   - Mark as `(Required)` or `(Optional)`
   - Add `(change forces recreation)` for ForceNew fields
   - Include validation constraints where applicable
3. **Examples**: Provide realistic, working examples
4. **Notes and Warnings**: Use appropriate callouts:
   - `-> **Note:**` for important information
   - `~> **Note:**` for warnings or caveats

#### Formatting Standards

1. **Code Blocks**: Use `hcl` syntax highlighting for Terraform code
2. **Nested Blocks**: Document nested schema blocks clearly
3. **Import Syntax**: Always include import examples
4. **Cross-References**: Link to related resources and data sources

### Documentation Generation Workflow

The documentation is **manually maintained** in the `docs/` folder and automatically published to the Terraform Registry when releases are created. The workflow is:

1. **Manual Documentation**: Write documentation in `docs/` following the templates
2. **Local Testing**: Test documentation locally during development
3. **Release Process**: Documentation is published to the Registry during the release workflow
4. **Registry Update**: The Terraform Registry automatically pulls documentation from the latest release

### Cross-Reference with Implementation

When implementing new resources, ensure documentation consistency:

#### Resource Schema → Documentation Mapping

```go
// In resource implementation
Schema: map[string]*schema.Schema{
    "name": {
        Description: "A meaningful name for the resource",
        Type:        schema.TypeString,
        Required:    true,
        ForceNew:    true,
    },
}
```

```markdown
<!-- In documentation -->
* `name` - (Required, change forces recreation) A meaningful name for the resource
```

#### Validation → Documentation

```go
// In resource implementation
ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"option1", "option2"}, false)),
```

```markdown
<!-- In documentation -->
* `field_name` - (Required) Description. Valid values are `option1` and `option2`.
```

### Documentation Examples from Existing Resources

#### Example: rediscloud_subscription Documentation Structure

The `rediscloud_subscription` resource documentation demonstrates the complete pattern:

```markdown
---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription"
description: |-
  Subscription resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_subscription

This resource allows you to manage a subscription within your Redis Enterprise Cloud account.

-> **Note:** This is for Pro Subscriptions only. See also `rediscloud_active_active_subscription` and `rediscloud_essentials_subscription`.

~> **Note:** The payment_method property is ignored after Subscription creation.
```

Key elements:
- **Frontmatter**: YAML metadata for the documentation system
- **Clear Title**: Follows the pattern "Redis Cloud: rediscloud_[resource_name]"
- **Description**: Brief explanation of the resource purpose
- **Notes**: Important behavioral information using appropriate callouts

#### Example: Complex Nested Block Documentation

From `rediscloud_subscription.md`, showing how to document nested blocks:

```markdown
The `cloud_provider` block supports:

* `provider` - (Optional) The cloud provider to use with the subscription, (either `AWS` or `GCP`). Default: 'AWS'. **Modifying this attribute will force creation of a new resource.**
* `cloud_account_id` - (Optional) Cloud account identifier. Default: Redis Labs internal cloud account.
* `region` - (Required) A region object, documented below.

The cloud_provider `region` block supports:

* `region` - (Required) Deployment region as defined by cloud provider.
* `multiple_availability_zones` - (Optional) Support deployment on multiple availability zones within the selected region. Default: 'false'.
* `networking_deployment_cidr` - (Required) Deployment CIDR mask. The total number of bits must be 24 (x.x.x.x/24).
```

#### Example: Simple Resource Documentation

From `rediscloud_acl_user.md`, showing a simpler resource pattern:

```markdown
# Resource: rediscloud_acl_user

Creates a User in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
resource "rediscloud_acl_user" "user-resource" {
  name     = "my-user"
  role     = rediscloud_acl_role.role-resource.name
  password = "mY.passw0rd"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, change forces recreation) A meaningful name for the User. Must be unique.
* `role` - (Required) The name of the Role held by the User.
* `password` - (Required, change forces recreation) The password for this ACL User.
```

#### Example: Data Source Documentation

From `rediscloud_subscription` data source:

```markdown
# Data Source: rediscloud_subscription

This data source allows access to the details of an existing Subscription within your Redis Enterprise Cloud account.

## Example Usage

```hcl
data "rediscloud_subscription" "example" {
  name = "My Example Subscription"
}
output "rediscloud_subscription" {
  value = data.rediscloud_subscription.example.id
}
```

## Argument Reference

* `name` - (Optional) The name of the subscription to filter returned subscriptions

## Attributes Reference

`id` is set to the ID of the found subscription.

* `payment_method_id` - A valid payment method pre-defined in the current account
* `memory_storage` - Memory storage preference: either 'ram' or a combination of 'ram-and-flash'
```

### Documentation Workflow Integration

#### Step-by-Step Documentation Process

1. **Create Resource Documentation**:
   ```bash
   # Create the resource documentation file
   touch docs/resources/rediscloud_[resource_name].md
   ```

2. **Follow the Template**: Use the established template and examples above

3. **Cross-Reference Implementation**: Ensure documentation matches the actual schema implementation

4. **Test Locally**: Review documentation for clarity and completeness

5. **Include in PR**: Documentation changes should be included in the same PR as the resource implementation

#### Documentation Validation

Before submitting documentation:

1. **Schema Consistency**: Verify all schema fields are documented
2. **Example Validity**: Ensure HCL examples are syntactically correct
3. **Import Instructions**: Test import syntax is accurate
4. **Cross-References**: Verify links to related resources work
5. **Formatting**: Check markdown formatting renders correctly

## Concrete Examples

### Example: Implementing rediscloud_user Resource

```go
// provider/resource_rediscloud_user.go
package provider

import (
    "context"
    "strconv"
    "time"

    "github.com/RedisLabs/rediscloud-go-api/redis"
    "github.com/RedisLabs/rediscloud-go-api/service/users"
    "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRedisCloudUser() *schema.Resource {
    return &schema.Resource{
        Description:   "Manages a user within your Redis Enterprise Cloud Account.",
        CreateContext: resourceRedisCloudUserCreate,
        ReadContext:   resourceRedisCloudUserRead,
        UpdateContext: resourceRedisCloudUserUpdate,
        DeleteContext: resourceRedisCloudUserDelete,

        Importer: &schema.ResourceImporter{
            StateContext: schema.ImportStatePassthroughContext,
        },

        Schema: map[string]*schema.Schema{
            "name": {
                Description: "User's full name",
                Type:        schema.TypeString,
                Required:    true,
            },
            "email": {
                Description: "User's email address",
                Type:        schema.TypeString,
                Required:    true,
                ForceNew:    true,
            },
            "role": {
                Description: "User's role in the account",
                Type:        schema.TypeString,
                Required:    true,
            },
        },
    }
}

func resourceRedisCloudUserCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    api := meta.(*apiClient)

    createUser := users.CreateUserRequest{
        Name:  redis.String(d.Get("name").(string)),
        Email: redis.String(d.Get("email").(string)),
        Role:  redis.String(d.Get("role").(string)),
    }

    id, err := api.client.Users.Create(ctx, createUser)
    if err != nil {
        return diag.FromErr(err)
    }

    d.SetId(strconv.Itoa(id))
    return resourceRedisCloudUserRead(ctx, d, meta)
}
```

### Example: Complete rediscloud_user Documentation

Create `docs/resources/rediscloud_user.md`:

```markdown
---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_user"
description: |-
  User resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_user

Manages a user within your Redis Enterprise Cloud Account. Users can be assigned roles and permissions to access specific resources.

-> **Note:** User management requires appropriate permissions in your Redis Enterprise Cloud account.

## Example Usage

```hcl
resource "rediscloud_user" "example" {
  name  = "john.doe"
  email = "john.doe@example.com"
  role  = "admin"
}

# Reference in other resources
resource "rediscloud_acl_role" "example" {
  name = "custom-role"
  # ... other configuration
}

resource "rediscloud_user" "custom_role_user" {
  name  = "jane.smith"
  email = "jane.smith@example.com"
  role  = rediscloud_acl_role.example.name
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) User's full name. Must be unique within the account.
* `email` - (Required, change forces recreation) User's email address. Must be a valid email format.
* `role` - (Required) User's role in the account. Valid values depend on your account configuration.

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 5 mins) Used when creating the user
* `update` - (Defaults to 5 mins) Used when updating the user
* `delete` - (Defaults to 5 mins) Used when destroying the user

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the user
* `status` - Current status of the user account
* `created_at` - Timestamp when the user was created
* `last_login` - Timestamp of the user's last login (if available)

## Import

`rediscloud_user` can be imported using the user ID, e.g.

```
$ terraform import rediscloud_user.example 12345
```

~> **Note:** When importing users, the role assignment will be read from the current state in Redis Enterprise Cloud.
```

### Example: Corresponding Data Source Documentation

Create `docs/data-sources/rediscloud_user.md`:

```markdown
---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_user"
description: |-
  User data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_user

Use this data source to access information about an existing user in your Redis Enterprise Cloud account.

## Example Usage

```hcl
data "rediscloud_user" "example" {
  email = "john.doe@example.com"
}

output "user_id" {
  value = data.rediscloud_user.example.id
}

# Use in other resources
resource "rediscloud_acl_role" "example" {
  name = "role-for-${data.rediscloud_user.example.name}"
  # ... other configuration
}
```

## Argument Reference

The following arguments are supported:

* `email` - (Optional) Filter users by email address
* `name` - (Optional) Filter users by name

-> **Note:** At least one of `email` or `name` must be specified.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the user
* `role` - The user's current role
* `status` - Current status of the user account
* `created_at` - Timestamp when the user was created
* `last_login` - Timestamp of the user's last login (if available)
```

## Best Practices and Considerations

### Naming Conventions
- **Resources**: `rediscloud_[resource_type]` (e.g., `rediscloud_user`)
- **Files**: `resource_rediscloud_[resource_type].go`
- **Functions**: `resourceRedisCloud[ResourceType][Operation]`

### Schema Design
- Use appropriate Terraform types (`TypeString`, `TypeInt`, `TypeBool`, `TypeList`, `TypeSet`)
- Add clear descriptions for all fields
- Use `ForceNew: true` for immutable fields
- Mark sensitive fields with `Sensitive: true`
- Add validation where appropriate

### Error Handling
- Handle API-specific errors (NotFound, Conflict, etc.)
- Use `diag.FromErr()` for error conversion
- Implement proper resource cleanup on creation failures

### State Management
- Always implement import functionality
- Handle resource drift detection
- Use computed fields for API-generated values

### Testing
- Write comprehensive acceptance tests
- Test all CRUD operations
- Test import functionality
- Test error scenarios

### Performance
- Set appropriate timeouts for long-running operations
- Implement retry logic for transient failures
- Use efficient API calls (avoid unnecessary requests)

### Documentation
- Add clear resource descriptions
- Document all schema fields
- Provide usage examples
- Document any limitations or special considerations

### Code Organization
- Follow existing patterns in the codebase
- Keep functions focused and single-purpose
- Use helper functions for complex logic
- Maintain consistent error handling patterns

## Advanced Implementation Patterns

### Complex Schema Structures

For resources with nested objects (like the subscription's cloud_provider block):

```go
Schema: map[string]*schema.Schema{
    "cloud_provider": {
        Description: "A cloud provider object",
        Type:        schema.TypeList,
        Required:    true,
        ForceNew:    true,
        MaxItems:    1,
        MinItems:    1,
        Elem: &schema.Resource{
            Schema: map[string]*schema.Schema{
                "provider": {
                    Description: "The cloud provider to use",
                    Type:        schema.TypeString,
                    Optional:    true,
                    Default:     "AWS",
                },
                "region": {
                    Description: "Cloud networking details",
                    Type:        schema.TypeSet,
                    Required:    true,
                    Elem: &schema.Resource{
                        Schema: map[string]*schema.Schema{
                            "region": {
                                Description: "Deployment region",
                                Type:        schema.TypeString,
                                Required:    true,
                            },
                            // ... more nested fields
                        },
                    },
                },
            },
        },
    },
}
```

### Wait Functions for Async Operations

Many Redis Cloud operations are asynchronous. Follow the established pattern of **resource status polling** rather than task-based polling:

#### Standard Wait Function Pattern

```go
func waitForResourceToBeActive(ctx context.Context, id int, api *apiClient) error {
    wait := &retry.StateChangeConf{
        Pending:      []string{"pending", "creating", "updating"},
        Target:       []string{"active"},
        Timeout:      10 * time.Minute,
        Delay:        10 * time.Second,
        PollInterval: 30 * time.Second,
        Refresh: func() (interface{}, string, error) {
            // Poll the resource directly, NOT the TasksController
            resource, err := api.client.Resource.Get(ctx, id)
            if err != nil {
                return nil, "", err
            }
            return resource, *resource.Status, nil
        },
    }
    _, err := wait.WaitForStateContext(ctx)
    return err
}
```

#### Real Example: Database Wait Function

```go
// From resource_rediscloud_pro_subscription.go
func waitForDatabaseToBeActive(ctx context.Context, subId, id int, api *apiClient) error {
    wait := &retry.StateChangeConf{
        Pending: []string{
            databases.StatusDraft,
            databases.StatusPending,
            databases.StatusActiveChangePending,
            databases.StatusRCPActiveChangeDraft,
        },
        Target:       []string{databases.StatusActive},
        Timeout:      safetyTimeout,
        Delay:        10 * time.Second,
        PollInterval: 30 * time.Second,
        Refresh: func() (interface{}, string, error) {
            database, err := api.client.Database.Get(ctx, subId, id)
            if err != nil {
                return nil, "", err
            }
            return redis.StringValue(database.Status), redis.StringValue(database.Status), nil
        },
    }
    if _, err := wait.WaitForStateContext(ctx); err != nil {
        return err
    }
    return nil
}
```

#### Best Practices for Async Operations

1. **Use Resource Status Polling**: Poll the resource's status field, not the TasksController
2. **Define Clear States**: Identify all possible pending and target states for your resource
3. **Set Appropriate Timeouts**: Use reasonable timeouts based on expected operation duration
4. **Handle Dependencies**: Wait for dependent resources to be ready before proceeding
5. **Error Handling**: Properly handle API errors and timeouts in the refresh function

#### Common Async Operation Patterns

```go
// Pattern 1: Simple resource creation
func resourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    // Create resource
    id, err := api.client.Resource.Create(ctx, request)
    if err != nil {
        return diag.FromErr(err)
    }

    d.SetId(strconv.Itoa(id))

    // Wait for resource to be ready
    if err := waitForResourceToBeActive(ctx, id, api); err != nil {
        return diag.FromErr(err)
    }

    return resourceRead(ctx, d, meta)
}

// Pattern 2: Resource with dependencies
func resourceCreateWithDependencies(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    // Ensure parent resource is ready
    if err := waitForParentToBeActive(ctx, parentId, api); err != nil {
        return diag.FromErr(err)
    }

    // Create child resource
    id, err := api.client.ChildResource.Create(ctx, parentId, request)
    if err != nil {
        return diag.FromErr(err)
    }

    // Wait for both child and parent to be ready
    if err := waitForChildToBeActive(ctx, parentId, id, api); err != nil {
        return diag.FromErr(err)
    }
    if err := waitForParentToBeActive(ctx, parentId, api); err != nil {
        return diag.FromErr(err)
    }

    return resourceRead(ctx, d, meta)
}
```

### Validation Functions

Add custom validation for complex fields:

```go
"memory_limit": {
    Description: "Memory limit in GB",
    Type:        schema.TypeFloat,
    Required:    true,
    ValidateFunc: validation.FloatBetween(0.1, 1000.0),
},
"cidr_block": {
    Description: "CIDR block for networking",
    Type:        schema.TypeString,
    Required:    true,
    ValidateFunc: validation.IsCIDR,
},
```

## Testing Patterns

### Acceptance Test Structure

```go
// provider/resource_rediscloud_user_test.go
func TestAccResourceRedisCloudUser_basic(t *testing.T) {
    name := acctest.RandomWithPrefix("test-user")
    email := fmt.Sprintf("%s@example.com", name)

    resource.Test(t, resource.TestCase{
        PreCheck:          func() { testAccPreCheck(t) },
        ProviderFactories: providerFactories,
        CheckDestroy:      testAccCheckRedisCloudUserDestroy,
        Steps: []resource.TestStep{
            {
                Config: testAccResourceRedisCloudUserConfig(name, email, "admin"),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("rediscloud_user.test", "name", name),
                    resource.TestCheckResourceAttr("rediscloud_user.test", "email", email),
                    resource.TestCheckResourceAttr("rediscloud_user.test", "role", "admin"),
                ),
            },
        },
    })
}

func testAccResourceRedisCloudUserConfig(name, email, role string) string {
    return fmt.Sprintf(`
resource "rediscloud_user" "test" {
    name  = "%s"
    email = "%s"
    role  = "%s"
}
`, name, email, role)
}
```

## Troubleshooting Common Issues

### API Client Errors
- **NotFound Errors**: Handle gracefully in Read operations by clearing the resource ID
- **Timeout Errors**: Increase timeout values for long-running operations
- **Rate Limiting**: Implement retry logic with exponential backoff

### Schema Issues
- **Type Mismatches**: Ensure Go types match Terraform schema types
- **Validation Errors**: Add appropriate validation functions
- **State Drift**: Implement proper diff suppression for computed fields

### Import Issues
- **ID Format**: Ensure import ID format matches resource expectations
- **Missing Fields**: Handle cases where imported resources have different field sets

## Migration and Versioning

### Breaking Changes
When making breaking changes to existing resources:
1. Increment the provider version
2. Document migration steps
3. Consider deprecation warnings before removal
4. Provide migration guides

### Backward Compatibility
- Use `DiffSuppressFunc` for fields that shouldn't trigger updates
- Add new optional fields rather than changing existing required fields
- Maintain support for older API versions when possible

## Security Considerations

### Sensitive Data
- Mark passwords and API keys as `Sensitive: true`
- Avoid logging sensitive information
- Use secure defaults where possible

### API Authentication
- Leverage the existing API client authentication
- Don't store credentials in resource state
- Use environment variables for configuration

## Performance Optimization

### Efficient API Usage
- Batch operations where possible
- Use pagination for large result sets
- Implement caching for frequently accessed data
- Minimize API calls in Read operations

### Resource Dependencies
- Use `depends_on` appropriately
- Implement proper resource ordering
- Handle circular dependencies

## Monitoring and Observability

### Logging
- Use structured logging with appropriate levels
- Log API requests and responses (excluding sensitive data)
- Include correlation IDs for debugging

### Metrics
- Track resource creation/update/deletion times
- Monitor API error rates
- Track resource drift detection

## Future Considerations

### API Evolution
- Monitor sm-cloud-api changes for new endpoints
- Plan for API versioning changes
- Consider GraphQL migration if applicable

### Provider Enhancement
- Implement provider-level configuration
- Add support for multiple Redis Cloud accounts
- Consider async operation improvements

### Community Contributions
- Maintain clear contribution guidelines
- Provide templates for new resources
- Establish code review processes

## Documentation Publishing Workflow

### Registry Publication Process

The terraform-provider-rediscloud uses an automated workflow to publish documentation to the Terraform Registry:

1. **Documentation Storage**: All documentation is stored in the `docs/` folder in the repository
2. **Release Trigger**: Documentation is published when a new version tag (e.g., `v1.2.3`) is pushed
3. **GoReleaser Integration**: The `.goreleaser.yml` configuration handles the release process
4. **Registry Update**: The Terraform Registry automatically pulls documentation from the latest release

### Release Workflow

The release process is defined in `.github/workflows/release.yml`:

```yaml
name: release
on:
  push:
    tags:
      - 'v*'
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
      - name: Set up Go
      - name: Import GPG key
      - name: Run GoReleaser
```

### Documentation Validation

Before releasing, ensure:

1. **All Resources Documented**: Every new resource has corresponding documentation
2. **Examples Tested**: All HCL examples are syntactically correct and functional
3. **Cross-References Valid**: Links between resources and data sources work correctly
4. **Import Instructions Accurate**: Import syntax has been tested
5. **Formatting Consistent**: Markdown formatting follows established patterns

### Registry Integration Points

#### Provider Configuration

The provider is registered in the Terraform Registry as:
- **Namespace**: `RedisLabs`
- **Provider Type**: `rediscloud`
- **Registry URL**: `registry.terraform.io/RedisLabs/rediscloud`

#### Documentation URLs

Published documentation follows this pattern:
- **Provider**: https://registry.terraform.io/providers/RedisLabs/rediscloud/latest
- **Resources**: https://registry.terraform.io/providers/RedisLabs/rediscloud/latest/docs/resources/[resource_name]
- **Data Sources**: https://registry.terraform.io/providers/RedisLabs/rediscloud/latest/docs/data-sources/[data_source_name]

### Maintenance and Updates

#### Documentation Lifecycle

1. **Development**: Create documentation alongside resource implementation
2. **Review**: Include documentation in code review process
3. **Testing**: Validate examples and formatting before merge
4. **Release**: Documentation is automatically published with provider releases
5. **Maintenance**: Update documentation when API changes or new features are added

#### Version Management

- **Breaking Changes**: Document in migration guides under `docs/guides/`
- **New Features**: Update existing documentation and add new resource docs
- **Deprecations**: Mark deprecated features clearly in documentation
- **API Changes**: Update examples and argument descriptions as needed

### Documentation Quality Checklist

Before submitting documentation:

- [ ] **Frontmatter Complete**: YAML metadata is properly formatted
- [ ] **Title Consistent**: Follows "Redis Cloud: rediscloud_[name]" pattern
- [ ] **Description Clear**: Concise explanation of resource purpose
- [ ] **Examples Working**: HCL code is syntactically correct and realistic
- [ ] **Arguments Documented**: All schema fields are documented with correct types
- [ ] **Attributes Listed**: All computed fields are documented
- [ ] **Import Instructions**: Import syntax is provided and tested
- [ ] **Timeouts Documented**: If custom timeouts are supported
- [ ] **Notes Added**: Important behaviors and limitations are highlighted
- [ ] **Cross-References**: Related resources and data sources are linked

## Key Findings: TasksController Classification

### Correction to Initial Analysis

The initial classification of `TasksController.java` as "Available for Implementation" was **incorrect**. Through code analysis, we discovered that:

#### TasksController is Used Internally

1. **Not a User-Facing Resource**: TasksController is an internal API mechanism for handling asynchronous operations
2. **Implementation Detail**: The terraform provider uses **resource status polling** rather than direct task polling
3. **Abstraction Layer**: Users interact with resources (subscriptions, databases), not the underlying tasks that create them

#### Current Async Operation Pattern

The terraform-provider-rediscloud implements asynchronous operations using:

```go
// Resource creation → Direct status polling (NOT task polling)
id, err := api.client.Resource.Create(ctx, request)
err = waitForResourceToBeActive(ctx, id, api) // Polls resource status directly
```

**Not this pattern:**
```go
// Alternative task-based pattern (NOT used)
taskId, err := api.client.Resource.CreateAsync(ctx, request)
err = waitForTaskToComplete(ctx, taskId, api) // Would poll TasksController
```

#### Evidence from Codebase

1. **Wait Functions**: All wait functions in the provider poll resource status directly:
   - `waitForSubscriptionToBeActive()` - polls subscription status
   - `waitForDatabaseToBeActive()` - polls database status
   - `waitForSubscriptionToBeDeleted()` - polls subscription status

2. **No Task Imports**: No imports of task-related services from rediscloud-go-api
3. **No Task IDs**: No handling of task IDs in resource creation flows
4. **Direct Status Checks**: All async operations check resource status fields, not task status

#### Implications for New Resource Development

When implementing new resources that may have asynchronous operations:

1. **Follow Existing Pattern**: Use resource status polling, not task polling
2. **Implement Wait Functions**: Create wait functions that poll the resource's status field
3. **Handle Dependencies**: Ensure dependent resources are ready before proceeding
4. **Don't Expose Tasks**: Tasks are implementation details, not user-facing resources

#### Updated Controller Classification

```
✅ Already Implemented:
- SubscriptionsController → rediscloud_subscription
- DatabasesController → rediscloud_subscription_database
- ACLController → rediscloud_acl_user, rediscloud_acl_role, rediscloud_acl_rule

🔄 Available for Implementation:
- UsersController → rediscloud_user
- DedicatedInstancesController → rediscloud_dedicated_instance
- MonitoringController → rediscloud_monitoring_config
- [others...]

🔧 Used Internally (NOT for direct resource implementation):
- TasksController → Internal async operation management
```

This correction ensures that future development efforts focus on appropriate controllers and follow the established patterns for handling asynchronous operations in the terraform provider.

---

This comprehensive guide provides everything needed to develop, document, and maintain resources in the terraform-provider-rediscloud. The documentation standards ensure consistency with the Terraform Registry and provide users with clear, actionable information for managing their Redis Enterprise Cloud infrastructure.
