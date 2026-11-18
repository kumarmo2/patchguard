# review-bot-api

Minimal unauthenticated test webhook server. It does not verify or process webhook payloads.

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

Pass a file path, not the password, and keep the source file outside the worktree. This target creates the `review-bot` service account if needed, prepares its home directory, and installs the password at `/etc/patchguard/secrets/bw-password` (`root:review-bot`, mode `0640`). It creates `/etc/patchguard/review-bot-api.env` with the single line `SECRET_PASSWORD_FILE=/etc/patchguard/secrets/bw-password`. It does not overwrite a different existing password or config, deploy a release, or start the service. The unit still sets `env=dev`; the systemd configuration takes precedence over `.env.dev` for `SECRET_PASSWORD_FILE`.

Deploy the current worktree with:

```sh
make -C review-bot-api deploy
```

Run this from the worktree root; alternatively run `make deploy` inside `review-bot-api/`. The command requires sudo access, systemd, Make, Go 1.24 or newer, and standard Linux tools including tar. It fails before creating a release if prerequisites are missing. It then copies the current working files (including uncommitted and untracked files, but excluding `target/` and `.git`) into a new directory under `/opt/patchguard/releases/`, builds as `review-bot`, installs the unit and a working-directory drop-in, and enables and restarts the service. Source files must be readable by `review-bot`. Each deployment preserves earlier release directories; remove old releases manually when no longer needed. Do not put secrets in the source directory: untracked and ignored files are copied too.

The service runs `make run`, so it **builds on every start**. Go and Make must be available in the unit's `PATH`; a build failure prevents startup. Deploy only trusted source. The unit's default `WorkingDirectory` is `/opt/patchguard/review-bot-api`, but the deployment drop-in overrides it with the selected release.

To change the port, use `sudo systemctl edit review-bot-api.service` and add a `[Service]` section with `Environment=PORT=8889`, then restart with `sudo systemctl restart review-bot-api.service`. Update Caddy's proxy port to match. Inspect logs with `journalctl -u review-bot-api.service -f`. Re-run the deployment command after changing the source or unit file.

## Test

Test locally:

```sh
curl -i -X POST http://127.0.0.1:8888/webhooks/github
```

Expected: HTTP `200 OK` with body `OK`. Once the API and Caddy route are deployed and DNS is configured, the public URL is `https://review-bot.kumarmo2.dev/webhooks/github`.

**This endpoint is intentionally unauthenticated.** Do not configure production webhooks or send sensitive payloads until webhook verification and processing are implemented. The checked-in Caddy route makes it public when deployed.
