# review-bot-api

GitHub webhook receiver. It verifies signed deliveries and logs pull request metadata; it does not run reviews yet.

## Configure the webhook secret

1. Generate a random secret (for example, `openssl rand -hex 32`). Do not commit or log it.
2. In **GitHub → Settings → Developer settings → GitHub Apps → Edit your app**, set the **Webhook secret** to that value. This is not the app's private key or an API token.
3. Save the same value in the **Password** field of a Vaultwarden login item named `github-review-app-v1`. Set `GITHUB_WEBHOOK_SECRET_ITEM` to use a different item name or ID; the environment variable contains the item reference, not the secret.

The API loads the secret once at startup and will not listen if it is missing, empty, or cannot be retrieved. The account running the API needs the `bw` CLI on its `PATH` and must already be logged in to the Vaultwarden server with access to the item. `SECRET_PASSWORD_FILE` points to the file used to unlock that vault, not a file containing the webhook secret. When rotating the secret on the systemd host, update GitHub and Vaultwarden, then run `make deploy` to sync the vault and restart the API. A plain service restart does not sync the vault. Deliveries can fail while the values differ.

## Run locally

```sh
cd review-bot-api
env=dev SECRET_PASSWORD_FILE=/absolute/path/to/password-file make run
```

`make run` builds the binary in `target/` and starts it. Use `make build` to build without running, `make test` to run tests, and `make clean` to remove the binary. Go 1.24 or newer and Make are required.

The server listens on `127.0.0.1:8888` by default. Set `PORT` to change the port, for example `env=dev SECRET_PASSWORD_FILE=/absolute/path/to/password-file PORT=8889 make run`. The Caddy route currently proxies to port `8888`; update the route if the deployed port changes. If containerizing, arrange for the API to be reachable at the VPS host's `127.0.0.1:8888` (the server itself currently binds loopback inside its network namespace).

## systemd deployment

On the systemd host, prepare deployment from the worktree root:

```sh
make -C review-bot-api deploy-ready PASSWORD_SOURCE=/absolute/path/to/existing-password-file
```

Pass a file path, not the password, and keep the source file outside the worktree. This target creates the `review-bot` service account if needed, prepares its home directory, and installs the password at `/etc/patchguard/secrets/bw-password` (`root:review-bot`, mode `0640`). If `/etc/patchguard/review-bot-api.env` does not exist, it creates it with `SECRET_PASSWORD_FILE=/etc/patchguard/secrets/bw-password`. Keep that assignment exactly once, on its own line. You can add other settings, comments, and blank lines. `deploy-ready` leaves existing config untouched, refuses to overwrite a different password, and does not deploy or start the service. The unit still sets `env=dev`; the systemd configuration takes precedence over `.env.dev` for `SECRET_PASSWORD_FILE`.

Deploy the current worktree with:

```sh
make -C review-bot-api deploy
```

Run this from the worktree root; alternatively run `make deploy` inside `review-bot-api/`. The command requires sudo access, systemd, Make, Go 1.24 or newer, and standard Linux tools including tar. Before creating a release or changing the running service, it checks the deployment prerequisites and runs `bw sync --nointeraction` as `review-bot`. Sync has a 60-second timeout; if it fails, the deploy stops without restarting the service. It then copies the current working files (including uncommitted and untracked files, but excluding `target/` and `.git`) into a new directory under `/opt/patchguard/releases/`, builds as `review-bot`, installs the unit and a working-directory drop-in, and enables and restarts the service. Source files must be readable by `review-bot`. Each deployment preserves earlier release directories; remove old releases manually when no longer needed. Do not put secrets in the source directory: untracked and ignored files are copied too.

The service runs `make run`, so it **builds on every start**. Go, Make, and `bw` must be available in the unit's `PATH`. The CLI must be logged in under the `review-bot` service account (`HOME=/var/lib/review-bot`), not just under your interactive account. If you use a custom item name or ID, set `GITHUB_WEBHOOK_SECRET_ITEM` in `/etc/patchguard/review-bot-api.env`. Deploy only trusted source. The unit's default `WorkingDirectory` is `/opt/patchguard/review-bot-api`, but the deployment drop-in overrides it with the selected release.

To change the port, add `PORT=8889` to `/etc/patchguard/review-bot-api.env`, then restart with `sudo systemctl restart review-bot-api.service`. Update Caddy's proxy port to match. Inspect logs with `journalctl -u review-bot-api.service -f`. Re-run the deployment command after changing the source or unit file.

## Test

Use `make test` to run the signature tests, or check that an unsigned request is rejected by the running API:

```sh
curl -i -X POST http://127.0.0.1:8888/webhooks/github
```

Expected: HTTP `403 Forbidden`. To test a signed delivery, point your GitHub App at `https://review-bot.kumarmo2.dev/webhooks/github` once the API, Caddy route, and DNS are deployed. Trigger or redeliver an event from a test repo, then check the response in GitHub's [Recent deliveries](GITHUB_WEBHOOKS.md#inspect-a-real-delivery). A valid `ping` or a valid `pull_request` delivery returns `200 OK`.
