# 升级

> 时限以 [architecture.md §5](../design/architecture.md#5-运维) 为准：严重漏洞 72 小时内升级到生产，其余随每月一次的例行升级。

## 关注什么

- 订阅 [ZITADEL 安全公告](https://github.com/zitadel/zitadel/security/advisories) 和 [Releases](https://github.com/zitadel/zitadel/releases)（GitHub 的 Watch → Custom → Releases、Security alerts）。
- 公告的严重级别为 critical 时，从发布时间起算 72 小时：当天在预发布上升级并验证，第二天上生产。
- 后端镜像版本 `ZITADEL_VERSION` 必须与定制登录源码 `frontend/upstream-version.txt` 一致。先更新上游源码与协议、保留 Kite Plus 页面和账号权限修改、完成回归，再升级后端；登录镜像由本仓库构建，版本用 `IDENTITY_UI_VERSION` 标识。`scripts/upgrade.sh` 会检查源码版本，避免把新后端与旧前端混用。
- 一并检查 nginx、PostgreSQL 镜像的补丁版本（只升补丁号；PostgreSQL 大版本升级另做方案）。前面的 Caddy 是 Explore 的，随 Explore 升级。

## 步骤

1. 读 Release Notes，找迁移说明和不兼容改动；跨大版本（v4 → v5）先在本机用 `make dev-up` 换版本试跑，再按这里做。
2. 预发布（和生产同机，平时停着，先启动）：

   ```sh
   cd /opt/identity-staging && git pull
   (cd deploy && docker compose up -d --wait)
   scripts/upgrade.sh v4.19.5
   ```

   脚本先拉镜像、做一次备份，再改 `.env` 里的版本、`docker compose up -d --wait`（`zitadel-setup` 自动跑迁移），最后检查 issuer 和 `/debug/ready`。
3. 验证：浏览器里登录一次 Explore 预发布和评论预发布；注册一个测试账号，验证邮件是中文（语言调用目标还认得新版本的请求格式）；在管理后台停用再启用一个测试账号，确认两边收到事件；`bin/identity-init ... -config config/staging.yaml` 应输出 no changes。新版本若改了登录界面创建用户的接口（目前是 `AddHumanUser`），语言调用目标要跟着改。
4. 生产：`cd /opt/identity && git pull && scripts/upgrade.sh v4.19.5`，在访问量低的时候做。期间有十几秒登录不可用，已登录的应用不受影响。做完把预发布停掉：`cd /opt/identity-staging/deploy && docker compose stop`。
5. 在下面的记录表里写一行。

## 回滚

ZITADEL 的迁移不能撤销，回滚就是**用升级前那份备份恢复到旧版本**（[restore.md](restore.md)）：把 `.env` 的 `ZITADEL_VERSION` 改回旧版本，`docker compose down`，删掉数据库卷，再用 `scripts/restore.sh` 恢复升级脚本刚做的那份备份。升级之后新产生的数据（注册、改密码）会丢失，所以只在新版本确实不能用时回滚，否则优先等补丁。

## 记录

| 日期 | 环境 | 从 → 到 | 不可用时长 | 备注 |
|---|---|---|---|---|
| 2026-10-11 | 本机演练（`identity-upgrade` 栈） | v4.18.0 → v4.19.4 | 约 11 秒 HTTP 502 | 自动迁移；端到端检查全部通过，配置无改动。演练中发现 nginx 只在启动时解析上游地址，已改为按 Docker DNS 重新解析 |
