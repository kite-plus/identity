# 恢复与恢复演练

> 备份怎么做、保存多久见 [architecture.md §5](../design/architecture.md#5-运维)。每个阶段结束做一次恢复到空机器的演练，记在本文末尾。

## 备份里有什么

`scripts/backup.sh` 生成一个 age 加密的 tar 包：

| 文件 | 内容 |
|---|---|
| `MANIFEST` | ZITADEL 版本、时间、环境 |
| `zitadel.dump` | `pg_dump -Fc` 的 zitadel 库 |
| `bootstrap/` | `admin.pat`、`login-client.pat` |

**不含主密钥**，也不含 `deploy/.env`。没有主密钥，库里加密的数据（签名密钥、客户端密钥、SMTP 密码等）解不开。

## 恢复到一台空机器

1. 按 [first-start.md](first-start.md) 准备服务器、共用入口和 DNS，`deploy/.env` 里填**原来的** `ZITADEL_MASTERKEY` 和 `IDENTITY_DOMAIN`、**和备份相同的** `ZITADEL_VERSION`；数据库密码可以重新生成。换机器（比如以后迁到香港）也是这一套：恢复、验证，再把 DNS 指到新机器。
2. 恢复（需要 age 私钥）：

   ```sh
   scripts/restore.sh /path/identity-production-20261011T031500Z.tar.age ~/age-backup.key
   ```

   脚本只往空库里恢复：先起 PostgreSQL，建 `zitadel` 角色和库，`pg_restore`，放回引导令牌，再 `docker compose up -d --wait`。
3. 验证：issuer 正确；`/oauth/v2/keys` 的 `kid` 和原来一致；用一个测试账号登录 Explore；`bin/identity-init -config config/production.yaml ...` 应输出 no changes（各应用的客户端密钥、签名密钥和对账令牌都在库里，环境文件不用重发）。
4. 版本不同时：先用备份的版本恢复，再按 [upgrade.md](upgrade.md) 升级。

## 演练记录

| 日期 | 环境 | 备份大小 | 恢复耗时 | 结果 |
|---|---|---|---|---|
| 2026-10-11 | 本机：`identity-dev` → 新的 `identity-restore` 栈（另一套端口，新的数据库密码，同一主密钥） | 420 KB | 26 秒（含启动） | 通过：签名密钥 `kid` 相同；8 个用户都在；用原来的客户端密钥换令牌、原来的对账令牌查事件、原来的登录界面令牌建会话，`make dev-test` 的 7 项检查全部通过（用 `E2E_OUT_DIR` 指向恢复的栈） |
