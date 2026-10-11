# Kite Plus Identity

The shared sign-in for every Kite Plus service, served at `id.kite.plus`.

It is a self-hosted open-source identity provider (OIDC), [ZITADEL](https://zitadel.com) v4 with its login v2 UI, deployed unmodified. This repository holds only what it takes to run it: deployment files, the repeatable setup tool, backups, runbooks, branding and email texts.

There is no public deployment yet; staging and production wait for a server, DNS and mail accounts. Everything else runs locally. The design, written in Chinese like the other Kite Plus projects, is in [docs/design](docs/design/README.md). Start with [architecture.md](docs/design/architecture.md) and the milestones in [roadmap.md](docs/design/roadmap.md). Operations are in [docs/runbooks](docs/runbooks/README.md).

## Layout

| Path | What |
|---|---|
| `config/` | Declarative instance settings: `base.yaml` plus one file per environment (`dev`, `staging`, `production`) |
| `cmd/identity-init` | Applies `config/` to an instance through the ZITADEL API; safe to re-run |
| `dev/` | The local stack (`identity-dev` compose project) |
| `deploy/` | Production compose file, nginx config, `.env.example` |
| `scripts/` | Backup, restore, upgrade, dev helpers |
| `test/e2e` | Browser-free end-to-end checks against the local stack |
| `branding/` | Logos used on the login pages and in mails |

## Local development

Requirements: Docker (Docker Desktop or OrbStack), Go 1.26, `make`, `openssl`.

```sh
make dev-up     # start ZITADEL, the login UI, PostgreSQL, Mailpit and nginx
make dev-init   # apply config/dev.yaml, write dev/out/*.env
make dev-test   # end-to-end check (code flow, logout, webhooks, event query)
make dev-down   # stop, keep data
make dev-reset  # wipe all data, start again and re-init (new client IDs and secrets)
```

`make dev-up` writes random secrets to `dev/.env` on first run; the file is gitignored and survives `dev-reset`. Everything listens on 127.0.0.1 only.

| What | Where |
|---|---|
| Issuer, login pages, console | `http://localhost:8090` (`/ui/console` for the admin console) |
| Mailpit SMTP / web UI | `localhost:1025` / `http://localhost:8025` |
| Admin login | `admin@localhost.test`, password `ADMIN_PASSWORD` in `dev/.env` |
| Test users | `dev/out/users.txt`: `alice` (zh) and `bob` (en), verified emails |

The port is `IDENTITY_DEV_PORT` in `dev/.env` (default 8090). If 8090 is taken on your machine, set another port there and run `make dev-reset`; `make dev-init` then writes `dev/out/NOTICE.txt` and the issuer in the env files follows the port. Other services must read the issuer from their env file rather than hard-coding it.

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
- `prompt=select_account` shows the account picker. Do not rely on `prompt=none`: login v2 shows the login page (no session) or an HTTP 400 page (stale session) instead of redirecting with `error=login_required` ([zitadel#12165](https://github.com/zitadel/zitadel/issues/12165)). A plain authorization request with a live session at the identity provider returns immediately without UI, which is what the comment widget popup needs.

The ZITADEL container reaches the services through `host.docker.internal`. On macOS (Docker Desktop and OrbStack) this reaches servers bound to `127.0.0.1` on the host; that was checked with OrbStack. On Linux, `host.docker.internal` maps to the Docker bridge gateway and the services must listen on `0.0.0.0` or that gateway address.

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

Signature header:

```text
ZITADEL-Signature: t=1791699417,v1=78f4f798824b8eeee4fa0ea903b2b209f379a3bf5ab720504738db1901d4dd5c
```

`v1` is the lowercase hex HMAC-SHA256, keyed with `P_IDP_WEBHOOK_SECRET`, of the string `<t>.<raw body>`, where `t` is the Unix time in seconds. Compare in constant time and reject timestamps more than 5 minutes off. During key rotation several `v1=` values may appear; accept the request if any matches. `test/e2e/events_test.go` (`verifySignature`) is a reference implementation.

Delivery is best effort: one attempt with a 10 second timeout, no retries, and a non-2xx answer is only logged by ZITADEL. Answer `2xx` quickly and do the work afterwards.

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

Production and staging run `deploy/docker-compose.yaml` behind nginx; see [docs/runbooks](docs/runbooks/README.md) for first start, upgrades, backups and restores. Configuration changes go into `config/` first and are applied with `identity-init`, never clicked in the console.
