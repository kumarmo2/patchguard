# review-bot-api

Minimal unauthenticated test webhook server. It does not verify or process webhook payloads.

```sh
cd review-bot-api
cargo run
```

The server listens on `127.0.0.1:8888` by default. Set `PORT` to change the port, for example `PORT=8889 cargo run`. The Caddy route currently proxies to port `8888`; update the route if the deployed port changes. If containerizing, arrange for the API to be reachable at the VPS host's `127.0.0.1:8888` (the server itself currently binds loopback inside its network namespace).

Test locally:

```sh
curl -i -X POST http://127.0.0.1:8888/webhooks/github
```

Expected: HTTP `200 OK` with body `OK`. Once the API and Caddy route are deployed and DNS is configured, the public URL is `https://review-bot.kumarmo2.dev/webhooks/github`.

**This endpoint is intentionally unauthenticated.** Do not configure production webhooks or send sensitive payloads until webhook verification and processing are implemented. The checked-in Caddy route makes it public when deployed.
