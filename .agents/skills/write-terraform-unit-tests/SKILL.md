---
name: write-terraform-unit-tests
description: >-
  Write or refactor Go tests that exercise Terraform resources and data sources
  against a fake Redis Cloud API, using direct SDKv2 or Framework handler calls
  or terraform-plugin-testing resource.UnitTest. Use explicit scenarios, shallow
  test runners, and small fixtures. Applies during feature work, bug fixes, and
  migrations. Excludes pure helper or schema tests and live acceptance tests.
---

# Write Terraform unit tests with a fake API

Read `AGENTS.md` for repository conventions and commands. Apply this skill when
testing real resource or data-source behaviour with fake API responses. Tests
may call handlers directly or exercise them through local Terraform execution.

Make the behaviour and complete set of scenarios understandable at a glance.
Readable repetition is preferable to making a reader reconstruct a test from
generated combinations, shared defaults, or layers of helpers.

## List scenarios explicitly

- Write a flat table of explicitly named, fully specified cases. Use one shallow
  `for _, tc := range tests` with one `t.Run(tc.name, ...)` to execute the rows.
- Spell out each meaningful combination of resource family, mode, and outcome
  as its own row. Do not generate cases with nested ranges over resource names,
  booleans, operations, or modes. Do not generate names from boolean flags.
- Keep business inputs and precise expectations together in each row. Include
  relevant configuration, initial API state, changed or unchanged values,
  expected requests, resulting state, and diagnostics. Repeat values when that
  makes the case self-contained.
- Put both initial and final API responses in the rows when visibility changes
  over time. Do not hide a fixed final response in a helper while making only
  the initial response configurable.
- When a table handles both success and failure, use an explicit `success bool`
  alongside `wantError` or similarly named error expectations. Add `wantTimeout`
  when timeout handling needs a separate check.
  Do not infer success from an empty error string or select behaviour by the
  test name. A name describes a case; it is not control data.
- When a table mixes successful and failing cases, group the successful rows
  first and the failing rows second. Separate them with short comment lines,
  such as `// These cases should succeed.` and `// These cases should fail.`
- Use visually distinct, valid identifiers. For example, account IDs such as
  `111111111111`, `222222222222`, and `333333333333` make membership changes
  obvious without violating the required identifier format.

Ordinary loops for assertions and fixture mechanics are fine. A descriptive
wrapper `t.Run` may group a coherent table, but do not use nested groups to
generate a scenario matrix. Put substantially different workflows, such as
Create and Update with different setup and assertions, in separate test
functions.

## Keep the runner short

Arrange the resource data and fake API, invoke the real handler, then assert
the behaviour. Keep these steps visible in the runner. Move SDK state
construction, client setup, routing, JSON handling, and synchronized request
recording into small local helpers.

For example, a table of Update cases can use this runner shape. The table must
provide the meaningful inputs in `given` and the exact expected requests and
outcome in each row. The fixture helpers supply mechanical setup and safe access
to recorded requests.

```go
for _, tc := range tests {
	t.Run(tc.name, func(t *testing.T) {
		resource, data := newResourceData(t, tc.given)
		ctx, fixture := newTestAPI(t, tc.given)

		diagnostics := resource.UpdateContext(ctx, data, fixture.client)

		require.Equal(t, tc.wantRequests, fixture.updateRequests())
		if tc.success {
			require.False(t, diagnostics.HasError(), "%v", diagnostics)
			return
		}

		require.True(t, diagnostics.HasError(), "%v", diagnostics)
		require.Contains(t, fmt.Sprint(diagnostics), tc.wantError)
	})
}
```

Use the same shape for Framework handlers or Terraform-driven tests with the
appropriate request, response, or state assertions. Avoid building a callback
framework or a conditional runner to make different workflows share this code.
Return after handling success so failure assertions stay at the same nesting
level. Declare API handlers, getters, and other fixture callbacks as named
functions or local variables before registering them or passing them to helpers.

## Share mechanics without hiding the case

Use `provider/testhelpers.FakeAPI` and `NewAPIClient` when they fit the API being
tested. Keep resource-specific behaviour in a small fixture local to the test
file. Reuse shared routing, JSON encoding, and task handling instead of writing
another transport or server for the same protocol.

Use SDK request and response types for wire models and exported SDK constants
for status or enum values. When the SDK has no exported constant, a small local
constant is sufficient. Model the endpoints and task response forms the SDK
actually uses. Add only the fake behaviour needed by the scenarios.

Let helpers hide map conversions, Terraform state construction, request
decoding, and synchronization. Keep the requested values and expected payloads
visible in the rows. Do not let a helper silently choose a business outcome,
supply meaningful missing inputs, or recompute the expected result using the
implementation under test.

Expose recorded requests through fixture methods that handle safe access. With
`FakeAPI`, use `WithHandlersPaused` when test code reads or writes fixture state
outside a handler. Keep the mutex and snapshot mechanics out of the runner.
Give parallel cases independent fixtures and identifiers when production
handlers lock resources by identifier.

Use the established test package and suite. Prefer a small, harmless export of a
new helper over creating a separate internal test suite solely to access that
helper, unless an existing package convention dictates otherwise.

## Prove the intended behaviour

Assert the important request fields, including fields that must be omitted,
and the resulting Terraform state or diagnostics. A generic error assertion
does not establish that the intended API call or verification step was reached.
Check the expected error and relevant recorded requests so setup failures cannot
satisfy a failure case.

Establish changed or unchanged Terraform values explicitly when testing Update.
Keep that meaningful distinction in the case data even when a helper handles
the SDK state construction.

A deliberate API failure may isolate request construction and error propagation.
Document that fixture contract. Tests of successful completion, state refresh,
or eventual consistency must actually exercise those behaviours rather than
stop at an earlier failure.

For asynchronous behaviour, distinguish task completion from visibility of the
requested result. Script stale and final reads explicitly when testing retries,
and assert that the retry or mutation happened when it matters to the case.
Use bounded contexts and cleanup for cancellation so a failure case cannot wait
for the production timeout.

Prefer existing client timing overrides. Where the test and its goroutines can
run inside a fake-time bubble, `testing/synctest` can exercise polling without
wall-clock sleeps or new production timing controls. Explain that use beside the
test and link to the [official synctest guide](https://go.dev/blog/synctest).

## Explain the non-obvious parts

Follow the comment rules in `AGENTS.md`. Explain the business rule or regression
a test protects when the reason is not clear from the cases. Document fixture
contracts such as the order of scripted reads, whether the final response
repeats, why a callback cancels a context, or why one request counts toward more
than one logical operation. Leave straightforward setup and assertions
uncommented.

Helper comments should explain their purpose, required inputs, and non-obvious
behaviour in natural sentences. Describe the code a reader can see, without
referring to earlier drafts or choices made during the conversation.

## Validate the focused change

Run the changed tests with `go test`, using `-run` to select their names and
`-count=1` to avoid cached results. Include relevant existing consumers when
changing a shared fake, and use `-race` when the change involves concurrent
fixture or task handling.

These tests should run without Redis Cloud credentials or live infrastructure.
Use the local-socket and cache guidance in `AGENTS.md` when the execution
environment restricts Terraform-based tests. Report environment failures and
test failures accurately; do not weaken assertions to make a run pass.
