# 密钥轮换

> 每年例行轮换一次；怀疑泄露时立即轮换对应的那一项（[incident.md](incident.md)）。下面的 API 调用都用 `deploy/secrets/bootstrap/admin.pat`：
>
> ```sh
> T=$(cat deploy/secrets/bootstrap/admin.pat); API=https://<域名>
> ```

| 什么 | 能否轮换 | 怎么做 | 影响 |
|---|---|---|---|
| 主密钥 `ZITADEL_MASTERKEY` | **不能** | 只能离线妥善保存。泄露时库里加密的数据视为泄露，轮换下面所有能轮换的密钥 | — |
| 各应用的事件签名密钥 | 能，立即生效 | `curl -X POST $API/v2/actions/targets/<id> -H "Authorization: Bearer $T" -d '{"expirationSigningKey":"0s"}'`（v4.19.4 只接受 `0s`，没有宽限期），再跑 `identity-init` 写出新的环境文件，交给应用部署 | 应用换上新密钥前，推送验签失败、被丢弃，靠每小时对账补上 |
| 各应用的客户端密钥 | 能，立即失效 | 从 `secrets/apps/state.json` 删掉该应用的 `clientSecret`，跑 `identity-init`（它会重新生成），把新的环境文件交给应用 | 换上之前该应用新登录失败；已登录的会话不受影响 |
| 各应用的对账令牌 | 能 | 从 `state.json` 删掉该应用 `machines` 里的 `token`，跑 `identity-init`：签发新令牌并吊销旧的。令牌到期前 30 天 `identity-init` 也会自动换新 | 换上之前对账失败，推送不受影响 |
| OIDC 签名密钥 | 能 | REST 网关在 v4.19.4 上没有这组接口，用 connect 协议（`POST`，JSON）：`$API/zitadel.webkey.v2.WebKeyService/CreateWebKey`（`{"rsa":{}}`）建新密钥 → 等一小时让各应用的 JWKS 缓存刷新 → `.../ActivateWebKey`（`{"id":"<新 id>"}`）→ 等 ID Token 有效期（12 小时）过去后 `.../DeleteWebKey`（`{"id":"<旧 id>"}`）；`.../ListWebKeys` 查看状态 | 无，应用按 JWKS 自动取新公钥 |
| 登录界面 Cookie 签名密钥 | 能 | `LOGIN_SESSION_COOKIE_SECRET=新,旧`（逗号分隔，第一个签名、都能验证），`docker compose up -d login`；过 30 天去掉旧值 | 无 |
| 引导令牌 `admin.pat` | 能 | 管理后台给 `identity-init` 机器用户新建 PAT，写进 `deploy/secrets/bootstrap/admin.pat`，删掉旧 PAT。到期时间是 `.env` 的 `INIT_PAT_EXPIRATION`，到期前必须换 | 无 |
| 登录界面令牌 `login-client.pat` | 能 | 同上，给 `login-client` 机器用户新建 PAT，写进 `login-client.pat`，`docker compose restart login`，再删旧 PAT | 重启时几秒登录不可用 |
| 数据库密码 | 能 | `docker compose exec db psql -U postgres -c "ALTER ROLE zitadel PASSWORD '…'"`，改 `.env`，`docker compose up -d` | 重启时几秒不可用 |
| SMTP、GitHub、Apple 凭据 | 能 | 在服务商处重新生成，改 `.env`，跑 `identity-init`（只在凭据变化时更新） | 无 |
| 备份用的 age 密钥 | 能 | 新密钥对，改 `BACKUP_AGE_RECIPIENT`；旧私钥保留到旧备份全部过期（30 天） | 无 |
