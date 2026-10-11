# Kite Plus 账号 总体设计

> 状态：I0 已完成，选定 ZITADEL（S1、S7 待 I1 确认）；本地开发环境、初始化工具和部署文件已就绪 · 最近更新：2026-10-11
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

### 1.1 落实情况

ZITADEL v4.19.4 上的做法。`[EV]` 表示本仓库的 `make dev-test` 或 I0 实测过（[lab identity-selection](https://github.com/kite-plus/lab/blob/main/prototypes/identity-selection/README.md)），接入细节和报文格式在 [README](../../README.md#contract-for-kite-plus-services)。

| # | 怎么满足 | 状态 |
|---|---|---|
| R1 | `ExternalDomain` 在部署的 `.env` 里，第一次启动前写死 | I2 初始化时落实 |
| R2 | `sub` 是实例统一的用户 ID | `[EV]` |
| R3 | 每个应用一个机密客户端，`client_secret_basic`，带 PKCE 时强制校验；ZITADEL 没有授权确认页。每个应用单独一个 project：同一 project 里所有客户端的 ID 都会写进 ID Token 的 `aud` | `[EV]` |
| R4 | `sid` 是 ZITADEL 的会话 ID；读者在登录界面退出、走 end_session 或会话被管理员结束时，向这次会话登录过的每个应用发 back-channel logout。**会话自然过期时不发** | `[EV]` |
| R5 | Actions v2 把 `user.removed`、`user.deactivated`、`user.reactivated`、`user.locked`、`user.unlocked` 推给各应用，HMAC 签名；各应用只在停用时暂停账号，锁定和解锁只作通知（密码不再触发锁定，§5）。**只投递一次，失败不重试**，所以每小时对账是必需的：每个应用一个只读机器用户（`IAM_OWNER_VIEWER`），用 admin `ListEvents` 按时间游标取事件，再查用户当前状态。这个角色能读到所有推送目标的签名密钥，应用只能把推送当提示，删除或暂停前先查一次用户 | `[EV]` |
| R6 | 注册、邮箱验证、找回密码、Passkey 在登录界面；账号页是 ZITADEL 控制台的 `/ui/console/users/me`：改资料、换邮箱、改密码、Passkey 和第二因素、删号，各应用的「管理／删除 Kite Plus 账号」都链到这里（[README](../../README.md#account-page)）。控制台用品牌设置里的 Logo 和图标，页面标题由反向代理改成 Kite Plus，配色和个别说明文字仍是 ZITADEL 的，改不了；以后是否换成自建账号页 `[待定]`（[roadmap.md §6](roadmap.md#6-待定事项)）。读者删号需要 `ORG_USER_SELF_MANAGER` 角色，ZITADEL 默认谁都不给：`identity-init` 给每个读者授予，新注册的读者由定时任务补上（[runbooks](../runbooks/README.md)）。关联 GitHub 在用 GitHub 登录时按已验证邮箱自动关联 | `[EV]`，GitHub 部分待 I1 |
| R7 | 登录保持 30 天（密码、Passkey、GitHub 的检查有效期都是 720h），不滑动续期。登录界面 v2 的 `sessions` Cookie 不带 Max-Age，关掉浏览器就没了（[zitadel#12434](https://github.com/zitadel/zitadel/issues/12434)），反向代理用 njs 给它补上 30 天（`deploy/nginx/njs`）；上游修好后删掉这段 | `[EV]`（代理改写用模拟上游验证） |
| R8 | 默认语言中文，允许中英文；品牌色和 Logo；邮件文案和登录界面里写着 Zitadel 的几处文字由 `config/base.yaml` 覆盖。登录界面注册时不带语言，ZITADEL 记为 `und`，邮件就退回英文：反向代理在注册表单经过时记下浏览器的语言（登录界面的语言切换优先，其次 `Accept-Language`，中英文以外按中文），再由一个 Actions v2 调用目标在创建用户时填上 `preferredLanguage`，没记到的一律中文 | `[EV]`：中文浏览器注册收到中文验证邮件，切到英文的浏览器收到英文 |
| R9 | 管理员在 Kite Plus Operators 组织里，这个组织单独的登录策略强制 MFA；读者在 Kite Plus 组织（默认组织） | 已配置，I2 实际启用 |
| R10 | nginx 限流，页面加载不限：每个 IP 在会发邮件或校验密码、验证码的页面（注册、验证、密码、验证码）每分钟提交 10 次，其他登录表单 60 次，OIDC 端点 300 次；所有客户端合计在注册、验证、密码、验证码页面每分钟 600 次、突发 300，挡住分布式猜密码，一批真实读者同时登录也够用。国内运营商 NAT 后很多人共用一个 IP，所以单 IP 放得比较宽。不再用账号锁定（§5）；用不到的旧登录界面 `/ui/login/` 在代理上屏蔽，绕不过限流 | `[EV]`：单 IP 第 12 次注册提交返回 429；合计限流在两个客户端之间生效（测试配置调低速率验证） |
| R11 | 配置已就绪（`APPLE_*` 环境变量），账号未开。**ZITADEL 不保存 Apple 的令牌，删号时也不撤销**（源码和文档都没有，Logto 也不做），撤销办法 I1 定（[roadmap.md §6](roadmap.md#6-待定事项)） | 待 I1 |

---

## 2. 登录方式

| 方式 | 决定 | 说明 |
|---|---|---|
| 邮箱加密码 | 做 | 邮箱必须验证；ZITADEL 要在登录容器上开启 `EMAIL_VERIFICATION` |
| Passkey | 做，推荐 | 绑定域名，钓不走。国内不少安卓手机没有谷歌服务，用不了 |
| GitHub | 做 | 技术博客的读者大多有；可以不设密码直接用 GitHub 注册 |
| 邮箱验证码 | 不做 | 没有 Passkey 的读者最省事的方式，但 ZITADEL 不支持作为首要因素（§3）。没有谷歌服务的安卓读者用邮箱加密码或 GitHub，I1 真机确认够不够用（S1） |
| Apple | 做，Kite 应用切换到统一登录之前上线 | R11。读者选择隐藏邮箱时，拿到的是 `privaterelay.appleid.com` 的转发地址，发信域名要先在 Apple 开发者账号里登记，否则验证邮件和通知收不到；Apple 只在第一次授权时给出名字，之后要靠身份服务自己存的资料 |
| Google | `[待定]` | 国内用不了，对海外读者有用 |
| 微信、QQ | 不做 | 微信网站登录只对企业等组织开放，还要求已备案的域名；QQ 互联个人可以申请，但同样要求网站已备案（[comments architecture.md §5](https://github.com/kite-plus/comments/blob/main/docs/design/architecture.md#5-部署地与合规)）。两者的 OAuth 也都不是标准实现 |

- 一个人的多种登录方式在身份服务里关联到同一个账号。已经分成两个账号的不提供合并：两个 `(iss, sub)` 在各应用里就是两个用户。
- 显示名：各应用首次登录时取 `name`，没有时取 `preferred_username`，之后由各应用自己管理。

---

## 3. 选型

两个候选，都自托管、都能满足 §1 的大部分要求。下表是 I0 的结论，2026-10-11 在本机实测 ZITADEL v4.19.4 和 Logto 1.44.0，逐项记录在 [lab identity-selection](https://github.com/kite-plus/lab/blob/main/prototypes/identity-selection/README.md)。

| | ZITADEL v4 | Logto |
|---|---|---|
| 许可证 | 核心 AGPL-3.0，登录界面 MIT；不修改、作为独立服务运行，对本项目没有影响 | MPL-2.0 |
| 技术栈 | Go；接口和登录界面（Next.js）两个容器；只支持 PostgreSQL | Node；PostgreSQL |
| OIDC | 机密客户端加 PKCE、`sid`、back-channel logout、`prompt=select_account` 都可用；`prompt=none` 没有会话时显示登录页而不是返回 `login_required`（[zitadel#12165](https://github.com/zitadel/zitadel/issues/12165)）`[EV]` | `prompt=none` 符合规范；**不支持 `prompt=select_account`**；`sid` 按客户端区分 `[EV]` |
| 登录方式 | 密码、可作为首要因素的 Passkey、GitHub 等社交登录；**不支持邮箱验证码或魔法链接作为首要因素** | 原生支持邮箱和短信验证码登录；官方有微信、QQ 等国内平台的连接器 |
| 中文 | 登录界面和邮件的简体中文完整；注册表单要求填姓和名，显示名拼成「名 姓」 `[EV]` | 界面自动识别中文，可只用邮箱加验证码注册，不收集名字 `[EV]` |
| 用户事件 | Actions v2 按事件推送（HMAC 签名），只投递一次不重试；admin `ListEvents` 可按时间游标查询 `[EV]` | Webhook 推送（HMAC 签名），失败会重试；**没有用户事件查询接口**，对账只能列全部用户 `[EV]` |
| 登录保持 | 时长可配；登录界面 v2 的 Cookie 关浏览器就失效，靠反向代理补 Max-Age `[EV]` | 会话 Cookie 默认 14 天 `[EV]` |
| 运维 | 空闲约 350 MB；`pg_dump` 一个库即可恢复；升级时自动迁移，小版本升级约 11 秒不可用 `[EV]` | 空闲约 270 MB；恢复还要重建数据库角色；升级要手工执行 alteration，否则起不来 `[EV]` |
| 防滥用 | 没有内置限流和验证码；有账号锁定，锁住后只能管理员解锁 `[EV]` | 临时锁定、验证码（Turnstile、reCAPTCHA）；没有按 IP 限流 |
| 安全公告 | 2024-10 以来 57 个，严重 7 个，多为登录流程中的账号接管；v4 约每周一个补丁版本 | 同期 13 个，没有严重级，都在 2026-07 之后 |
| FedCM | 不支持 | 不支持 |
| Apple 登录 | 有 Apple 身份提供方；不保存 Apple 令牌，删号时不撤销（源码和文档） | 有 Apple 连接器；同样不撤销 |

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

**已选 ZITADEL（I1 确认）**：OIDC 能力最完整，和本组织的 Go 技术栈一致。Logto 在 S2（邮箱验证码注册）、S3 的一部分（`prompt=none`、推送重试、持久 Cookie）和 S5（公告少）上更好，但 S3 缺 `prompt=select_account`（应用里「换一个账号」要用）和用户事件查询（R5 的对账），S4 的恢复和升级也更费事，所以不满足「S2–S6 都不比 ZITADEL 差」。条件是 S1：没有谷歌服务的安卓读者只靠邮箱加密码和 GitHub 能不能接受，I1 用真机确认；S7 的 Apple 登录同样等 I1。两项任何一项不成立，按决定规则重新评估。ZITADEL 的几处缺口已有对策：`prompt=none` 各应用不依赖（评论组件本来就不做自动识别），登录保持由反向代理补 Cookie（§1.1 R7），读者删号由初始化工具授权（R6）。

另外两条路：自己写，不采用（§0）；托管的身份服务上线快，但要持续付费，数据放在哪、以后能不能迁走都由供应商决定，只作为运维上的备选。

---

## 4. 部署

- **地点**：香港，和以后的评论服务同一台服务器。不需要备案，到国内三网的延迟在可接受范围。账号是个人信息，放在境外涉及的出境规则，和评论服务一样，见 [comments architecture.md §5](https://github.com/kite-plus/comments/blob/main/docs/design/architecture.md#5-部署地与合规)。服务商和线路 `[待定]`，I1 开工前定。
- **环境**：生产 `id.kite.plus`；预发布 `id.staging.kite.plus`，独立实例、独立数据库。每次升级先在预发布上跑。
- **组成**（Docker Compose，`deploy/docker-compose.yaml`，版本固定在 `.env`）：
  - 反向代理：nginx 1.30（官方镜像）。终止 TLS，证书由镜像自带的 ACME 模块向 Let's Encrypt 申请和续期（HTTP-01，要开放 80 端口）；对登录、注册、找回密码、发验证码的表单提交限流（R10，§1.1）；用 njs 给登录界面的 Cookie 补 Max-Age（R7），并为新读者设语言（R8：内部端口 8082 上的调用目标，只对 Compose 网络开放）；改写控制台的页面标题（R6）；清掉客户端伪造的 `X-Forwarded-Host`、`X-Zitadel-*` 头，ZITADEL 会用它们拼 issuer 和邮件里的链接。选 nginx 不选 Caddy：限流、njs、ACME 都在官方镜像里，不用自己编译插件 `[EV]`。
  - 网络：Compose 网络固定为 `172.31.250.0/24`、关掉 IPv6。ZITADEL 对外请求保留默认的内网拒绝列表，只放行这个网段，让语言调用目标够得着代理 `[EV]`。
  - 身份服务：ZITADEL v4.19.4，接口和登录界面两个容器，两个镜像版本必须一致。`zitadel-init`（建库建角色，只有它拿到 PostgreSQL 超级用户密码）和 `zitadel-setup`（迁移、首次建实例）是一次性容器，每次启动都先跑。
  - PostgreSQL 17：身份服务独占一个库和一个角色。以后评论服务上同一台机器，用另外的库和角色。
- **资源**：只跑身份服务时 2 核 4 GB 起步（ZITADEL 自己建议至少 2 GB 内存，密码哈希会让 CPU 短时冲高；本机空闲时整套约 350 MB `[EV]`）；评论服务加入后再扩 `[待定]`。
- **发信**：SMTP 服务商 `[待定]`；发件人 `Kite Plus 账号 <noreply@kite.plus>`；`kite.plus` 配好 SPF、DKIM 和 DMARC，否则 QQ 邮箱、163 邮箱很可能直接拒收。
- **GitHub 登录**：在 `kite-plus` 组织下注册 OAuth App，生产和预发布各一个，回调指向各自的 `id` 域名。
- **Apple 登录**：在 Apple 开发者账号下建 Services ID 和用于登录的私钥，生产和预发布各一个 Services ID，回调指向各自的 `id` 域名；`noreply@kite.plus` 所在的发信域名登记到 Apple 的邮件转发服务（§2）。
- **密钥**：
  - ZITADEL 的主密钥是 32 个字符，初始化后不能改；丢了，库里加密的数据就再也解不开。离线保存两份。
  - 数据库密码、SMTP 凭据、GitHub 客户端密钥、Apple 登录私钥放在服务器的 `deploy/.env`，不进仓库。
  - 首次启动写出两个引导令牌：`admin.pat`（`identity-init` 用，实例所有者权限）和 `login-client.pat`（登录界面用），放在 `deploy/secrets/bootstrap`，只有 root 能读。
  - 各应用的客户端密钥、事件签名密钥、对账令牌由 `identity-init` 写进 `secrets/` 下按应用分开的环境文件，再交给各应用的部署。轮换见 [runbooks](../runbooks/key-rotation.md)。

---

## 5. 运维

- **升级**：订阅身份服务的安全公告。**严重漏洞 72 小时内升级到生产，其余随每月一次的例行升级**；都先在预发布环境升级。本组织所有提到身份服务升级时限的地方都以这一条为准。
- **备份**：每天 `scripts/backup.sh`：`pg_dump` 身份服务的库连同引导令牌，用 age 公钥加密，保存 30 天并异地复制（存放位置 `[待定]`，I2 前定）；主密钥不进备份。每个阶段结束做一次恢复到空机器的演练（[restore.md](../runbooks/restore.md)，2026-10-11 本机演练过一次 `[EV]`）。
- **监控**：外部探测 `/healthz`，探测点包括国内三网；对登录失败率、发信失败、证书到期、磁盘和备份结果设告警。
- **防滥用**：反向代理限流（§4，§1.1 R10）；登录失败的延迟保持默认。**不开密码锁定**：ZITADEL 锁住的账号只能由管理员解锁，重置密码也解不开 `[EV]`，开着的话，知道读者邮箱的人连输几次错密码就能把他锁在外面。一次性验证码的锁定保留（连续错 10 次），它要先输对密码才会出现。ZITADEL 没有按账号的节流，合计限流是猜密码的上限；合计限流被打满时真实读者也会收到 429，见 [incident.md](../runbooks/incident.md)。
- **定时任务**：每天备份；每 5 分钟 `identity-init -only readers`，给新注册的读者授予自助删号的角色（§1.1 R6）。
- **管理员**：平台管理员账号强制 MFA；日常不用管理员账号登录各应用。
- **隐私政策和服务条款**：账号存着邮箱等个人信息，生产上线（I2）前必须有，中英文。
- **故障**：身份服务停机时，已经登录的应用不受影响，因为各应用都有自己的会话；只是新登录进不来。

---

## 6. 仓库结构

```text
identity/
├── cmd/identity-init/   初始化工具：把 config/ 应用到实例，写出各应用的环境文件
├── internal/            初始化工具的实现
├── config/              base.yaml 加 dev、staging、production 各一份
├── deploy/              docker-compose.yaml、nginx 配置、.env.example
├── dev/                 本地开发用的 identity-dev 栈
├── scripts/             备份、恢复、升级、开发辅助
├── test/e2e/            对本地栈的端到端检查（不用浏览器）
├── branding/            Logo、配色
└── docs/
    ├── design/          本文和路线图
    └── runbooks/        首次部署、升级、恢复、密钥轮换、故障处理
```

初始化工具要能重复执行：对着一个空实例跑一遍，就得到和生产一样的配置；紧接着再跑一遍不做任何改动 `[EV]`。改配置先改 `config/`，再跑工具，不在管理后台里手工点。

---

## 7. 参考

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)、[OpenID Connect Back-Channel Logout 1.0](https://openid.net/specs/openid-connect-backchannel-1_0.html)。
- I0 实测记录：[lab identity-selection](https://github.com/kite-plus/lab/blob/main/prototypes/identity-selection/README.md)。
- ZITADEL：[Docker Compose 部署](https://zitadel.com/docs/self-hosting/deploy/compose)、[自定义域名](https://zitadel.com/docs/self-hosting/manage/custom-domain)、[back-channel logout](https://zitadel.com/docs/guides/integrate/back-channel-logout)、[Actions v2](https://zitadel.com/docs/guides/integrate/actions/usage)、[安全公告](https://github.com/zitadel/zitadel/security/advisories)。
- Logto：[连接器](https://github.com/logto-io/logto/tree/master/packages/connectors)。
