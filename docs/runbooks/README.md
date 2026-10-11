# Kite Plus 账号 运维手册

> 状态：初稿，I1 在预发布环境上按手册走一遍后修订；已按与 Explore 同机的边缘模式改写 · 最近更新：2026-10-11
> 为什么这样部署、升级时限、备份保存多久，以 [architecture.md §4–§5](../design/architecture.md#4-部署) 为准；这里只写怎么做。

| 手册 | 什么时候用 |
|---|---|
| [first-start.md](first-start.md) | 在一台新服务器上第一次部署预发布或生产 |
| [upgrade.md](upgrade.md) | 升级 ZITADEL 版本 |
| [restore.md](restore.md) | 从备份恢复；恢复演练和记录 |
| [key-rotation.md](key-rotation.md) | 轮换密码、令牌、签名密钥 |
| [incident.md](incident.md) | 停机、账号被大量锁定、凭据泄露等故障 |

## 约定

- 生产和预发布在同一台机器上（与 Explore 同机，[architecture.md §4](../design/architecture.md#4-部署)）：生产的检出在 `/opt/identity`，预发布在 `/opt/identity-staging`，各自一份 `deploy/.env`，以下命令在对应目录执行。`docker compose` 一律在 `deploy/` 下运行，`.env` 里的 `COMPOSE_FILE` 让它带上边缘模式，`scripts/` 里的脚本也一样。
- 证书和入口归 Explore 的 Caddy（`/opt/explore`）：站点文件在 `/opt/explore/sites/`，增改后 `docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile`。
- 秘密只在三个地方：`deploy/.env`（`chmod 600`）、`deploy/secrets/`（引导令牌和各应用的环境文件，只有 root 可读）、离线保存的主密钥。都不进仓库。
- 配置只改 `config/`，再跑 `identity-init`；不在管理后台里手工改。管理后台在 `https://<域名>/ui/console`。
- 每次操作先在预发布上做，确认无误再上生产。
- `identity-init` 在服务器上编译一次即可：`go build -o bin/identity-init ./cmd/identity-init`（或在本机编译 `GOOS=linux` 后拷过去）。
