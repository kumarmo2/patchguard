# GitHub `pull_request` webhook: Go DTO mapping

This documents the initial [wire DTO for PatchGuard](../../review-bot-api/github_pull_request.go), not a review-trigger policy or a copy of GitHub's full schema. Sources: [GitHub webhook event reference][events] and [GitHub's published OpenAPI schema][schema]. Use the action-specific schema when determining what is required or nullable.

## Fields to model first

GitHub sends the event name in `X-GitHub-Event`; when it is `pull_request`, the JSON body has a separate `action`. The `opened` and `edited` schemas require `action`, `number`, `pull_request`, `repository`, and `sender`. `edited` additionally requires `changes`. Delivery ID and signature are HTTP headers, not DTO fields. [Events][events] · [Headers][headers] · [Edited schema][edited]

| JSON path | Proposed Go type | Why keep it |
| --- | --- | --- |
| `action` | `string` | Select action-specific handling after decoding. Unknown actions should not fail JSON decoding. |
| `number` | `int64` | Repository-scoped PR number; use this, not `pull_request.id`, for the [REST PR lookup][get-pr]. |
| `repository.id`, `repository.full_name` | `int64`, `string` | Use `(repository.id, number)` for PR identity and `owner/repo` for REST routing. For fork PRs, route using the event repository, not `pull_request.head.repo`. |
| `pull_request.title`, `html_url` | `string` | Display context and a browser link. |
| `pull_request.body` | `*string` | Current description; GitHub permits `null`. Preserve the distinction between no text and an empty string. |
| `pull_request.draft` | `*bool` | Lets the receiver distinguish explicit `false` from missing or null if it needs to validate draft status. |
| `pull_request.head.sha`, `base.sha` | `string` | Source and target commits in this delivery. Keep both when identifying the revision to review. |
| `sender.id`, `sender.login` | `int64`, `string` | Actor that triggered the event, not necessarily the PR author. Not a permission check. |
| `installation.id` | `int64` inside an optional `*Installation` | The installation to use for GitHub App authentication; require it when the application needs installation credentials, not for every kind of webhook. |
| `changes.body.from` | `*GitHubPRChanges` containing optional `*GitHubPRBodyChange` with `From *string` | `changes` is specific to `edited` here. The [edited schema][edited] requires `from` as a string when `body` is present. The pointer lets later validation distinguish a missing/null value from a valid empty previous body. |

The `edited` payload can have `changes.title` or `changes.base` without `changes.body`. Use `changes.body != nil` to tell whether the description was edited; compare `changes.body.from` with `pull_request.body` for any mention logic. An omitted field and explicit JSON `null` both decode to `nil` with Go pointers; use presence-aware decoding only when the distinction matters. [Edited schema][edited]

## Actions and scope

- `opened`: a PR was created; the payload's `draft` field can support an app-level draft policy.
- `edited`: model the old description only if the bot needs description edits. Title-only or base-only edits are valid deliveries.
- `synchronize`: GitHub includes required top-level `before` and `after` strings. Leave them out until a handler needs this action. [Synchronize schema][sync]
- `closed`: check `pull_request.merged` to distinguish merged from closed without merging. There is no `merged` action. [Events][events]
- Reviews, review comments, and PR conversation comments are separate events (`pull_request_review`, `pull_request_review_comment`, and `issue_comment`); they do not need to fit this DTO. [Events][events]

**Repo decision to keep separate:** [`review-bot-api/GITHUB_WEBHOOKS.md`](../../review-bot-api/GITHUB_WEBHOOKS.md) describes `opened` plus PR-description mention edits. The [earlier architecture guide](self-hosted-pr-review-bot.md) describes non-draft `opened` plus explicit conversation-comment requests. Neither policy needs to be baked into the JSON DTO.

## Receiver boundary and test cases

When wiring the receiver, bound reads and validate `X-Hub-Signature-256` on the raw bytes before decoding; use `X-GitHub-Event` to select the DTO. Decode only what the app uses and allow unknown JSON fields. Separately validate the fields required by the action the bot actually processes; for example, don't look for `changes` on `opened`. Use `X-GitHub-Delivery` for delivery deduplication; redelivery reuses that ID. [Signature validation][signature] · [Best practices][practices]

Start with decoding fixtures for `opened` with a null description, `edited` with an empty previous description, `edited` with only a title change, and an otherwise valid payload with extra fields. Keep these tests independent of handler wiring and trigger policy.

[events]: https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request
[headers]: https://docs.github.com/en/webhooks/webhook-events-and-payloads#delivery-headers
[schema]: https://github.com/github/rest-api-description/blob/c9c95fe34ced05fe076001f5c55704742bc8e7f8/descriptions/api.github.com/api.github.com.json#L265763
[edited]: https://github.com/github/rest-api-description/blob/c9c95fe34ced05fe076001f5c55704742bc8e7f8/descriptions/api.github.com/api.github.com.json#L256482
[sync]: https://github.com/github/rest-api-description/blob/c9c95fe34ced05fe076001f5c55704742bc8e7f8/descriptions/api.github.com/api.github.com.json#L305785
[get-pr]: https://docs.github.com/en/rest/pulls/pulls#get-a-pull-request
[signature]: https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
[practices]: https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks
