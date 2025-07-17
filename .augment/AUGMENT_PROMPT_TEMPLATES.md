# Augment AI Prompt Templates for Terraform Provider Development

## Template 1: New Resource Implementation

### Context Setup
```
Reference Documents:
- TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md (comprehensive patterns and examples)
- REDIS_CLOUD_TERRAFORM_SUMMARY.md (architecture overview and controller classification)

Target Controller: {CONTROLLER_NAME} (e.g., UsersController.java)
Resource Name: rediscloud_{RESOURCE_NAME} (e.g., rediscloud_user)
```

### Prompt Template
```
Based on the TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md patterns, implement a complete Terraform resource for {CONTROLLER_NAME}.

Requirements:
1. Follow the Step-by-Step Implementation Guide (Section 5)
2. Use established schema patterns from existing resources
3. Implement async operations using resource status polling (NOT TasksController)
4. Follow the exact file structure and naming conventions

Generate the following files:
- provider/resource_rediscloud_{RESOURCE_NAME}.go
- docs/resources/rediscloud_{RESOURCE_NAME}.md  
- provider/resource_rediscloud_{RESOURCE_NAME}_test.go
- Update to provider/provider.go for resource registration

Deliverables:
[ ] Complete Go resource implementation with CRUD operations
[ ] Proper schema definition with validation
[ ] Wait functions for async operations (if needed)
[ ] Comprehensive documentation following templates
[ ] Acceptance tests with proper setup/teardown
[ ] Provider registration update

Quality Checkpoints:
- Schema follows TypeString/TypeInt/TypeBool patterns
- Error handling uses diag.FromErr()
- Documentation includes frontmatter, examples, and import instructions
- Tests use acctest.RandomWithPrefix() for unique naming
- No direct TasksController usage
```

## Template 2: Resource Documentation Generation

### Context Setup
```
Reference: TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md Section 6 (Documentation Standards)
Target Resource: rediscloud_{RESOURCE_NAME}
Existing Implementation: provider/resource_rediscloud_{RESOURCE_NAME}.go
```

### Prompt Template
```
Generate complete Terraform resource documentation for rediscloud_{RESOURCE_NAME} following the exact template from TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md.

Input: Analyze the Go resource implementation to extract:
- Schema fields and their types
- Required vs optional parameters
- Computed attributes
- Nested blocks structure
- Timeout configurations

Output: docs/resources/rediscloud_{RESOURCE_NAME}.md with:

Required Sections:
1. Frontmatter (layout, page_title, description)
2. Resource title and description
3. Example Usage (realistic HCL examples)
4. Argument Reference (all schema fields documented)
5. Timeouts section (if custom timeouts supported)
6. Attribute Reference (computed fields)
7. Import section with syntax example

Quality Standards:
- All schema fields documented with (Required/Optional) markers
- ForceNew fields marked as "change forces recreation"
- Nested blocks properly documented
- Examples are syntactically correct HCL
- Import syntax tested and accurate
- Cross-references to related resources included
```

## Template 3: Comprehensive Test Suite Generation

### Context Setup
```
Reference: TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md Section 8 (Testing Patterns)
Target Resource: rediscloud_{RESOURCE_NAME}
Implementation: provider/resource_rediscloud_{RESOURCE_NAME}.go
```

### Prompt Template
```
Generate comprehensive acceptance tests for rediscloud_{RESOURCE_NAME} following established testing patterns.

Create: provider/resource_rediscloud_{RESOURCE_NAME}_test.go

Test Cases to Include:
1. TestAccResourceRedisCloud{ResourceName}_basic
2. TestAccResourceRedisCloud{ResourceName}_update  
3. TestAccResourceRedisCloud{ResourceName}_import
4. TestAccResourceRedisCloud{ResourceName}_disappears

Test Structure Requirements:
- Use acctest.RandomWithPrefix("test-{resource}") for unique naming
- Include proper PreCheck, ProviderFactories, CheckDestroy
- Test all schema fields with TestCheckResourceAttr
- Include update scenarios for mutable fields
- Test import functionality
- Handle resource cleanup properly

Quality Checkpoints:
[ ] All required fields tested
[ ] Update scenarios for mutable fields
[ ] Import functionality verified
[ ] Proper resource cleanup in CheckDestroy
[ ] Realistic test configurations
[ ] Error scenarios handled appropriately
```

## Template 4: Code Review Against Established Patterns

### Context Setup
```
Reference: TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md (all sections)
Review Target: [Generated or existing code files]
```

### Prompt Template
```
Review the following Terraform resource implementation against the established patterns in TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md:

[PASTE CODE HERE]

Review Criteria:

1. Architecture Compliance:
   - Follows resource-to-controller mapping patterns
   - Uses correct Go API service imports
   - Proper provider registration

2. Schema Design:
   - Consistent field types (TypeString, TypeInt, TypeBool)
   - Proper Required/Optional/Computed settings
   - Appropriate ForceNew settings
   - Clear descriptions for all fields

3. CRUD Operations:
   - Standard function signatures
   - Proper error handling with diag.FromErr()
   - Correct resource ID management
   - Appropriate mutex usage if needed

4. Async Operations:
   - Uses resource status polling (NOT TasksController)
   - Proper wait function implementation
   - Correct state transitions defined
   - Appropriate timeouts set

5. Documentation:
   - Follows template structure exactly
   - All schema fields documented
   - Realistic examples provided
   - Import syntax included

6. Testing:
   - Comprehensive test coverage
   - Proper test naming conventions
   - Realistic test configurations
   - Import testing included

Provide specific feedback on:
- Pattern violations and how to fix them
- Missing components
- Quality improvements
- Consistency issues
```

## Template 5: sm-cloud-api Controller Analysis

### Context Setup
```
Reference: TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md Section 4 (Available Controllers)
Target: New controller in sm-cloud-api for potential Terraform resource
```

### Prompt Template
```
Analyze the {CONTROLLER_NAME} from sm-cloud-api to determine its suitability for Terraform resource implementation.

Analysis Framework:

1. Controller Classification:
   - Is this a user-facing resource or internal utility?
   - Does it manage stateful infrastructure components?
   - Is it similar to TasksController (internal-only) or SubscriptionsController (resource-worthy)?

2. API Endpoint Analysis:
   - What CRUD operations are available?
   - What is the resource lifecycle (create → active → delete)?
   - Are there async operations that require wait functions?

3. Terraform Resource Potential:
   - What would the resource name be? (rediscloud_{name})
   - What schema fields would be needed?
   - How would it integrate with existing resources?
   - What would be the user value proposition?

4. Implementation Complexity:
   - Does rediscloud-go-api support this controller?
   - Are there dependencies on other resources?
   - What async operation patterns would be needed?

5. Priority Assessment:
   - High/Medium/Low priority for implementation
   - Business value and user demand
   - Technical complexity vs. benefit ratio

Output Format:
- Classification: [User-Facing Resource | Internal Utility | Helper Controller]
- Recommended Action: [Implement | Skip | Investigate Further]
- Priority: [High | Medium | Low]
- Estimated Effort: [Low | Medium | High]
- Dependencies: [List any dependent resources or services]
- Notes: [Special considerations or concerns]
```

## Template 6: TerraForge AI Hackathon Workflow

### Context Setup
```
Project: TerraForge AI - Intelligent Terraform Resource Generator
References: 
- TERRAFORM_PROVIDER_DEVELOPMENT_GUIDE.md (complete patterns)
- REDIS_CLOUD_TERRAFORM_SUMMARY.md (controller priorities)
Target Controllers: [List 3 controllers for hackathon implementation]
```

### Prompt Template
```
Execute TerraForge AI workflow to automatically generate production-ready Terraform resources.

Phase 1: Analysis & Planning
1. Analyze target controllers: {CONTROLLER_LIST}
2. Extract patterns from existing implementations
3. Identify schema requirements for each resource
4. Plan async operation handling strategies

Phase 2: Code Generation
For each controller, generate:
1. Complete Go resource implementation
2. Comprehensive documentation
3. Full test suite
4. Provider registration updates

Phase 3: Quality Assurance
1. Validate against established patterns
2. Ensure async operations use resource polling (NOT tasks)
3. Verify documentation completeness
4. Confirm test coverage

Phase 4: Integration
1. Update provider registration
2. Ensure cross-references work
3. Validate import functionality
4. Test end-to-end workflows

Success Metrics:
- 3 complete resources implemented
- 100% pattern compliance
- Complete documentation and tests
- <30 minutes per resource (vs 6 hours manual)

Deliverables:
[ ] provider/resource_rediscloud_{resource1}.go + docs + tests
[ ] provider/resource_rediscloud_{resource2}.go + docs + tests  
[ ] provider/resource_rediscloud_{resource3}.go + docs + tests
[ ] Updated provider/provider.go
[ ] Workflow documentation for future use
[ ] Metrics on time savings and quality improvements

Quality Gates:
- All generated code passes linting
- Documentation follows exact templates
- Tests achieve 100% coverage
- No TasksController direct usage
- Import functionality works correctly
```

---

## Usage Instructions

1. **Choose the appropriate template** based on your development task
2. **Fill in the placeholders** (marked with {PLACEHOLDER}) with specific values
3. **Reference the context documents** to ensure AI has proper background
4. **Use the quality checkpoints** to validate outputs
5. **Iterate and refine** based on results

These templates ensure consistent, high-quality Terraform resource development while leveraging AI tools effectively.
