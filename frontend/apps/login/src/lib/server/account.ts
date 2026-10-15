"use server";

import { AccountCommand, AccountOverview, AccountProfile, AccountResult } from "@/lib/account-types";
import { getAllSessions, removeSessionFromCookie, setLanguageCookie } from "@/lib/cookies";
import { getLoginSettings } from "@/lib/zitadel";
import { getLocale } from "next-intl/server";
import { revalidatePath } from "next/cache";
import {
  AccountAccessError,
  AccountContext,
  accountRequest,
  listAccountDevices,
  listAccountSessions,
  requireAccountContext,
  updateAccountSession,
} from "./account-context";

function requiredText(value: unknown, max = 200): string {
  if (typeof value !== "string" || !value.trim() || value.length > max) throw new AccountAccessError("invalid");
  return value.trim();
}
async function waitForAccountChange(check: () => Promise<boolean>) {
  // Wait for the query projection so a reload does not show the old value.
  for (let attempt = 0; attempt < 30; attempt++) {
    if (await check()) return;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new AccountAccessError("unavailable");
}

async function ensureVerified(context: AccountContext) {
  const { email } = await accountRequest<{ email: { isEmailVerified?: boolean } }>(context, "/auth/v1/users/me/email");
  if (!email.isEmailVerified) throw new AccountAccessError("unverified");
}
async function actionError(error: unknown): Promise<AccountResult> {
  const zh = (await getLocale()) === "zh";
  if (error instanceof AccountAccessError) {
    if (error.reason === "fresh")
      return {
        ok: false,
        reauthRequired: true,
        message: zh ? "请再次验证身份，然后继续这项操作。" : "Verify your identity again to continue.",
      };
    if (error.reason === "expired")
      return {
        ok: false,
        redirect: "/expired",
        message: zh ? "登录已过期，请重新登录。" : "Your session expired. Sign in again.",
      };
    if (error.reason === "login")
      return {
        ok: false,
        redirect: "/loginname",
        message: zh ? "登录已过期，请重新登录。" : "Your session has expired. Sign in again.",
      };
    if (error.reason === "unverified")
      return {
        ok: false,
        message: zh ? "请先验证邮箱，再修改账号安全设置。" : "Verify your email before changing security settings.",
      };
    if (error.status === 429)
      return { ok: false, message: zh ? "操作过于频繁，请稍后重试。" : "Too many attempts. Try again later." };
    if (error.status === 403)
      return {
        ok: false,
        message: zh
          ? "暂时没有执行此操作的权限。新注册的账号请稍后重试。"
          : "You don't have permission yet. Newly registered accounts can try again shortly.",
      };
    if (error.reason === "invalid")
      return {
        ok: false,
        message: zh
          ? "操作未完成，请检查输入、密码或验证码后重试。"
          : "The change wasn't saved. Check your input, password or code and try again.",
      };
  }
  return {
    ok: false,
    message: zh ? "账号服务暂时无法响应，请稍后重试。" : "The account service is unavailable. Try again later.",
  };
}

export async function getAccountOverview(): Promise<AccountOverview> {
  const context = await requireAccountContext();
  const path = `/v2/users/${encodeURIComponent(context.userId)}`;
  const [user, profileResponse, emailResponse, methods, passkeys, devices, policy, loginSettings] = await Promise.all([
    accountRequest<{ user?: { details?: { creationDate?: string } } }>(context, "/auth/v1/users/me").catch(() => undefined),
    accountRequest<{ profile: AccountProfile }>(context, "/auth/v1/users/me/profile"),
    accountRequest<{ email: { email: string; isEmailVerified?: boolean } }>(context, "/auth/v1/users/me/email"),
    accountRequest<{ authMethodTypes?: string[] }>(context, `${path}/authentication_methods`),
    accountRequest<{ result?: AccountOverview["passkeys"] }>(context, `${path}/passkeys/_search`, "POST", {}),
    listAccountDevices(context),
    accountRequest<{ policy: AccountOverview["passwordPolicy"] }>(context, "/auth/v1/policies/passwords/complexity"),
    getLoginSettings({ serviceConfig: context.serviceConfig, organization: context.session.factors?.user?.organizationId }),
  ]);
  return {
    userId: context.userId,
    createdAt: user?.user?.details?.creationDate,
    profile: profileResponse.profile,
    email: emailResponse.email.email,
    emailVerified: !!emailResponse.email.isEmailVerified,
    loginName: context.cookie.loginName,
    passkeys: (passkeys.result || []).map((key) => ({
      ...key,
      name: key.name || "",
      state: key.state || "AUTH_FACTOR_STATE_UNSPECIFIED",
    })),
    devices,
    totp: !!methods.authMethodTypes?.includes("AUTHENTICATION_METHOD_TYPE_TOTP"),
    hasPassword: !!methods.authMethodTypes?.includes("AUTHENTICATION_METHOD_TYPE_PASSWORD"),
    forceMfa: !!(loginSettings?.forceMfa || loginSettings?.forceMfaLocalOnly),
    passwordPolicy: policy.policy,
  };
}

export async function mutateAccount(command: AccountCommand): Promise<AccountResult> {
  try {
    if (!command || typeof command !== "object") throw new AccountAccessError("invalid");
    const context = await requireAccountContext({
      fresh: command.type !== "profile" && command.type !== "email-verify" && command.type !== "email-resend",
    });
    if (!["profile", "email", "email-verify", "email-resend"].includes(command.type)) await ensureVerified(context);
    const userPath = `/v2/users/${encodeURIComponent(context.userId)}`;
    switch (command.type) {
      case "profile": {
        const { profile } = await accountRequest<{ profile: AccountProfile }>(context, "/auth/v1/users/me/profile");
        if (!["zh", "en"].includes(command.language)) throw new AccountAccessError("invalid");
        await accountRequest(context, "/auth/v1/users/me/profile", "PUT", {
          ...profile,
          displayName: requiredText(command.displayName),
          preferredLanguage: command.language,
        });
        await waitForAccountChange(async () => {
          const result = await accountRequest<{ profile: AccountProfile }>(context, "/auth/v1/users/me/profile");
          return (
            result.profile.displayName === command.displayName.trim() &&
            result.profile.preferredLanguage === command.language
          );
        });
        await setLanguageCookie(command.language);
        break;
      }
      case "email":
        await accountRequest(context, "/auth/v1/users/me/email", "PUT", { email: requiredText(command.email) });
        await waitForAccountChange(async () => {
          const result = await accountRequest<{ email: { email: string } }>(context, "/auth/v1/users/me/email");
          return result.email.email === command.email.trim();
        });
        break;
      case "email-verify":
        await accountRequest(context, "/auth/v1/users/me/email/_verify", "POST", { code: requiredText(command.code) });
        await waitForAccountChange(async () => {
          const result = await accountRequest<{ email: { isEmailVerified?: boolean } }>(context, "/auth/v1/users/me/email");
          return !!result.email.isEmailVerified;
        });
        break;
      case "email-resend":
        await accountRequest(context, "/auth/v1/users/me/email/_resend_verification", "POST", {});
        break;
      case "password":
        if (
          typeof command.password !== "string" ||
          !command.password ||
          command.password.length > 256 ||
          typeof command.current !== "string"
        )
          throw new AccountAccessError("invalid");
        await accountRequest(context, "/auth/v1/users/me/password", "PUT", {
          oldPassword: command.current,
          newPassword: command.password,
        });
        break;
      case "passkey-start": {
        const result = await accountRequest<{
          passkeyId: string;
          publicKeyCredentialCreationOptions: Record<string, unknown>;
        }>(context, `${userPath}/passkeys`, "POST", { domain: context.domain });
        return { ok: true, data: { id: result.passkeyId, options: result.publicKeyCredentialCreationOptions } };
      }
      case "passkey-finish":
        await accountRequest(context, `${userPath}/passkeys/${encodeURIComponent(requiredText(command.id))}`, "POST", {
          passkeyName: requiredText(command.name),
          publicKeyCredential: command.credential,
        });
        await waitForAccountChange(async () => {
          const keys = await accountRequest<{ result?: { id: string; state?: string }[] }>(
            context,
            `${userPath}/passkeys/_search`,
            "POST",
            {},
          );
          return !!keys.result?.some((key) => key.id === command.id && key.state === "AUTH_FACTOR_STATE_READY");
        });
        break;
      case "passkey-remove": {
        const methods = await accountRequest<{ authMethodTypes?: string[] }>(context, `${userPath}/authentication_methods`);
        const keys = await accountRequest<{ result?: { id: string; state: string }[] }>(
          context,
          `${userPath}/passkeys/_search`,
          "POST",
          {},
        );
        const ready = keys.result?.filter((key) => key.state === "AUTH_FACTOR_STATE_READY") || [];
        if (
          ready.length === 1 &&
          ready[0].id === command.id &&
          !methods.authMethodTypes?.some((method) => method.endsWith("_PASSWORD") || method.endsWith("_IDP"))
        ) {
          return {
            ok: false,
            message:
              (await getLocale()) === "zh"
                ? "请先设置密码或添加另一枚通行密钥，再移除唯一的登录方式。"
                : "Set a password or add another passkey before removing your only sign-in method.",
          };
        }
        await accountRequest(context, `${userPath}/passkeys/${encodeURIComponent(requiredText(command.id))}`, "DELETE");
        await waitForAccountChange(async () => {
          const keys = await accountRequest<{ result?: { id: string }[] }>(
            context,
            `${userPath}/passkeys/_search`,
            "POST",
            {},
          );
          return !keys.result?.some((key) => key.id === command.id);
        });
        break;
      }
      case "totp-start": {
        const result = await accountRequest<{ secret: string; uri: string }>(context, `${userPath}/totp`, "POST", {});
        return { ok: true, data: { secret: result.secret, uri: result.uri } };
      }
      case "totp-verify":
        await accountRequest(context, `${userPath}/totp/verify`, "POST", { code: requiredText(command.code) });
        await updateAccountSession(context, { checks: { totp: { code: command.code } } });
        break;
      case "totp-remove": {
        const settings = await getLoginSettings({
          serviceConfig: context.serviceConfig,
          organization: context.session.factors?.user?.organizationId,
        });
        if (settings?.forceMfa || settings?.forceMfaLocalOnly) throw new AccountAccessError("invalid");
        await accountRequest(context, `${userPath}/totp`, "DELETE");
        break;
      }
      case "device-revoke": {
        const id = requiredText(command.id);
        const result = await accountRequest<{ session: { factors?: { user?: { id: string } } } }>(
          context,
          `/v2/sessions/${encodeURIComponent(id)}`,
        );
        if (result.session.factors?.user?.id !== context.userId) throw new AccountAccessError("invalid");
        await accountRequest(context, `/v2/sessions/${encodeURIComponent(id)}`, "DELETE");
        if (id === context.cookie.id) {
          await removeSessionFromCookie({ session: context.cookie });
          return { ok: true, redirect: "/logout/done" };
        }
        break;
      }
      case "devices-revoke": {
        const sessions = await listAccountSessions(context);
        for (const session of sessions)
          if (session.id !== context.cookie.id)
            await accountRequest(context, `/v2/sessions/${encodeURIComponent(session.id)}`, "DELETE");
        break;
      }
      case "delete":
        if (command.confirmation !== "DELETE" && command.confirmation !== "删除我的账号")
          throw new AccountAccessError("invalid");
        await accountRequest(context, "/auth/v1/users/me", "DELETE");
        for (const cookie of await getAllSessions())
          if (cookie.loginName === context.cookie.loginName) await removeSessionFromCookie({ session: cookie });
        return { ok: true, redirect: "/account/deleted" };
      default:
        throw new AccountAccessError("invalid");
    }
    revalidatePath("/account", "layout");
    return { ok: true };
  } catch (error) {
    return actionError(error);
  }
}

export async function reauthenticateAccount(password: string, code?: string): Promise<AccountResult> {
  try {
    const context = await requireAccountContext();
    if (typeof password !== "string" || !password || password.length > 256) throw new AccountAccessError("invalid");
    await updateAccountSession(context, {
      checks: { password: { password }, ...(code ? { totp: { code: requiredText(code) } } : {}) },
    });
    return { ok: true };
  } catch (error) {
    return actionError(error);
  }
}
export async function startAccountPasskeyCheck(): Promise<AccountResult> {
  try {
    const context = await requireAccountContext();
    const result = await updateAccountSession(context, {
      challenges: {
        webAuthN: { domain: context.domain, userVerificationRequirement: "USER_VERIFICATION_REQUIREMENT_REQUIRED" },
      },
    });
    return { ok: true, data: { options: result.challenges?.webAuthN?.publicKeyCredentialRequestOptions } };
  } catch (error) {
    return actionError(error);
  }
}
export async function finishAccountPasskeyCheck(credential: Record<string, unknown>): Promise<AccountResult> {
  try {
    const context = await requireAccountContext();
    await updateAccountSession(context, { checks: { webAuthN: { credentialAssertionData: credential } } });
    return { ok: true };
  } catch (error) {
    return actionError(error);
  }
}
