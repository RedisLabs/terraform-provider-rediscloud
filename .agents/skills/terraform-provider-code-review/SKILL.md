---
name: terraform-provider-code-review
description: Review Redis Cloud Terraform provider pull requests, including stacked SDKv2 to Plugin Framework migrations. Compare Terraform configuration, plans, persisted state and Cloud API operations with the parent, then give actionable findings or explain an approval.
---

# Review the Redis Cloud Terraform provider

Start with the PR's actual base and head commits, its purpose, the existing review threads, and the checks attached to that head. Fetch the refs. Read the complete three-dot diff and trace changed helpers to their callers. For a stack, identify the parent PR and compare with that branch. Report a shared defect where it was introduced and tell downstream authors which parent fix they need. Recheck the head before submitting a verdict.

Read [maintainer review examples](references/maintainer-reviews.md) for how to conduct the review. For a migration, read the complete [implementation skill](../migrate-to-plugin-framework/SKILL.md) alongside the SDKv2 predecessor. Use the following checks to test the behavior the implementation promises.

## Compare what users can configure, plan and upgrade

- Compare the exported protocol-v5 schemas by path, including attribute versus block, list versus set, cardinality, defaults, sensitivity, validation and replacement. Check the mux registers each public name once and preserves the implicit legacy ID and import format. A schema/model type equality assertion alone cannot verify flags, nesting mode or plan modifiers.
- Distinguish a configurable nested object block from an output-only collection. In protocol v5, a computed list of objects can be a plain computed ListAttribute with an object element type (likewise for sets). A nested block can plan as empty when a data source read is deferred. Test a downstream expression indexing that output while the input ID is unknown. Inspect the actual SDK schema for explicit ConfigMode overrides. See the [worked correction](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/1176).
- Trace omission, explicit zero/false/empty, unknown inputs, null elements and absent API fields separately. SDK GetOk often treats a zero value as absent. Test zero, one and multiple matches for singular lookups. Keep the old null versus known-empty representation where users can observe it.
- Exercise a no-op upgrade from released state, import followed by plan, a refresh after external drift, and a change to one field. Check the plan for unintended replacement and for in-place changes the API cannot perform. UseStateForUnknown is appropriate only when the prior value remains valid across that operation. Consider pending provisioning and incomplete reads before treating an apparent change as real. [Migration testing](https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing)
- For configurable collections preserve known planned values on apply and reconcile actual drift on refresh. For computed lists from unordered API results use a stable identifier or complete sort key. Compare two responses with the same members in opposite order. Keep genuine order where it is part of the contract. Check identical nested shapes across consumers before sharing a custom type. Verify the nested element schema separately and propagate conversion diagnostics.

## Follow a change through the API

Trace plan → API request → task completion → read → state. Verify the endpoint and composite ID for the subscription type and region. Inspect omitted, empty and populated payload fields, especially when another Terraform resource owns part of the remote object. Verify the scope of API calls in list reads. An extra request per returned object can make a previously cheap plan expensive.

Locate the first successful remote mutation. If a subsequent wait or GET fails, check what identity Terraform actually receives in the response state. An ID assigned to a local model does not survive an early return by itself. Exercise retry or import after that failure. Check immutable fields, create-only secrets, and which values the API can return during refresh. Keep secrets out of diagnostics, fixtures, logs and review excerpts.

For asynchronous operations, verify which states mean pending, ready, terminal failure and confirmed deletion. Distinguish temporary invisibility from absence, honor cancellation and operation timeouts, and retain the last observed state in errors where available. Compare delete and not-found handling with the specific resource's previous behavior. Read SDK retry semantics when a pending callback returns nil.

## Gather evidence and give a verdict

Run the affected package and changed helper tests on the reviewed commit. Prefer a fake API plus a Terraform plan/apply assertion for state and lifecycle regressions. An acceptance test with TF_ACC disabled does not exercise cloud behavior. Inspect what CI actually ran, particularly for stacked branches. State any material verification gap. Use the repository [build and test guide](../../../AGENTS.md) for broader gates.

For each blocking finding, identify the changed location, smallest realistic trigger, user-visible outcome, and the fix or regression assertion that would settle it. Link an existing thread if it already describes the defect. Group related findings in one review. Mark a preference or a future improvement as optional. When re-reviewing, inspect the fixing diff, reproduce or check the regression case and acknowledge the resolved concern. Leave others' review decisions to them.

When asked to submit reviews, request changes for a verified blocker. If the head is sound, approve with the paths or behaviors checked and the test evidence. If essential evidence is missing, explain exactly what prevents a verdict. Submit against the reviewed commit and verify the recorded verdict. A review does not authorize merging the PR.

## Summarize reviews in dev Slack

When asked to post results in the development channel, use two lists. Group PRs by the latest review you submitted in this batch and verify each verdict on GitHub.

- **Approved (count)** — one bullet per approved PR. Link the PR number to the review and give one sentence naming the change and what justified approval.
- **Comments left (count)** — one bullet per PR where you commented or requested changes. Link the PR number to the review. Start its one-line summary with **Changes requested** or **Comment** and state the specific fix, question or suggestion.

Include each reviewed PR exactly once, even when several share the same finding. An approval with optional notes belongs under Approved. Keep both headings when a list is empty and write “None” beneath it.
