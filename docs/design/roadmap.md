# 路线图

> 状态：I0 完成（S1、S7 随 I1）；I1 中不依赖服务器的部分已在本地完成 · 最近更新：2026-10-11
> 每个阶段只在退出条件全部满足后才进入下一阶段。进度如实标注。

---

## 1. 总览

| 阶段 | 要回答的问题 | 交付 | 状态 |
|---|---|---|---|
| **I0 选型** | ZITADEL 还是 Logto？ | `lab/prototypes/identity-selection/` 的实测记录；结论写进 [architecture.md §3](architecture.md#3-选型) | 完成（2026-10-11）：已选 ZITADEL，S1、S7 待 I1 确认 |
| **I1 预发布** | 在真实域名和真机上能不能用？ | `id.staging.kite.plus`；本仓库的部署和初始化脚本；Explore 预发布环境接入 | 进行中：部署文件、初始化工具、本地开发栈已完成；等服务器和账号 |
| **I2 生产** | 能不能稳定运行、出了事能不能恢复？ | `id.kite.plus`；恢复演练、升级演练；隐私政策和服务条款 | 未开始 |
| **I3 Explore 上线** | 已有的 Explore 用户能不能平滑迁过来？ | Explore 生产环境用 Kite Plus 账号登录 | 未开始 |

I3 完成后，评论服务再开工（[comments roadmap.md](https://github.com/kite-plus/comments/blob/main/docs/design/roadmap.md)）。评论服务真正依赖的是 I2：有了生产可用的身份服务就能开始，I3 只是 Explore 自己的迁移。

---

## 2. I0 选型

在本机用 Docker 各起一套 ZITADEL 和 Logto，按 [architecture.md §3](architecture.md#3-选型) 的 S2–S6 逐项记录。S1 要用真机，S7 要 Apple 开发者账号，都放到 I1。

**环境**：

- 两套身份服务都用固定版本的官方 Compose，数据库各自独立。
- 本地的邮件收集服务（如 Mailpit）接住所有邮件，验证注册、找回密码、邮箱验证码的流程和中英文模板。
- 一个很小的 Go 测试客户端：走一遍 Authorization Code + PKCE，打印 ID Token 的声明，接收 back-channel logout 和用户事件，检查 `prompt=none` 的行为。
- GitHub 登录需要一个测试用的 OAuth App，由组织管理员在 GitHub 上创建；没有的话这一项放到 I1。

**交付**：`lab/prototypes/identity-selection/README.md`，逐项写明两边的实测结果；结论写回 architecture.md §3。

**退出条件**：

- S2–S6 两边都有记录，能按决定规则给出结论；S1、S7 只差真机和 Apple 账号时，结论可以附带「I1 确认」。
- 选中的实现满足 [architecture.md §1](architecture.md#1-各服务对身份服务的要求) 的 R1–R11，做不到的写明替代办法。

**结果**（2026-10-11）：两条都满足。S2–S6 的记录在 lab，结论「已选 ZITADEL（I1 确认）」写进 [architecture.md §3](architecture.md#3-选型)，R1–R11 的落实和替代办法在 [§1.1](architecture.md#11-落实情况)。GitHub 登录因为没有测试用的 OAuth App 放到 I1。

---

## 3. I1 预发布

**需要维护者准备的**：一台香港服务器；`id.staging.kite.plus` 的域名解析；SMTP 服务商的账号；`kite-plus` 组织下预发布用的 GitHub OAuth App；Apple 开发者账号下预发布用的 Services ID 和登录私钥；备份异地存放的位置和 age 密钥对。

**已在本地完成**（2026-10-11）：

- `deploy/`（Compose、nginx、`.env.example`）、`config/`（dev、staging、production）、`scripts/`（备份、恢复、升级）、`cmd/identity-init`，按 [architecture.md §6](architecture.md#6-仓库结构)。
- 本地开发栈 `identity-dev` 和端到端检查 `make dev-test`：代码流程、`sid`、back-channel logout、事件推送和查询、读者自助删号，Explore 和评论服务按 [README](../../README.md#contract-for-kite-plus-services) 的约定对接。
- 生产 Compose 在本机起过一次：issuer、HTTP 跳转、初始化工具对 staging 配置连跑两遍（第二遍无改动）都正常；ACME 申请证书没法在本机验证。
- 限流（第 12 次注册提交返回 429）、账号锁定、恢复演练（[restore.md](../runbooks/restore.md)）、小版本升级演练（v4.18.0 → v4.19.4，约 11 秒不可用）。
- 运维手册初稿：[runbooks](../runbooks/README.md)。

**范围**：

- 本仓库落地：`deploy/`、`config/`、`scripts/`（[architecture.md §6](architecture.md#6-仓库结构)）。对着空实例跑一遍初始化脚本，就得到完整配置。
- 反向代理的限流、TLS、账号锁定；中英文品牌和邮件模板；SPF、DKIM、DMARC。
- Explore 的预发布环境接入：OIDC 登录、ID Token 校验、本地会话记下 `sid`、back-channel logout、用户事件和每小时对账（[Explore accounts.md §1](https://github.com/kite-plus/explore/blob/main/docs/design/accounts.md#1-登录与会话)）。
- 真机测试：iPhone、没有谷歌服务的国产安卓手机，走完注册、登录、找回密码、Passkey、GitHub 登录，完成 S1；在 iPhone 上用隐藏邮箱走一遍 Apple 登录，完成 S7。
- Kite 应用的测试版本走应用登录连预发布环境（[Explore accounts.md §1.1](https://github.com/kite-plus/explore/blob/main/docs/design/accounts.md#11-应用登录-设计中)）：系统登录页、认领的 https 回调、兑换、退出、换账号、跳转删号。

**退出条件**：

- S1–S7 在预发布环境上全部确认；选型结论不再附带条件。
- 从零重建一次预发布环境（新机器、空数据库、跑脚本），配置和原来一致。
- 运维手册初稿：升级、恢复、密钥轮换。

---

## 4. I2 生产

**需要维护者准备的**：`id.kite.plus` 的域名解析；生产用的 GitHub OAuth App 和 Apple Services ID；离线保存主密钥的地方；隐私政策和服务条款的最终审阅。

**范围**：

- 生产实例；issuer 按 R1 定死后再初始化。
- 每日备份，异地保存；监控和告警（[architecture.md §5](architecture.md#5-运维)）。
- 平台管理员账号开启 MFA。
- 中英文的隐私政策和服务条款，在登录页可见。

**退出条件**：

- 从备份恢复到一台空机器，登录、令牌、用户都和原来一致；主密钥离线保存两份。
- 完成一次版本升级演练：先预发布、后生产，记下耗时和回滚步骤。
- [Explore identity-and-comments.md §5](https://github.com/kite-plus/explore/blob/main/docs/design/identity-and-comments.md#5-交付顺序与验收) 里与身份服务有关的安全检查全部完成。

---

## 5. I3 Explore 上线

**范围**（细节见 [Explore accounts.md §1](https://github.com/kite-plus/explore/blob/main/docs/design/accounts.md#1-登录与会话)）：

- 新注册只用 Kite Plus 账号。
- 已有的本站账号保留密码登录，读者用密码登录一次、关联 Kite Plus 账号后关掉。
- 会话 Cookie 改名为 `__Host-explore_session`，记下 `sid`，处理 back-channel logout。
- 处理用户删除、停用事件，每小时对账。
- Kite 应用（iOS、Android）改用应用登录：先发一版改用 Bearer 令牌，再切换登录方式；Apple 登录在生产上可用之后才切（[Explore accounts.md §1.1](https://github.com/kite-plus/explore/blob/main/docs/design/accounts.md#11-应用登录-设计中)）。

**退出条件**：

- 生产环境的新老用户都能登录；关联过的账号订阅和博客归属都在。
- 在 `id.kite.plus` 退出、删除账号，Explore 这边按设计结束会话、删除数据。
- Kite 应用的正式版本能登录、退出、换账号，「删除 Kite Plus 账号」直达删号页面。
- 过渡期的截止时间和到期未关联账号的处理办法已定（[Explore accounts.md §1](https://github.com/kite-plus/explore/blob/main/docs/design/accounts.md#1-登录与会话)）。

---

## 6. 待定事项

| 事项 | 写在哪里 | 什么时候定 |
|---|---|---|
| ~~ZITADEL 还是 Logto~~ | [architecture.md §3](architecture.md#3-选型) | 2026-10-11 定：ZITADEL，S1、S7 待 I1 确认 |
| ~~登录保持时长~~ | [architecture.md §1.1](architecture.md#11-落实情况) 的 R7 | 2026-10-11 定：30 天 |
| ~~反向代理用 Caddy 还是 nginx~~ | [architecture.md §4](architecture.md#4-部署) | 2026-10-11 定：nginx |
| 是否提供 Google 登录 | [architecture.md §2](architecture.md#2-登录方式) | I1 |
| 服务商、线路和机器配置 | [architecture.md §4](architecture.md#4-部署) | I1 开工前 |
| SMTP 服务商 | [architecture.md §4](architecture.md#4-部署) | I1 开工前 |
| 删号时撤销 Apple 令牌的办法（ZITADEL 不做） | [architecture.md §1.1](architecture.md#11-落实情况) 的 R11 | I1，有 Apple 开发者账号后 |
| 账号锁定被滥用时的处理 | [architecture.md §5](architecture.md#5-运维) | I1 观察后 |
| 备份异地存放位置 | [architecture.md §5](architecture.md#5-运维) | I2 开工前 |
| Explore 过渡期的截止时间 | [Explore accounts.md §1](https://github.com/kite-plus/explore/blob/main/docs/design/accounts.md#1-登录与会话) | I3 开工前 |
