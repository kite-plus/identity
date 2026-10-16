# 故障处理

> 先判断影响面：身份服务停机时，已经登录的应用不受影响，只是新登录进不来（[architecture.md §5](../design/architecture.md#5-运维)）。

## 登录不可用

1. 在 `/opt/identity/deploy` 下 `docker compose ps`：哪个容器不健康。`docker compose logs --tail 200 zitadel login proxy`。
2. 常见原因：
   - 数据库满盘或没起来：`df -h`，`docker compose logs db`。
   - 证书过期或申请失败：边缘模式下证书归 Explore 的 Caddy，`cd /opt/explore && docker compose logs caddy | grep -i -E 'id\.kite\.plus|error'`，Cloudflare 的设置见 [architecture.md §4](../design/architecture.md#4-部署)。独立模式下 `docker compose logs proxy | grep -i acme`，80 端口必须对外开放，DNS 指向本机。
   - 容器都健康，浏览器却是 502：Caddy 找不到上游。`docker network inspect kite-edge` 里要有身份服务的 proxy（别名 `identity`）和 Explore 的 caddy；Explore 的 `docker-compose.override.yaml` 丢了时 caddy 会掉出 kite-edge。
   - 登录界面令牌过期（`login-client.pat`）：登录页报权限错误；按 [key-rotation.md](key-rotation.md) 换新。
   - 升级后出问题：先查 Release Notes，不行就按 [upgrade.md](upgrade.md) 回滚。
3. 恢复后在外部探测确认 `/debug/healthz`，并在这里写一行记录。

## 登录被限流或被锁

密码错误不再锁账号（[architecture.md §5](../design/architecture.md#5-运维)），读者登不进来多半是限流：

1. `docker compose logs proxy | grep -c ' 429 '`，再看是单个 IP（`limiting requests, excess ... by zone "identity_mail"`）还是合计（`zone "identity_global"`）。日志里的客户端要是全成了 Cloudflare 的地址或 `172.30.10.x`，是读者地址没传过来：查 Explore `.env` 的 `EXPLORE_CDN_RANGES`（Cloudflare 改过地址段）和 `sites/identity.caddy` 里的 `header_up X-Forwarded-For {client_ip}`。
2. 单个 IP：多半是攻击或脚本，等它停下即可；共用出口的读者（学校、公司、运营商 NAT）被误伤时，调高 `deploy/nginx/limits.conf` 里 `identity_mail` 的速率，`docker compose exec proxy nginx -s reload`。
3. 合计限流被打满：说明正在被分布式猜密码，真实读者也会收到 429。短时间可以接受；持续时在 `limits.conf` 调高 `identity_global` 或针对攻击来源加 `deny`，事后在 architecture.md 记下调整。
4. 账号真的被锁（`USER_STATE_LOCKED`）只会来自管理员操作，或输对密码后连续输错 10 次验证码：确认是本人后 `curl -X POST $API/v2/users/<id>/unlock -H "Authorization: Bearer $T"`。

## 凭据泄露

| 泄露的是 | 立即做 |
|---|---|
| `admin.pat` 或管理员账号 | 管理后台删除该 PAT／重置管理员密码和第二因素；检查「事件」里最近的配置改动；按 key-rotation 轮换所有应用的客户端密钥和签名密钥 |
| 某个应用的环境文件 | 轮换该应用的客户端密钥、签名密钥、对账令牌（key-rotation.md） |
| 主密钥 | 视为库内加密数据泄露：轮换所有能轮换的密钥，评估是否要重建实例（会改变所有 `sub`，代价极大） |
| 读者账号被盗 | 管理后台结束该用户的会话（`DELETE /v2/sessions/<id>`，会触发 back-channel logout）、重置密码 |

## 推送丢失

事件推送只投递一次（[README](../../README.md#user-event-webhooks)）。应用停机期间的事件靠每小时对账补上，不需要人工处理；如果某个应用的对账也失败（令牌过期），按 key-rotation 换令牌后它会自己补齐。

## 发信失败

读者收不到验证邮件：先看 `docker compose logs zitadel | grep smtp`。`535 Authentication credentials invalid` 说明 ZITADEL 存的密码不对：v4.19.4 单独改 SMTP 密码不会生效，`identity-init` 在密码变化时删除并重建发信配置，换 key 一律用服务器上的 `kite-set-smtp`。再到 Resend 的 Emails 页看有没有发出、是否被退回；QQ、163 拒收时检查 `id.kite.plus` 的 DKIM 和 `kite.plus` 的 DMARC。读者说点了「忘记密码」却没收到信，也可能是这个邮箱还没有 Kite Plus 账号：为了不暴露账号是否存在，页面照样显示已发送。

## 记录

| 日期 | 环境 | 现象 | 原因 | 处理 |
|---|---|---|---|---|
| | | | | |
