# 首次部署

> 适用：在新服务器上第一次部署 `id.staging.kite.plus` 或 `id.kite.plus`。生产的 issuer 一经初始化就不能再改（R1），动手前把域名再核对一遍。

## 准备

- 服务器：2 核 4 GB 起，Docker 和 Compose 插件，80、443 端口对外开放。
- DNS：身份服务的域名指向这台服务器。**Explore 和评论服务的域名也要先解析好**：`identity-init` 创建客户端时，ZITADEL 会解析 back-channel logout 地址，解析不到或解析到内网地址会报 `Errors.Project.App.Blocked.BackchannelLogoutURL`。
- SMTP 账号；`kite.plus` 的 SPF、DKIM、DMARC。
- 一对 age 密钥用于备份（`age-keygen`），私钥离线保存，公钥写进 `.env`。

## 步骤

1. 拉代码，填环境文件：

   ```sh
   git clone https://github.com/kite-plus/identity /opt/identity && cd /opt/identity
   cp deploy/.env.example deploy/.env && chmod 600 deploy/.env
   ```

   按注释填写。`ZITADEL_MASTERKEY` 用 `openssl rand -hex 16` 生成，**立刻离线抄两份**；其余密码用 `openssl rand -hex 24`。第一次可以先用 Let's Encrypt 的测试目录（`ACME_DIRECTORY`）确认证书流程，再换成正式目录。

2. 启动：

   ```sh
   cd deploy && docker compose up -d --wait && cd ..
   chmod 700 deploy/secrets && chmod 600 deploy/secrets/bootstrap/*
   curl -fsS https://<域名>/.well-known/openid-configuration | head -c 200
   ```

   issuer 必须是 `https://<域名>`。`zitadel-init`、`zitadel-setup` 退出码为 0 是正常的。

3. 应用配置（预发布用 `config/staging.yaml`，生产用 `config/production.yaml`）：

   ```sh
   set -a && . deploy/.env && set +a
   bin/identity-init -config config/staging.yaml -token-file deploy/secrets/bootstrap/admin.pat -out deploy/secrets/apps
   bin/identity-init -config config/staging.yaml -token-file deploy/secrets/bootstrap/admin.pat -out deploy/secrets/apps   # 第二遍应输出 no changes
   ```

   `deploy/secrets/apps/explore.env` 和 `comments.env` 交给各自的部署（安全通道，不经聊天工具）。GitHub、Apple 没填时会提示跳过，填好后再跑一遍即可。

4. 管理员：用 `ADMIN_EMAIL` 和初始密码登录 `/ui/console`，按提示改密码、绑定第二因素（运维组织强制 MFA，R9）。然后在「默认设置」里确认 SMTP 能发信（或注册一个测试账号看验证邮件）。

5. 定时任务（root 的 crontab）：

   ```cron
   */5 * * * * cd /opt/identity && set -a && . deploy/.env && set +a && bin/identity-init -config config/staging.yaml -token-file deploy/secrets/bootstrap/admin.pat -only readers >/dev/null
   15 3 * * *  cd /opt/identity && scripts/backup.sh >>/var/log/identity-backup.log 2>&1
   ```

   第一条让新注册的读者能自助删号（architecture.md §1.1 R6），第二条是每日备份，异地复制按 I2 定的存放位置另配。

6. 验收：在浏览器里走一遍注册、验证邮件、登录、退出；`curl -X POST https://<域名>/ui/v2/login/register` 连发 12 次，第 12 次应为 429；外部探测 `https://<域名>/debug/healthz`。

## 记录

| 日期 | 环境 | 版本 | 备注 |
|---|---|---|---|
| 2026-10-11 | 本机，生产 Compose 加端口覆盖 | v4.19.4 | 全部容器健康，issuer 为 `https://id.staging.kite.plus`，HTTP 跳转 HTTPS；`identity-init` 对 staging 配置连跑两遍，第二遍无改动。证书无法在本机申请；本机代理把外部域名解析到 198.18.0.0/15，临时放宽了拒绝列表。顺带在 Let's Encrypt 测试目录注册了一个 ACME 账号（account 351160693），随卷删除，未签发证书 |
