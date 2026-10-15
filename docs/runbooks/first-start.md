# 首次部署

> 适用：在与 Explore 同机的服务器上第一次部署 `id-staging.kite.plus` 或 `id.kite.plus`，用边缘模式（[architecture.md §4](../design/architecture.md#4-部署)）。生产的 issuer 一经初始化就不能再改（R1），动手前把域名再核对一遍。没有共用入口的机器用独立模式，见文末。

## 准备

- 服务器：与 Explore 同机的美国 RackNerd 服务器（地址不写进仓库）。Explore 在 `/opt/explore` 运行，它的 Caddy 占 80 和 443，`.env` 的 `EXPLORE_CDN_RANGES` 列着 Cloudflare 的地址段。另外装好 git、age、rclone（异地备份），以及编译 `identity-init` 的 Go 1.26（或在本机交叉编译后拷过去，见 [README](README.md)）。
- DNS（Cloudflare，开代理）：身份服务的域名。**Explore 和评论服务的域名也要先有记录**：`identity-init` 创建客户端时，ZITADEL 会解析 back-channel logout 地址，解析不到或解析到内网地址会报 `Errors.Project.App.Blocked.BackchannelLogoutURL`。预发布的二级子域和其他 Cloudflare 设置见 architecture.md §4。
- SMTP 账号；`kite.plus` 的 SPF、DKIM、DMARC。
- 一对 age 密钥用于备份（`age-keygen`），私钥离线保存，公钥写进 `.env`。生产另要异地存放的位置：`rclone config` 建一个 remote（SFTP 或对象存储）。

## 一次性：共用入口

每台服务器只做一次，评论服务以后直接用。

1. 建共用网络：

   ```sh
   docker network create --subnet 172.30.10.0/24 kite-edge
   ```

2. 让 Explore 的 Caddy 接上它、读入站点目录，按 [Explore project-layout.md §10](https://github.com/kite-plus/explore/blob/main/docs/design/project-layout.md#10-部署)「与同机其他服务共用入口」：确认 `Caddyfile` 末尾有 `import /etc/caddy/sites/*.caddy`、compose 给 caddy 挂了 `./sites`（服务器上的旧版本手工加上这两处），建 `sites/`，放 `docker-compose.override.yaml`，再重建 Caddy：

   ```sh
   cd /opt/explore
   mkdir -p sites
   docker compose config --format json | grep -c kite-edge   # 大于 0：override 已生效
   docker compose up -d caddy
   docker compose logs --tail 50 caddy | grep -i 'import glob'   # sites 还空着，只有这条警告
   ```

   重建 Caddy 的几秒里整台机器的站点都不通，挑访问少的时候做。

## 步骤

1. 拉代码，填环境文件：

   ```sh
   git clone https://github.com/kite-plus/identity /opt/identity && cd /opt/identity
   cp deploy/.env.example deploy/.env && chmod 600 deploy/.env
   ```

   按注释填写，并去掉 `COMPOSE_FILE=docker-compose.yaml:docker-compose.edge.yaml` 前的 `#`（边缘模式）。`ZITADEL_MASTERKEY` 用 `openssl rand -hex 16` 生成，**立刻离线抄两份**；其余密码用 `openssl rand -hex 24`。`ACME_*` 保持原样，边缘模式用不到。生产填上 `BACKUP_REMOTE`（rclone 的 `remote:路径`）。

   **预发布**是另一份检出，放在 `/opt/identity-staging`：`IDENTITY_DOMAIN` 填预发布的域名，再去掉 `COMPOSE_PROJECT_NAME=identity-staging`、`IDENTITY_EDGE_ALIAS=identity-staging`、`IDENTITY_SUBNET=172.31.251.0/24`、`IDENTITY_ENV=staging` 四行的注释；`BACKUP_DIR=/var/backups/identity-staging`，`BACKUP_REMOTE` 留空（脚本以 `.env` 为准）。以下步骤在 `/opt/identity-staging` 里照做，站点文件换成 `identity-staging.caddy`，配置文件换成 `config/staging.yaml`，不配定时任务。

2. 启动：

   ```sh
   cd deploy && docker compose up -d --wait && cd ..
   chmod 700 deploy/secrets && chmod 600 deploy/secrets/bootstrap/*
   ```

   `zitadel-init`、`zitadel-setup` 退出码为 0 是正常的；`docker compose ps` 里 proxy 没有端口映射。

3. 交给 Caddy：

   ```sh
   cp deploy/edge/identity.caddy /opt/explore/sites/
   (cd /opt/explore && docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile)
   (cd /opt/explore && docker compose logs --tail 50 caddy) | grep -i 'certificate obtained'
   curl -fsS https://<域名>/.well-known/openid-configuration | head -c 200
   ```

   issuer 必须是 `https://<域名>`。站点文件写错时 reload 报错，旧配置照常运行 `[EV]`。证书签不下来先看 architecture.md §4 的 Cloudflare 一条（Always Use HTTPS、二级子域）。

4. 应用配置（生产用 `config/production.yaml`，预发布用 `config/staging.yaml`）：

   ```sh
   set -a && . deploy/.env && set +a
   bin/identity-init -config config/production.yaml -token-file deploy/secrets/bootstrap/admin.pat -out deploy/secrets/apps
   bin/identity-init -config config/production.yaml -token-file deploy/secrets/bootstrap/admin.pat -out deploy/secrets/apps   # 第二遍应输出 no changes
   ```

   请求经 Cloudflare 和 Caddy 到 ZITADEL。`deploy/secrets/apps/explore.env` 和 `comments.env` 交给各自的部署（安全通道，不经聊天工具）。GitHub、Apple 没填时会提示跳过，填好后再跑一遍即可。

5. 管理员：用 `ADMIN_EMAIL` 和初始密码登录 `/ui/console`，按提示改密码、绑定第二因素（运维组织强制 MFA，R9）。然后在「默认设置」里确认 SMTP 能发信（或注册一个测试账号看验证邮件）。

6. 定时任务（只在生产）。服务器的 Ubuntu 没装 cron，用 systemd 定时器，每个任务一对 `/etc/systemd/system/<名字>.service`（`Type=oneshot`，`ExecStart=/bin/sh -c '<命令>'`）和 `<名字>.timer`（`OnCalendar=<时间>`、`Persistent=true`），再 `systemctl enable --now <名字>.timer`：

   | 名字 | 时间 | 命令 |
   |---|---|---|
   | `kite-identity-readers` | `*:0/5` | `cd /opt/identity && set -a && . deploy/.env && set +a && exec bin/identity-init -config config/production.yaml -token-file deploy/secrets/bootstrap/admin.pat -only readers` |
   | `kite-identity-backup` | `*-*-* 03:15:00 UTC` | `cd /opt/identity && exec scripts/backup.sh` |
   | `kite-comments-backup` | `*-*-* 03:25:00 UTC` | `cd /opt/comments && exec scripts/backup.sh` |

   第一个让新注册的读者能自助删号（architecture.md §1.1 R6），后两个是每日备份，`BACKUP_REMOTE` 设了就同时复制到异地。`systemctl list-timers 'kite-*'` 看下次运行时间，`journalctl -u kite-identity-backup` 看日志。

7. 验收：
   - 在中文浏览器里走一遍注册，验证邮件应为中文；把登录界面切到 English 再注册一个，邮件应为英文（新读者的语言，architecture.md §1.1 R8）。
   - 登录、退出；5 分钟后在 `https://<域名>/ui/v2/login/account/delete` 能删除账号；旧地址 `/ui/console/users/me` 应跳到 `/ui/v2/login/account`。
   - 在自己的电脑上 `curl -X POST https://<域名>/ui/v2/login/register` 连发 12 次，第 12 次应为 429；服务器上 `cd deploy && docker compose logs --tail 20 proxy` 里这些请求的客户端是自己电脑的公网地址，不是 Cloudflare 或 `172.30.10.x` 的地址，说明读者地址一路传到了 nginx。`https://<域名>/ui/login/loginname` 应为 404。
   - 别的 Docker 网络够不着身份服务的 nginx（下面的命令必须超时失败）。Docker 默认丢弃不同网络之间的流量；Docker 的 iptables 被关掉时这里会通，别的网络上的容器（比如抓任意网址的 Explore worker）就能直接连到 nginx，还能冒充读者的地址：

     ```sh
     ip=$(docker inspect -f '{{(index .NetworkSettings.Networks "kite-edge").IPAddress}}' identity-proxy-1)
     docker run --rm --network explore_default caddy:2-alpine wget -T 3 -qO- --header 'Host: <域名>' "http://$ip/debug/healthz"
     ```

   - 外部探测 `https://<域名>/debug/healthz`。

8. 预发布用完就停：`cd /opt/identity-staging/deploy && docker compose stop`，下次 `docker compose up -d --wait`。站点文件留在 `sites/` 里，停着时那个域名返回 502。

## 独立模式

没有共用入口的机器：不设 `COMPOSE_FILE`，80、443 要空着并对外开放，DNS 指向这台机器；nginx 用 ACME 模块自己申请证书，第一次可以先用 Let's Encrypt 的测试目录（`ACME_DIRECTORY`）确认证书流程，再换成正式目录。没有第 3 步，其余相同。

## 记录

| 日期 | 环境 | 版本 | 备注 |
|---|---|---|---|
| 2026-10-11 | 本机，生产 Compose 加端口覆盖 | v4.19.4 | 全部容器健康，issuer 为 `https://id.staging.kite.plus`（当时的预发布域名），HTTP 跳转 HTTPS；`identity-init` 对 staging 配置连跑两遍，第二遍无改动。证书无法在本机申请；本机代理把外部域名解析到 198.18.0.0/15，临时放宽了拒绝列表。顺带在 Let's Encrypt 测试目录注册了一个 ACME 账号（account 351160693），随卷删除，未签发证书 |
| 2026-10-11 | 本机按服务器布局演练边缘模式：`kite-edge` 网络，Explore 的 Caddy 单独运行（`*.localhost` 站点名、Caddy 内部证书、没有 ACME），身份服务和评论服务各有生产、预发布两个项目 | v4.19.4 | issuer 为 `https://id.localhost` 和 `https://id.staging.localhost`；登录页、控制台标题、`/ui/login/` 404 正常；`identity-init` 经 Caddy 连跑两遍，第二遍无改动；新用户语言调用目标照常（kite-edge 上的容器调它得 403）。模拟 CDN 的容器替读者 203.0.113.10 连发 12 次注册提交，第 12 次 429，读者 203.0.113.20 不受影响，读者自己写在左边的地址不被采用；直连 Caddy、每次自带不同 `X-Forwarded-For` 的客户端仍按它自己的地址计数。发现 `return 302` 在边缘模式下成了 http 的绝对地址，已改成相对地址。OrbStack 会在不同网络之间转发（kite-edge 外的容器经网关 172.30.10.1 够到了 nginx），Linux 上的 Docker 默认不转发，上线时按第 7 步验证。备份写出、age 解密、经 rclone 替身复制到异地并清理过期副本 |
| 2026-10-11 | **生产**，与 Explore 同机的服务器，边缘模式 | v4.19.4 | `kite-edge` 网络建好，Explore 的 Caddy 加上 `sites/` 的引入和挂载、接上 kite-edge（重建时几秒不通）；`id.kite.plus` 证书由 Caddy 签发（TLS-ALPN 在 Cloudflare 后失败，HTTP-01 成功），issuer 正确；`identity-init` 第一遍 47 处改动、第二遍无改动，GitHub、Apple 和 SMTP 跳过（SMTP 由维护者用服务器上的 `kite-set-smtp` 写入密码后生效）。验收：Explore 的网络连不到 nginx；nginx 日志里是读者的公网地址；经 Cloudflare 每次请求约 1 秒，串行 12 次会被令牌补回，并发 16 次里 11 次 429；`/ui/login/loginname` 404。备份 age 加密、在维护者电脑上用私钥解开核对过。服务器没有 cron，定时任务改用 systemd 定时器 |
