# Kite Plus 账号 总体设计

> 状态：规划草案，实现方式在 I0 选型后定案 · 最近更新：2026-10-10
> 文档约定沿用 [Kite 设计文档](https://github.com/kite-plus/kite/blob/main/docs/design/README.md)：正文中文，专有名词保留英文；`[设计中]` 表示实现前还可调整，`[待定]` 表示明确推迟决策；`[EV]` 表示结论有实测或既有项目佐证。
> 各应用怎么接入（OIDC 流程、`(iss, sub)`、本地会话）是跨项目约定，写在 [Explore identity-and-comments.md §3](https://github.com/kite-plus/explore/blob/main/docs/design/identity-and-comments.md#3-协议与数据)。本文只写身份服务本身。

---

## 0. 定位

- **`id.kite.plus` 是 Kite Plus 所有服务共用的登录**，是基础设施：Explore 先接入（[roadmap.md](roadmap.md) 的 I3），评论服务之后接入，以后的服务也都走它。读者看到的名字是「Kite Plus 账号」。
- **不自己写身份服务**：自托管一个开源实现，不修改它的代码。自己写要维护认证协议、签名密钥、账号恢复、密码和 Passkey、漏洞响应，一个人维护不起。
- **本仓库（`kite-plus/identity`）只放**部署配置、可重复执行的初始化脚本、备份与恢复脚本、运维手册、品牌素材和邮件模板。
- **它先于一切上线**：各服务的用户键都是它签发的 `(iss, sub)`，issuer 一经公开就不能换，所以 §1 的要求必须在第一次初始化之前全部满足。

---

## 1. 各服务对身份服务的要求

| # | 要求 | 谁需要 | 为什么 |
|---|---|---|---|
| R1 | issuer 是 `https://id.kite.plus`，第一次初始化前定死 | 全部 | 写在每个 ID Token 里，所有服务按 `(iss, sub)` 认人。ZITADEL 的 `ExternalDomain` 初始化后再改要重装 |
| R2 | `sub` 在所有应用里一致，不用 pairwise | 全部 | 同一个人在 Explore 和评论服务里是同一个用户键 |
| R3 | 机密客户端使用 Authorization Code + PKCE，回调地址精确登记；第一方应用不弹授权确认 | 全部 | 标准做法；少一步确认 |
| R4 | ID Token 带 `sid`，支持 back-channel logout | Explore、评论 | 读者在 `id.kite.plus` 退出时，各应用结束这次登录产生的会话 |
| R5 | 用户删除、停用、锁定、恢复时推送带签名的事件，并提供可查询的事件接口 | Explore、评论 | 删除账号要跨服务清理；推送会丢，各应用要能对账 |
| R6 | 读者自助：注册、邮箱验证、找回密码、Passkey、关联 GitHub、换邮箱、删除账号 | 全部 | 不让维护者手工处理账号 |
| R7 | 登录保持时间足够长，重启浏览器后仍然有效 | 评论 | 评论组件点「登录」时弹窗一闪而过，靠的就是它（[comments login.md §1](https://github.com/kite-plus/comments/blob/main/docs/design/login.md#1-目标与边界)） |
| R8 | 中英文界面和邮件，Kite Plus 的品牌 | 全部 | 读者以中文为主 |
| R9 | 平台管理员必须开启 MFA | 全部 | 管理员账号被盗等于全部账号被盗 |
| R10 | 能被限流：登录、注册、找回密码、发验证码这几类地址 | 全部 | 防撞库、防邮件轰炸 |
| R11 | 提供 Apple 登录，删除账号时撤销 Apple 的令牌 | Kite 应用 | 登录页有 GitHub 登录，App Store 审核指南 4.8 要求应用同时提供一种只收集姓名和邮箱、允许隐藏邮箱的登录方式；用 Apple 登录的应用删号时要调用 Apple 的接口撤销令牌（§2） |

不需要：授权确认页、pairwise `sub`、SAML、多租户组织、原生应用客户端。Kite 应用经 Explore 服务端登录，身份服务里只有 Explore 这个机密客户端（[Explore accounts.md §1.1](https://github.com/kite-plus/explore/blob/main/docs/design/accounts.md#11-应用登录-设计中)）。

应用用到的声明：`sub`、`sid`、`email`、`email_verified`、`name`、`preferred_username`、`picture`、`locale`。

---

## 2. 登录方式

| 方式 | 决定 | 说明 |
|---|---|---|
| 邮箱加密码 | 做 | 邮箱必须验证；ZITADEL 要在登录容器上开启 `EMAIL_VERIFICATION` |
| Passkey | 做，推荐 | 绑定域名，钓不走。国内不少安卓手机没有谷歌服务，用不了 |
| GitHub | 做 | 技术博客的读者大多有；可以不设密码直接用 GitHub 注册 |
| 邮箱验证码 | 看选型 | 没有 Passkey 的读者最省事的方式。ZITADEL 不支持作为首要因素，Logto 支持（§3） |
| Apple | 做，Kite 应用切换到统一登录之前上线 | R11。读者选择隐藏邮箱时，拿到的是 `privaterelay.appleid.com` 的转发地址，发信域名要先在 Apple 开发者账号里登记，否则验证邮件和通知收不到；Apple 只在第一次授权时给出名字，之后要靠身份服务自己存的资料 |
| Google | `[待定]` | 国内用不了，对海外读者有用 |
| 微信、QQ | 不做 | 微信网站登录只对企业等组织开放，还要求已备案的域名；QQ 互联个人可以申请，但同样要求网站已备案（[comments architecture.md §5](https://github.com/kite-plus/comments/blob/main/docs/design/architecture.md#5-部署地与合规)）。两者的 OAuth 也都不是标准实现 |

- 一个人的多种登录方式在身份服务里关联到同一个账号。已经分成两个账号的不提供合并：两个 `(iss, sub)` 在各应用里就是两个用户。
- 显示名：各应用首次登录时取 `name`，没有时取 `preferred_username`，之后由各应用自己管理。

---

## 3. 选型

两个候选，都自托管、都能满足 §1 的大部分要求。以下是 2026-09 查到的情况，I0 实测核对（[roadmap.md §2](roadmap.md#2-i0-选型)）。

| | ZITADEL v4 | Logto |
|---|---|---|
| 许可证 | 核心 AGPL-3.0，登录界面 MIT；不修改、作为独立服务运行，对本项目没有影响 | MPL-2.0 |
| 技术栈 | Go；接口和登录界面（Next.js）两个容器；只支持 PostgreSQL | Node；PostgreSQL |
| OIDC | 机密客户端加 PKCE、`prompt=none`、back-channel logout（4.11 起）；`sub` 是整个实例统一的用户 ID；没有授权确认页 | I0 核对 R2–R4 |
| 登录方式 | 密码、可作为首要因素的 Passkey、GitHub 等社交登录；**不支持邮箱验证码或魔法链接作为首要因素** | 原生支持邮箱和短信验证码登录；官方有微信、QQ 等国内平台的连接器 |
| 中文 | 登录界面和邮件的简体中文完整；注册表单要求填姓和名 | I0 核对 |
| 用户事件 | Actions v2 按事件推送 Webhook（带 HMAC 签名），投递 5 分钟内失败即丢弃；另有事件查询 API | I0 核对 R5 |
| 防滥用 | 没有内置限流和验证码；有账号锁定 | I0 核对 |
| 安全公告 | 2026 年至 9 月已发 36 个，其中严重 5 个，多为登录流程中的账号接管 | I0 查近两年的记录 |
| FedCM | 不支持 | 不支持 |
| Apple 登录 | 有 Apple 身份提供方；删号时是否撤销 Apple 令牌，I0 查文档、I1 实测（S7） | 有 Apple 连接器；同左 |

**对比标准**（I0 按这张表逐项记录）：

| # | 标准 | 怎么测 |
|---|---|---|
| S1 | 没有 Passkey 的国内安卓读者能不能方便地注册、登录 | 用一台没有谷歌服务的华为或小米手机，走完注册、登录、找回密码（I1 在预发布环境上用真机测） |
| S2 | 中文注册体验 | 表单要填哪些字段；邮箱验证；用 GitHub 注册而 GitHub 资料里没有名字或公开邮箱时怎么处理 |
| S3 | §1 的 R2–R5、R7 | 用一个小的 Go 测试客户端逐项验证：`prompt=none`、`prompt=select_account`（应用里「换一个账号」要用）、ID Token 里的 `sid`、back-channel logout（包括身份服务会话自然过期时发不发）、用户删除和停用事件的推送与查询、登录保持时长 |
| S4 | 运维 | 内存和 CPU 占用；从备份恢复到空机器；跨一个小版本升级 |
| S5 | 安全 | 近两年安全公告的数量、严重程度和修复速度；自带的限流、锁定和验证码 |
| S6 | 自定义域名、品牌、中英文邮件模板 | 按 `id.staging.kite.plus` 配一遍 |
| S7 | Apple 登录（R11） | 用隐藏邮箱注册，确认能收到验证邮件；第一次授权后的资料怎么存；删号时 Apple 令牌有没有撤销。要 Apple 开发者账号，放到 I1 |

**决定规则**：ZITADEL 在 S1 上的缺口可以接受（比如 GitHub 加密码已经够用），就用 ZITADEL；否则，Logto 在 S2–S6 上都不比 ZITADEL 差时改用 Logto。结论写回本节，把「首选」改成「已选」。

**首选 ZITADEL**：OIDC 能力最完整，和本组织的 Go 技术栈一致。对比 Logto，是因为邮箱验证码登录对国内读者很重要，而 ZITADEL 没有。

另外两条路：自己写，不采用（§0）；托管的身份服务上线快，但要持续付费，数据放在哪、以后能不能迁走都由供应商决定，只作为运维上的备选。

---

## 4. 部署

- **地点**：香港，和以后的评论服务同一台服务器。不需要备案，到国内三网的延迟在可接受范围。账号是个人信息，放在境外涉及的出境规则，和评论服务一样，见 [comments architecture.md §5](https://github.com/kite-plus/comments/blob/main/docs/design/architecture.md#5-部署地与合规)。服务商和线路 `[待定]`，I1 开工前定。
- **环境**：生产 `id.kite.plus`；预发布 `id.staging.kite.plus`，独立实例、独立数据库。每次升级先在预发布上跑。
- **组成**（Docker Compose）：
  - 反向代理：终止 TLS，对登录、注册、找回密码、发验证码的地址限流（R10）。Caddy 的限流要自己编译插件，nginx 自带；用哪个 `[待定]`，I1 定。
  - 身份服务：ZITADEL 是接口和登录界面两个容器，两个镜像版本必须一致。
  - PostgreSQL：身份服务独占一个库和一个角色。以后评论服务上同一台机器，用另外的库和角色。
- **资源**：只跑身份服务时 2 核 4 GB 起步（ZITADEL 自己建议至少 2 GB 内存，密码哈希会让 CPU 短时冲高）；评论服务加入后再扩 `[待定]`。
- **发信**：SMTP 服务商 `[待定]`；发件人 `Kite Plus 账号 <noreply@kite.plus>`；`kite.plus` 配好 SPF、DKIM 和 DMARC，否则 QQ 邮箱、163 邮箱很可能直接拒收。
- **GitHub 登录**：在 `kite-plus` 组织下注册 OAuth App，生产和预发布各一个，回调指向各自的 `id` 域名。
- **Apple 登录**：在 Apple 开发者账号下建 Services ID 和用于登录的私钥，生产和预发布各一个 Services ID，回调指向各自的 `id` 域名；`noreply@kite.plus` 所在的发信域名登记到 Apple 的邮件转发服务（§2）。
- **密钥**：
  - ZITADEL 的主密钥是 32 个字符，初始化后不能改；丢了，库里加密的数据就再也解不开。离线保存两份。
  - 数据库密码、SMTP 凭据、GitHub 客户端密钥、Apple 登录私钥、事件签名密钥放在服务器的环境文件里，不进仓库。

---

## 5. 运维

- **升级**：订阅身份服务的安全公告。**严重漏洞 72 小时内升级到生产，其余随每月一次的例行升级**；都先在预发布环境升级。本组织所有提到身份服务升级时限的地方都以这一条为准。
- **备份**：每天 `pg_dump` 身份服务的库，加密后异地保存 30 天。每个阶段结束做一次恢复到空机器的演练。
- **监控**：外部探测 `/healthz`，探测点包括国内三网；对登录失败率、发信失败、证书到期、磁盘和备份结果设告警。
- **防滥用**：反向代理限流（§4）；开启账号锁定；登录失败的延迟保持默认。
- **管理员**：平台管理员账号强制 MFA；日常不用管理员账号登录各应用。
- **隐私政策和服务条款**：账号存着邮箱等个人信息，生产上线（I2）前必须有，中英文。
- **故障**：身份服务停机时，已经登录的应用不受影响，因为各应用都有自己的会话；只是新登录进不来。

---

## 6. 仓库结构

```text
identity/
├── deploy/        docker-compose.yaml、反向代理配置、.env.example
├── config/        实例设置、登录策略、应用客户端、事件推送、品牌、邮件模板
├── scripts/       初始化（把 config/ 应用到实例）、备份、恢复、升级
├── branding/      Logo、配色
└── docs/
    ├── design/    本文和路线图
    └── runbooks/  升级、恢复、密钥轮换、故障处理
```

初始化脚本要能重复执行：对着一个空实例跑一遍，就得到和生产一样的配置。改配置先改 `config/`，再跑脚本，不在管理后台里手工点。

---

## 7. 参考

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)、[OpenID Connect Back-Channel Logout 1.0](https://openid.net/specs/openid-connect-backchannel-1_0.html)。
- ZITADEL：[Docker Compose 部署](https://zitadel.com/docs/self-hosting/deploy/compose)、[自定义域名](https://zitadel.com/docs/self-hosting/manage/custom-domain)、[back-channel logout](https://zitadel.com/docs/guides/integrate/back-channel-logout)、[Actions v2](https://zitadel.com/docs/guides/integrate/actions/usage)、[安全公告](https://github.com/zitadel/zitadel/security/advisories)。
- Logto：[连接器](https://github.com/logto-io/logto/tree/master/packages/connectors)。
