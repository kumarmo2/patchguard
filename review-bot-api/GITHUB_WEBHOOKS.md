# GitHub webhooks for PatchGuard

GitHub sends an HTTP `POST` to `https://review-bot.kumarmo2.dev/webhooks/github`. For pull requests, set **Repository permissions → Pull requests → Read-only** and subscribe to **Pull request**. You don't need **Issue comment** for mentions in the PR description.

## What GitHub sends

| Header | Purpose |
| --- | --- |
| `Content-Type: application/json` | The request body is JSON. |
| `X-GitHub-Event: pull_request` | The event type. The action (`opened`, `edited`, etc.) is a field in the JSON body, not part of the header. |
| `X-GitHub-Delivery` | Delivery identifier for logging and deduplication. |
| `X-Hub-Signature-256: sha256=…` | HMAC-SHA256 signature of the raw request body. GitHub includes it when a webhook secret is configured. |

Example `pull_request` payload when a description is edited (abbreviated):

```json
{
  "action": "edited",
  "number": 42,
  "changes": {
    "body": { "from": "Please review this change." }
  },
  "pull_request": {
    "title": "Add login",
    "body": "Please review this change, @patchguard-bot.",
    "html_url": "https://github.com/owner/repo/pull/42"
  },
  "repository": { "full_name": "owner/repo" },
  "sender": { "login": "contributor" },
  "installation": { "id": 123456 }
}
```

## When the bot should run

After verifying the signature, handle only `X-GitHub-Event: pull_request`:

- **`opened`**: Run for every new PR, with or without a bot mention. A mention in the initial description does not trigger a second run.
- **`edited`**: Run only if `changes.body` is present and the bot mention is in the new description (`pull_request.body`) but not the old one (`changes.body.from`). An edit that leaves an existing mention in place does not retrigger the bot. Skip title-only or base-branch-only edits.
- **Other actions and events**: Ignore them for review purposes. This includes new commits (`synchronize`), reopened PRs, comments, and installation events.

The handler computes HMAC-SHA256 over the exact request bytes using the [shared secret loaded from Vaultwarden](README.md#configure-the-webhook-secret). It compares the digest in constant time and returns `403 Forbidden` for a missing or invalid signature. After verification it decodes `pull_request` deliveries and logs only the delivery ID, action, repository, and PR number. It acknowledges other signed events, including `ping`, without decoding them. Review triggers are not implemented yet.

The signature covers the body, not the event or delivery ID headers, and does not prevent replay. Keep HTTPS enabled. Before triggering reviews, add delivery deduplication and check that the installation and repository are allowed to trigger the bot.

## Inspect a real delivery

In GitHub, go to **Settings → Developer settings → GitHub Apps → Edit your app → Advanced → Recent deliveries**. Select a delivery to see the request headers, payload, and server response. GitHub lists deliveries from the past **3 days**. Open a PR in the test repository to get a `pull_request` delivery; an `installation` delivery isn't a test of the review trigger. A `2xx` only tells you that the endpoint acknowledged the request.

## References

- [GitHub: Webhook delivery headers and example request](https://docs.github.com/en/webhooks/webhook-events-and-payloads#delivery-headers)
- [GitHub: Pull request event and payload](https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request)
- [GitHub: Validating webhook deliveries](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries)
- [GitHub: Viewing webhook deliveries](https://docs.github.com/en/webhooks/testing-and-troubleshooting-webhooks/viewing-webhook-deliveries#viewing-deliveries-for-github-app-webhooks)
- [Octokit: `pull_request.edited` payload schema](https://github.com/octokit/webhooks/blob/main/payload-schemas/api.github.com/pull_request/edited.schema.json)
