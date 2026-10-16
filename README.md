# Kite Plus Identity

The shared sign-in for every Kite Plus service, served at `id.kite.plus`.

It is a self-hosted open-source identity provider (OIDC), [ZITADEL](https://zitadel.com) v4 with its login v2 UI, deployed unmodified. This repository holds only what it takes to run it: deployment files, the repeatable setup tool, backups, runbooks, branding and email texts.

There is no public deployment yet; staging and production will run on the US server that already hosts Explore, and wait for DNS records and mail accounts. Everything else runs locally. The design, written in Chinese like the other Kite Plus projects, is in [docs/design](docs/design/README.md). Start with [architecture.md](docs/design/architecture.md) and the milestones in [roadmap.md](docs/design/roadmap.md). Operations are in [docs/runbooks](docs/runbooks/README.md).

## Layout

| Path | What |
|---|---|
| `config/` | Declarative instance settings: `base.yaml` plus one file per environment (`dev`, `staging`, `production`) |
| `cmd/identity-init` | Applies `config/` to an instance through the ZITADEL API; safe to re-run |
| `cmd/identity-mailer` | ZITADEL's HTTP email provider: lays account mail out in the Kite Plus look and sends it over SMTP (`deploy/mailer`) |
| `dev/` | The local stack (`identity-dev` compose project) |
| `deploy/` | Compose files for staging and production (standalone and edge mode), nginx config, Caddy site files for the edge, `.env.example` |
| `scripts/` | Backup, restore, upgrade, dev helpers |
| `test/e2e` | Browser-free end-to-end checks against the local stack |
| `branding/` | Logos used on the login pages and in mails |

## Local development

Requirements: Docker (Docker Desktop or OrbStack), Go 1.26, `make`, `openssl`.

```sh
make dev-up     # start ZITADEL, the login UI, PostgreSQL, Mailpit and nginx
make dev-init   # apply config/dev.yaml, write dev/out/*.env
make dev-test   # end-to-end check (code flow, logout, webhooks, event query, self-deletion)
make dev-down   # stop, keep data
make dev-reset  # wipe all data, start again and re-init (new client IDs and secrets)
```

`make dev-up` writes random secrets to `dev/.env` on first run; the file is gitignored and survives `dev-reset`. Everything listens on 127.0.0.1 only.

| What | Where |
|---|---|
| Issuer, login pages, console | `http://localhost:8091` (`/ui/console` for the admin console) |
| Mailpit SMTP / web UI | `localhost:1025` / `http://localhost:8025` |
| Admin login | `admin@localhost.test`, password `ADMIN_PASSWORD` in `dev/.env` |
| Test users | `dev/out/users.txt`: `alice` (zh) and `bob` (en), verified emails |

The agreed dev issuer is `http://localhost:8091` (`IDENTITY_DEV_PORT` in `dev/.env`, default 8091). If that port is taken on your machine, set another one there and run `make dev-reset`; `make dev-init` then writes `dev/out/NOTICE.txt` and the issuer in the env files follows the port. Services should read the issuer from their env file rather than hard-coding it.

Other services can send their own dev mail to Mailpit at `localhost:1025` (no auth, no TLS).

### Contract for Kite Plus services

`make dev-init` registers two confidential clients. Change these only together with the services that use them.

| | Explore | Comments |
|---|---|---|
| Redirect URIs | `http://localhost:4321/auth/callback`, `http://localhost:8080/auth/callback` | `http://localhost:8081/auth/callback` |
| Post-logout redirect | `http://localhost:4321/` | `http://localhost:8081/` |
| Back-channel logout | `http://host.docker.internal:8080/auth/backchannel-logout` | `http://host.docker.internal:8081/auth/backchannel-logout` |
| User event webhook | `http://host.docker.internal:8080/hooks/idp` | `http://host.docker.internal:8081/hooks/idp` |
| Env file | `dev/out/explore.env` | `dev/out/comments.env` |

Each env file holds, with `P` = `EXPLORE` or `COMMENTS`:

| Variable | Meaning |
|---|---|
| `P_OIDC_ISSUER` | Issuer; use OIDC discovery from here |
| `P_OIDC_CLIENT_ID`, `P_OIDC_CLIENT_SECRET` | Confidential client, `client_secret_basic` |
| `P_IDP_WEBHOOK_SECRET` | HMAC key of the service's Actions v2 target (see below) |
| `P_IDP_API_TOKEN` | Personal access token of the service's sync machine user, for reconciliation |

Client behaviour, as checked by `make dev-test`:

- Authorization code flow; PKCE `S256` is enforced when a challenge is sent. Scopes `openid profile email`. No consent screen.
- The ID token carries `sub`, `sid`, `email`, `email_verified`, `name`, `given_name`, `family_name`, `preferred_username`, `locale`, `amr`. `sub` is the ZITADEL user ID and is identical for every client. `aud` holds the client ID and the app's project ID; `azp` is the client ID. Each app has its own project, so a token for one service never lists the other.
- `sid` is the ZITADEL session ID. Store it with the local session.
- Back-channel logout: `POST`, `application/x-www-form-urlencoded`, field `logout_token`, a JWT signed with the issuer's keys (`iss`, `aud` = client ID, `sub`, `sid`, `jti`, `iat`, `exp` 15 minutes, `events` with `http://schemas.openid.net/event/backchannel-logout`, no `nonce`). It is sent when the session is ended at the identity provider (sign-out in the login UI, end_session, or an admin terminating it), up to 3 attempts. It is **not** sent when a session simply expires.
- The proxy rate-limits the login UI's form submissions per IP and across all clients, and the OIDC endpoints per IP (300 a minute, burst 100); a service's token requests come from one address, so keep them to sign-ins.
- `prompt=select_account` shows the account picker, which names the site asking and lets the reader continue with an account or use another; with no account signed in it goes straight to the sign-in form. Explore's web sign-in always sends it. Do not rely on `prompt=none`: login v2 shows the login page (no session) or an HTTP 400 page (stale session) instead of redirecting with `error=login_required` ([zitadel#12165](https://github.com/zitadel/zitadel/issues/12165)). A plain authorization request with a live session at the identity provider returns immediately without UI, which is what the comment widget popup needs.

The ZITADEL container reaches the services through `host.docker.internal`. On macOS (Docker Desktop and OrbStack) this reaches servers bound to `127.0.0.1` on the host; that was checked with OrbStack. On Linux, `host.docker.internal` maps to the Docker bridge gateway and the services must listen on `0.0.0.0` or that gateway address.

### Account page

Readers manage their Kite Plus Account at **`<issuer>/ui/v2/login/account`**: `http://localhost:8091/ui/v2/login/account` in dev, `https://id.kite.plus/ui/v2/login/account` in production. Link "Kite Plus 账号" / "管理 Kite Plus 账号" there and "删除 Kite Plus 账号" to `/ui/v2/login/account/delete`; both sign the reader in first if needed. Old links to `/ui/console/users/me` redirect to it; operators keep the console at `/ui/console`.

- The page is our own (`frontend/`): profile, email verification, password, passkeys, TOTP, signed-in devices and account deletion, in Chinese and English, light and dark, desktop and phone.
- Sign-in itself stays the pinned upstream Login V2 flow. Account actions run on the server for the user of the verified session only; changing security settings, signing out other devices and deleting the account need a sign-in within the last 10 minutes.
- Deleting needs the self-deletion role, which `identity-init` grants to every reader; production runs that step every five minutes, so a reader who signed up a moment ago may have to wait. In dev, run `make dev-readers` after signing up a new test reader. Deletion still pushes `user.removed`, and signing out a device still sends back-channel logout.
- Building and upgrading the frontend: [frontend/README.md](frontend/README.md).

### New readers' language

Users created through the login UI get `preferredLanguage` `zh` or `en` from the registering browser (the login UI's language switch, else `Accept-Language`), and `zh` when neither is Chinese or English or when a user is created through the API without one. Mails follow it, and the ID token carries it as `locale`. ZITADEL's tag for Simplified Chinese is `zh`.

### User event webhooks

ZITADEL Actions v2 sends one `POST` per event to each service's webhook for these events: `user.removed`, `user.deactivated`, `user.reactivated`, `user.locked`, `user.unlocked`. Body (`Content-Type: application/json`), exactly as ZITADEL v4.19.4 sends it:

```json
{
  "aggregateID": "394605853950385668",
  "aggregateType": "user",
  "resourceOwner": "394605815396343300",
  "instanceID": "394605785415458308",
  "version": "v2",
  "sequence": 3,
  "event_type": "user.deactivated",
  "created_at": "2026-10-11T06:16:57.531491Z",
  "userID": "394605785416048132"
}
```

- `aggregateID` is the affected user, i.e. the `sub`. `userID` is whoever made the change (an admin, a machine user, or the user themselves), not the affected user.
- `sequence` counts per user; `(aggregateID, sequence)` identifies an event, use it to deduplicate.
- `event_payload` is added when the event has one; these five events have none.
- Failed passwords no longer lock accounts (a lock could only be undone by an admin, so anyone could lock a reader out). `user.locked` now only comes from an admin or from too many wrong one-time codes after a correct password. Services suspend accounts on `user.deactivated` only; `user.locked` and `user.unlocked` are pushed for information.

Signature header:

```text
ZITADEL-Signature: t=1791699417,v1=78f4f798824b8eeee4fa0ea903b2b209f379a3bf5ab720504738db1901d4dd5c
```

`v1` is the lowercase hex HMAC-SHA256, keyed with `P_IDP_WEBHOOK_SECRET`, of the string `<t>.<raw body>`, where `t` is the Unix time in seconds. Compare in constant time and reject timestamps more than 5 minutes off. The format allows several `v1=` values; accept the request if any matches. (Rotating the key in v4.19.4 is immediate, without an overlap: [key-rotation.md](docs/runbooks/key-rotation.md).) `test/e2e/events_test.go` (`verifySignature`) is a reference implementation.

Delivery is best effort: one attempt with a 10 second timeout, no retries, and a non-2xx answer is only logged by ZITADEL. Answer `2xx` quickly and do the work afterwards.

`make dev-test` changes a throwaway user through every state, so a running service also receives those events for a `sub` it has never seen; ignore unknown users.

Every sync token can read every target's signing key (ZITADEL has no narrower role that can also read events), so a compromised service could forge webhooks for the other. Treat a webhook as a hint: before deleting or suspending anything, read the user with the API (next section) and act on what it says.

### Reconciliation

Pushes get lost (service down, network), so each service reconciles hourly with its `P_IDP_API_TOKEN`. The token belongs to machine user `explore-sync` or `comments-sync` with the instance role `IAM_OWNER_VIEWER`: read-only, but instance-wide.

List user events after a cursor (admin API `ListEvents`):

```sh
curl -s -X POST "$EXPLORE_OIDC_ISSUER/admin/v1/events/_search" \
  -H "Authorization: Bearer $EXPLORE_IDP_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
        "asc": true,
        "limit": 1000,
        "from": "2026-10-11T06:00:00Z",
        "aggregateTypes": ["user"],
        "eventTypes": ["user.removed", "user.deactivated", "user.reactivated", "user.locked", "user.unlocked"]
      }'
```

```json
{
  "events": [
    {
      "editor": {"userId": "394605785416048132", "displayName": "identity-init", "service": "zitadel"},
      "aggregate": {"id": "394605831418584580", "type": {"type": "user"}, "resourceOwner": "394605815396343300"},
      "sequence": "3",
      "creationDate": "2026-10-11T06:16:44.097020Z",
      "type": {"type": "user.deactivated"}
    }
  ]
}
```

- `from` is exclusive (`creationDate > from`) and results are ordered by `creationDate` when `asc` is true. Store the `creationDate` of the last event as the cursor. To be safe against events committed slightly out of order and against equal timestamps at a page boundary, query from `cursor - 5 minutes` and skip `(aggregate.id, sequence)` pairs already handled. A full page (`limit` results) means: query again from the new cursor.
- `aggregate.id` is the `sub`. The response also carries localized labels, which can be ignored.

Then read the current state of each affected user rather than replaying events:

```sh
curl -s "$EXPLORE_OIDC_ISSUER/v2/users/$SUB" -H "Authorization: Bearer $EXPLORE_IDP_API_TOKEN"
```

`404` means the user is deleted. Otherwise `user.state` is `USER_STATE_ACTIVE`, `USER_STATE_INACTIVE` (deactivated) or `USER_STATE_LOCKED`. The same call confirms a webhook before acting on it.

## Deployment

Production and staging share one server with Explore and the comment service, and Explore's Caddy is the only thing on ports 80 and 443 there. The identity service runs in **edge mode**: `deploy/docker-compose.edge.yaml`, enabled through `COMPOSE_FILE` in `deploy/.env`, takes nginx off the host ports and onto the shared `kite-edge` Docker network as `identity`; nginx speaks plain HTTP there, trusts `X-Forwarded-For` only from that network (Caddy sends the one reader address it found behind Cloudflare), and the rate limits keep counting readers. `deploy/edge/identity.caddy` and `identity-staging.caddy` are Caddy's site blocks. Staging is a second checkout and compose project on the same server, started for tests and upgrades only.

On its own, `deploy/docker-compose.yaml` is the **standalone mode** for a server without a shared proxy: nginx owns ports 80 and 443 and gets its certificate through its ACME module. Both modes share the nginx rate limits, njs and routing.

`scripts/backup.sh` writes an age-encrypted dump and, with `BACKUP_REMOTE` set, copies it off the server with rclone. See [docs/runbooks](docs/runbooks/README.md) for first start, upgrades, backups and restores, and [architecture.md section 4](docs/design/architecture.md#4-部署) for why it is deployed this way. Configuration changes go into `config/` first and are applied with `identity-init`, never clicked in the console.
