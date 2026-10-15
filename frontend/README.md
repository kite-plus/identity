# Kite Plus 账号界面

这里是实际运行的 Next.js Login V2 和账号管理前端，基于 ZITADEL v4.19.4。页面结构来自 `lab/design/identity/2026-10-12-product-ui/v2`，外观在 2026-10-15 改为与 kite.plus 一致的纸鸢风格（`lab/design/brand/2026-10-15-id-explore` 的方案 A）。上游来源和许可证见 [UPSTREAM.md](UPSTREAM.md)。

## 开发与验证

推荐在 identity 仓库执行 `make dev-up`：Docker 会从固定的依赖锁文件构建前端，启动本地身份环境。登录入口为 `http://localhost:8091/ui/v2/login/loginname`，账号管理为 `/ui/v2/login/account`。只启动或重建前端可执行：

```sh
docker compose --project-directory dev -f dev/docker-compose.yaml --env-file dev/.env up -d --build --no-deps login
```

本机调试使用 Node.js 22 或 24、pnpm 10.30.3。在本目录安装依赖、生成客户端后，从 `apps/login` 启动开发服务。需要配置 `ZITADEL_API_URL`、`NEXT_PUBLIC_BASE_PATH=/ui/v2/login`、`ZITADEL_SERVICE_USER_TOKEN_FILE`、`ZITADEL_SESSION_COOKIE_SECRET`、`CUSTOM_REQUEST_HEADERS`、`EMAIL_VERIFICATION=true`，沿用当前环境的配置，不要把实际令牌写入源码。

```sh
corepack pnpm install --frozen-lockfile
corepack pnpm generate
corepack pnpm build:client
corepack pnpm check
corepack pnpm test
corepack pnpm build
```

认证相关的 Go 回归从 identity 仓库执行 `make dev-test`。实际通行密钥注册、登录和再次验证身份需要在支持 WebAuthn 的浏览器中完成设备验证；后端登记和权限边界另有自动化覆盖。

## 页面与权限

- 原官方登录路由继续支持 OIDC、PKCE、会话续用、密码、通行密钥、MFA、注册、邮箱验证和退出。
- `/password/forgot`、`/password/reset-sent` 补齐密码恢复界面，调用官方密码恢复服务。
- `/account`、`/account/security`、`/account/devices`、`/account/delete`、`/account/deleted` 是完整账号管理界面。
- 账号页先校验签名 Cookie、服务端会话、主因素和所需 MFA；用户 ID 由已验证的服务端会话确定，客户端不能指定操作其他用户。
- 用户资料和认证方式的读写使用该用户的会话令牌；只有更新当前登录会话的验证因素时，沿用官方 Login 客户端，因为新启用 MFA 后用户令牌会立即要求 MFA。
- 敏感操作要求 10 分钟内的身份验证，可重新输入密码及 TOTP，或使用通行密钥。设备撤销在服务端再次核对所属用户。
- 更换邮箱会立即替换旧地址并标记为待验证；完成验证前不能修改其他安全设置。
- 删除账号仍由 ZITADEL 自助角色约束；新注册账号获得角色可能需等待现有读者同步任务。删除和撤销会话继续使用现有跨服务事件与退出通知。

## 样式与升级

`src/styles/kite.css` 提供纸张底色、颗粒、衬线标题、印章红和风筝三色等样式，以及响应式和暗色规则；通用组件保留官方表单接口。认证页和独立提示页都包在 `src/components/kite-auth-frame.tsx` 里：天空、风筝和连到卡片的风筝线。页面 CSP 只允许本站图片，所以纸张颗粒是 `public/grain*.svg`，不能写成 data URL。操作图标统一 Lucide 的 1.6px 线宽，状态图标与确认的 v2 稿一致。Kite 文案在 `src/i18n/kite-messages.ts`。

升级前必须更新 `upstream-version.txt` 和上游协议、客户端、登录源码，核对本仓库的页面及权限修改，再运行构建、前端认证测试、Go 端到端验证和浏览器回归。`scripts/upgrade.sh` 会拒绝后端版本与当前前端源码不一致的升级。
