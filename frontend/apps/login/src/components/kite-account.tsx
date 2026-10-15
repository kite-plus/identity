"use client";

import { useAccountPasskey } from "@/helpers/account-webauthn";
import { AccountCommand, AccountOverview, AccountResult, AccountSection } from "@/lib/account-types";
import {
  finishAccountPasskeyCheck,
  mutateAccount,
  reauthenticateAccount,
  startAccountPasskeyCheck,
} from "@/lib/server/account";
import {
  ArrowUpRight,
  Check,
  Compass,
  Fingerprint,
  KeyRound,
  Laptop,
  LogOut,
  Mail,
  MessageSquare,
  Plus,
  ShieldCheck,
  Smartphone,
  Trash2,
  UserRound,
  X,
} from "lucide-react";
import { useLocale } from "next-intl";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { QRCodeSVG } from "qrcode.react";
import { FormEvent, ReactNode, useEffect, useRef, useState } from "react";
import { Alert } from "./alert";
import { Button, ButtonColors, ButtonVariants } from "./button";
import { TextInput } from "./input";
import { KiteFlying } from "./kite-auth-frame";
import { Spinner } from "./spinner";

const EXPLORE_URL = "https://explore.kite.plus";

type Dialog = {
  type:
    | "email"
    | "email-verify"
    | "password"
    | "passkey"
    | "remove-key"
    | "setup-mfa"
    | "disable-mfa"
    | "revoke-device"
    | "revoke-all"
    | "delete-final"
    | "reauth";
  id?: string;
  name?: string;
};
const symbols = { UserRound, ShieldCheck, Laptop, Trash2, LogOut, Mail, KeyRound, Fingerprint, Smartphone, Plus, Check };
function Icon({ name, size = 18 }: { name: keyof typeof symbols; size?: number }) {
  const Component = symbols[name];
  return (
    <Component
      className="icon"
      size={size}
      style={{ width: size, height: size }}
      strokeWidth={1.6}
      absoluteStrokeWidth
      aria-hidden="true"
    />
  );
}

export function KiteAccount({ data, section }: { data: AccountOverview; section: AccountSection }) {
  const locale = useLocale();
  const text = (zh: string, en: string) => (locale === "zh" ? zh : en);
  const router = useRouter();
  const [name, setName] = useState(data.profile.displayName);
  const [language, setLanguage] = useState(data.profile.preferredLanguage === "en" ? "en" : "zh");
  const [confirmation, setConfirmation] = useState("");
  const [modal, setModal] = useState<Dialog | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [toast, setToast] = useState("");
  const [busy, setBusy] = useState(false);
  const [setup, setSetup] = useState<{ secret?: string; uri?: string }>({});
  const [pending, setPending] = useState<{ command: AccountCommand; dialog: Dialog | null } | null>(null);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const previousFocus = useRef<HTMLElement | null>(null);
  const expected = text("删除我的账号", "DELETE");
  useEffect(() => {
    setName(data.profile.displayName);
    setLanguage(data.profile.preferredLanguage === "en" ? "en" : "zh");
  }, [data.profile]);
  useEffect(() => {
    if (modal && dialogRef.current) {
      if (!dialogRef.current.open) {
        previousFocus.current = document.activeElement as HTMLElement;
        dialogRef.current.showModal();
      }
      const input = dialogRef.current.querySelector<HTMLInputElement>("input, select");
      input?.focus();
    } else {
      dialogRef.current?.close();
      previousFocus.current?.focus();
    }
  }, [modal]);
  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => setToast(""), 5000);
    return () => clearTimeout(timer);
  }, [toast]);
  function open(dialog: Dialog) {
    setValues({});
    setError("");
    setSetup({});
    setModal(dialog);
  }
  async function close() {
    if (busy) return;
    if (modal?.type === "setup-mfa" && setup.secret) await mutateAccount({ type: "totp-remove" });
    setModal(null);
    setPending(null);
    setError("");
    setValues({});
    setSetup({});
  }
  function change(key: string) {
    return (event: React.ChangeEvent<HTMLInputElement>) => {
      setValues((current) => ({ ...current, [key]: event.target.value }));
      setError("");
    };
  }
  function success(message: string) {
    setModal(null);
    setPending(null);
    setValues({});
    setError("");
    setSetup({});
    setToast(message);
    router.refresh();
  }
  async function handleResult(result: AccountResult, command: AccountCommand) {
    if (!result.ok) {
      if (result.redirect) {
        router.push(result.redirect);
        return;
      }
      if (result.reauthRequired) {
        setPending({ command, dialog: modal });
        setModal({ type: "reauth" });
        setError("");
      } else setError(result.message);
      return;
    }
    if (result.redirect) {
      router.push(result.redirect);
      return;
    }
    if (command.type === "email") {
      setModal({ type: "email-verify", name: command.email });
      setValues({});
      router.refresh();
      return;
    }
    if (command.type === "email-resend") {
      setToast(text("验证邮件已发送", "Verification email sent"));
      return;
    }
    if (command.type === "totp-start") {
      setSetup(result.data || {});
      return;
    }
    if (command.type === "passkey-start") {
      try {
        if (!result.data?.id || !result.data.options) throw new Error("missing");
        const credential = await useAccountPasskey(result.data.options, true);
        await handleResult(
          await mutateAccount({
            type: "passkey-finish",
            id: result.data.id,
            name: values.name?.trim() || text("我的通行密钥", "My passkey"),
            credential,
          }),
          { type: "passkey-finish", id: result.data.id, name: values.name || "Passkey", credential },
        );
      } catch {
        if (result.data?.id) await mutateAccount({ type: "passkey-remove", id: result.data.id });
        setError(
          text(
            "通行密钥未添加。请在支持的浏览器中重试，并完成系统验证。",
            "Passkey wasn't added. Try a supported browser and complete system verification.",
          ),
        );
      }
      return;
    }
    const messages: Partial<Record<AccountCommand["type"], [string, string]>> = {
      profile: ["资料已保存", "Profile saved"],
      "email-verify": ["邮箱已验证", "Email verified"],
      password: ["密码已修改", "Password changed"],
      "passkey-finish": ["通行密钥已添加", "Passkey added"],
      "passkey-remove": ["通行密钥已移除", "Passkey removed"],
      "totp-verify": ["双重验证已开启", "Two-step verification enabled"],
      "totp-remove": ["双重验证已关闭", "Two-step verification disabled"],
      "device-revoke": ["设备已退出登录", "Device signed out"],
      "devices-revoke": ["其他设备已退出登录", "Other devices signed out"],
    };
    success(text(...(messages[command.type] || ["已保存", "Saved"])));
  }
  async function run(command: AccountCommand) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await handleResult(await mutateAccount(command), command);
    } catch {
      setError(text("操作未完成，请稍后重试。", "The change wasn't saved. Try again later."));
    } finally {
      setBusy(false);
    }
  }
  async function finishReauthentication(result: AccountResult) {
    if (!result.ok) {
      setError(result.message);
      return;
    }
    setError("");
    if (pending) {
      const operation = pending;
      setPending(null);
      setModal(operation.dialog);
      await handleResult(await mutateAccount(operation.command), operation.command);
    } else success(text("身份已验证", "Identity verified"));
  }
  async function verifyIdentity(passkey = false) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      if (passkey) {
        const start = await startAccountPasskeyCheck();
        if (!start.ok) {
          setError(start.message);
          return;
        }
        if (!start.data?.options) throw new Error("missing");
        const credential = await useAccountPasskey(start.data.options, false);
        await finishReauthentication(await finishAccountPasskeyCheck(credential));
      } else await finishReauthentication(await reauthenticateAccount(values.verifyPassword || "", values.verifyCode));
    } catch {
      setError(text("身份验证未完成，请重新尝试。", "Identity verification wasn't completed. Try again."));
    } finally {
      setBusy(false);
    }
  }
  function submit(event: FormEvent) {
    event.preventDefault();
    if (!modal) return;
    if (modal.type === "email") void run({ type: "email", email: values.email || "" });
    if (modal.type === "email-verify") void run({ type: "email-verify", code: values.code || "" });
    if (modal.type === "password") {
      if (values.password !== values.confirm) {
        setError(text("两次输入的新密码不一致。", "The new passwords don't match."));
        return;
      }
      void run({ type: "password", current: values.current || "", password: values.password || "" });
    }
    if (modal.type === "passkey") void run({ type: "passkey-start" });
    if (modal.type === "setup-mfa") void run({ type: "totp-verify", code: values.code || "" });
    if (modal.type === "reauth") void verifyIdentity();
  }
  const titles: Record<AccountSection, [string, string]> = {
    profile: ["个人资料", "Your profile"],
    security: ["账号安全", "Account security"],
    devices: ["登录设备", "Signed-in devices"],
    delete: ["删除账号", "Delete your account"],
  };
  const descriptions: Record<AccountSection, [string, string]> = {
    profile: ["你的名字、邮箱，以及这个账号能登录的地方。", "Your name, your email, and where this account signs you in."],
    security: ["管理登录方式，保护你的 Kite Plus 账号。", "Manage your sign-in methods and protect your Kite Plus account."],
    devices: ["查看保持登录的设备，结束不再使用的会话。", "Review signed-in devices and end sessions you no longer use."],
    delete: ["永久删除你的 Kite Plus 账号和认证方式。", "Permanently delete your Kite Plus account and sign-in methods."],
  };
  const navigation: [AccountSection, keyof typeof symbols][] = [
    ["profile", "UserRound"],
    ["security", "ShieldCheck"],
    ["devices", "Laptop"],
    ["delete", "Trash2"],
  ];
  const button = (label: string, onClick: () => void, danger = false) => (
    <Button
      variant={ButtonVariants.Secondary}
      color={danger ? ButtonColors.Warn : ButtonColors.Neutral}
      disabled={busy}
      onClick={onClick}
    >
      {label}
    </Button>
  );
  const row = (icon: keyof typeof symbols, title: ReactNode, description: ReactNode, action: ReactNode) => (
    <div className="settings-row">
      <div className="row-leading">
        <Icon name={icon} size={20} />
        <div>
          <div className="row-title">{title}</div>
          <p className="row-description">{description}</p>
        </div>
      </div>
      {action}
    </div>
  );
  const badge = (
    <span className={`badge ${data.emailVerified ? "success" : ""}`}>
      {data.emailVerified ? text("已验证", "Verified") : text("待验证", "Unverified")}
    </span>
  );
  const date = (value?: string) =>
    value
      ? new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value))
      : text("未知", "Unknown");
  const passwordRequirements = [
    text(`至少 ${data.passwordPolicy.minLength || 10} 位`, `At least ${data.passwordPolicy.minLength || 10} characters`),
    data.passwordPolicy.hasUppercase && text("大写字母", "uppercase letter"),
    data.passwordPolicy.hasLowercase && text("小写字母", "lowercase letter"),
    data.passwordPolicy.hasNumber && text("数字", "number"),
    data.passwordPolicy.hasSymbol && text("符号", "symbol"),
  ]
    .filter(Boolean)
    .join(text("、", ", "));
  const readyKeys = data.passkeys.filter((key) => key.state === "AUTH_FACTOR_STATE_READY").length;
  const checks = [
    {
      name: text("密码", "Password"),
      done: data.hasPassword,
      state: data.hasPassword ? text("已设置", "Set") : text("未设置", "Not set"),
    },
    {
      name: text("通行密钥", "Passkey"),
      done: readyKeys > 0,
      state: readyKeys ? text(`${readyKeys} 个`, `${readyKeys} added`) : text("未添加", "None yet"),
    },
    {
      name: text("双重验证", "Two-step verification"),
      done: data.totp,
      state: data.totp ? text("已开启", "On") : text("未开启", "Off"),
    },
  ];
  let content: ReactNode;
  if (section === "profile")
    content = (
      <>
        <div className="kite-pass">
          <div className="pass-main">
            <div className="pass-who">
              <span className="seal xl">{Array.from(data.profile.displayName || data.email)[0]}</span>
              <div className="pass-names">
                <div className="pass-name">{data.profile.displayName}</div>
                <div className="pass-mail">
                  <span>{data.email}</span>
                  {badge}
                </div>
              </div>
            </div>
            <dl className="pass-meta">
              <div>
                <dt>{text("账号编号", "Account ID")}</dt>
                <dd className="mono">{data.userId.replace(/(\d{4})(?=\d)/g, "$1 ")}</dd>
              </div>
              <div>
                <dt>{text("界面语言", "Language")}</dt>
                <dd>{data.profile.preferredLanguage === "en" ? "English" : "简体中文"}</dd>
              </div>
              {data.createdAt && (
                <div>
                  <dt>{text("加入于", "Joined")}</dt>
                  <dd>
                    {new Intl.DateTimeFormat(locale, { dateStyle: "long", timeZone: "UTC" }).format(
                      new Date(data.createdAt),
                    )}
                  </dd>
                </div>
              )}
            </dl>
          </div>
          <div className="pass-stub" aria-hidden="true">
            <svg className="stub-art" viewBox="0 0 120 120">
              <path className="stub-string" d="M70.7 25 C 80 56, 42 74, 60 100" />
              <g transform="rotate(-12 60 40)">
                <KiteFlying x="24" y="0" width="74" height="74" />
              </g>
              <circle className="stub-hole" cx="60" cy="106" r="6" />
            </svg>
            <div className="stub-label">
              <b>KITE PLUS</b>ACCOUNT
            </div>
          </div>
        </div>
        <section className="settings-section">
          <div className="section-heading">
            <div>
              <h2>{text("这个账号可以登录", "Where this account works")}</h2>
              <p>{text("同一个账号，下面这些地方都能用。", "One account signs you in to each of these.")}</p>
            </div>
          </div>
          <div className="place-grid">
            <a className="place" href={EXPLORE_URL}>
              <span className="place-icon">
                <Compass className="icon" size={19} strokeWidth={1.8} aria-hidden="true" />
              </span>
              <b>Explore</b>
              <small>{text("订阅独立博客，收藏想读的文章", "Follow independent blogs and save posts to read")}</small>
              <span className="place-go">
                {text("打开 Explore", "Open Explore")}
                <ArrowUpRight className="icon" size={13} aria-hidden="true" />
              </span>
            </a>
            <div className="place">
              <span className="place-icon red">
                <MessageSquare className="icon" size={19} strokeWidth={1.8} aria-hidden="true" />
              </span>
              <b>{text("Kite 评论", "Kite Comments")}</b>
              <small>{text("在使用 Kite 评论的博客下留言", "Comment on blogs that use Kite Comments")}</small>
              <span className="place-go muted">{text("在博客的评论区登录", "Sign in from a blog's comments")}</span>
            </div>
            <div className="place">
              <span className="place-icon gold">
                <Smartphone className="icon" size={19} strokeWidth={1.8} aria-hidden="true" />
              </span>
              <b>{text("Kite 应用", "Kite app")}</b>
              <small>{text("在 iPhone 和 Android 上阅读", "Read on iPhone and Android")}</small>
              <span className="place-go muted">{text("即将上架", "Coming soon")}</span>
            </div>
          </div>
        </section>
        <section className="settings-section">
          <div className="section-heading">
            <div>
              <h2>{text("账号安全", "Account security")}</h2>
              <p>
                {checks.every((check) => check.done)
                  ? text("登录方式都已设置好。", "All sign-in protections are on.")
                  : text("还可以加强：", "You can still add: ") +
                    checks
                      .filter((check) => !check.done)
                      .map((check) => check.name)
                      .join(text("、", ", "))}
              </p>
            </div>
            <Link className="button small outline" href="/account/security">
              {text("管理", "Manage")}
            </Link>
          </div>
          <ul className="security-checks">
            {checks.map((check) => (
              <li key={check.name} className={check.done ? "done" : "todo"}>
                <span className="check-state">
                  {check.done ? (
                    <Check className="icon" size={14} strokeWidth={2.2} aria-hidden="true" />
                  ) : (
                    <Plus className="icon" size={14} strokeWidth={2.2} aria-hidden="true" />
                  )}
                </span>
                <div>
                  <b>{check.name}</b>
                  <small>{check.state}</small>
                  {!check.done && <Link href="/account/security">{text("去设置", "Set up")}</Link>}
                </div>
              </li>
            ))}
          </ul>
        </section>
        <section className="settings-section">
          <div className="section-heading">
            <h2>{text("基本信息", "Basic information")}</h2>
          </div>
          <form
            className="settings-form"
            onSubmit={(event) => {
              event.preventDefault();
              void run({ type: "profile", displayName: name, language });
            }}
          >
            <TextInput
              label={text("显示名称", "Display name")}
              value={name}
              onChange={(event) => setName(event.target.value)}
              required
              maxLength={200}
              disabled={busy}
              autoComplete="name"
            />
            <div className="field">
              <div className="field-label">
                <label htmlFor="account-language">{text("首选语言", "Preferred language")}</label>
              </div>
              <select
                id="account-language"
                className="input"
                value={language}
                onChange={(event) => setLanguage(event.target.value)}
                disabled={busy}
              >
                <option value="zh">简体中文</option>
                <option value="en">English</option>
              </select>
              <p className="field-hint">
                {text("用于账号页面和邮件通知。", "Used for your account pages and email notifications.")}
              </p>
            </div>
            <div className="form-actions">
              <Button
                type="submit"
                disabled={
                  busy ||
                  !name.trim() ||
                  (name === data.profile.displayName && language === (data.profile.preferredLanguage === "en" ? "en" : "zh"))
                }
              >
                {busy && <Spinner />}
                {text("保存修改", "Save changes")}
              </Button>
            </div>
          </form>
        </section>
        <section className="settings-section">
          <div className="section-heading">
            <h2>{text("邮箱地址", "Email address")}</h2>
          </div>
          {row(
            "Mail",
            <>
              {data.email}
              {badge}
            </>,
            text("用于登录、账号恢复和接收安全通知。", "Used for sign-in, recovery and security notifications."),
            button(text("更换邮箱", "Change email"), () => open({ type: "email" })),
          )}
          {!data.emailVerified && (
            <div className="form-actions">
              {button(text("验证邮箱", "Verify email"), () => open({ type: "email-verify", name: data.email }))}
              {button(text("重新发送", "Resend email"), () => void run({ type: "email-resend" }))}
            </div>
          )}
        </section>
      </>
    );
  else if (section === "security")
    content = (
      <>
        <section className="settings-section">
          <div className="section-heading">
            <h2>{text("密码", "Password")}</h2>
          </div>
          {row(
            "KeyRound",
            data.hasPassword ? text("登录密码", "Sign-in password") : text("未设置密码", "No password set"),
            text("使用较长的独特密码，避免与其他网站重复。", "Use a long, unique password you don't use elsewhere."),
            data.hasPassword ? (
              button(text("修改密码", "Change password"), () => open({ type: "password" }))
            ) : (
              <Link className="button outline" href={`/password/set?loginName=${encodeURIComponent(data.loginName)}`}>
                {text("设置密码", "Set password")}
              </Link>
            ),
          )}
        </section>
        <section className="settings-section">
          <div className="section-heading">
            <div>
              <h2>{text("通行密钥", "Passkeys")}</h2>
              <p>
                {text("使用指纹、面容或设备锁屏验证登录。", "Sign in with your fingerprint, face or device screen lock.")}
              </p>
            </div>
            {button(text("添加", "Add passkey"), () => open({ type: "passkey" }))}
          </div>
          {data.passkeys.length ? (
            data.passkeys.map((key) => (
              <div key={key.id}>
                {row(
                  "Fingerprint",
                  key.name || text("通行密钥", "Passkey"),
                  key.state === "AUTH_FACTOR_STATE_READY"
                    ? text("可用于登录", "Ready to use")
                    : text("设置未完成", "Setup incomplete"),
                  button(text("移除", "Remove"), () => open({ type: "remove-key", id: key.id, name: key.name })),
                )}
              </div>
            ))
          ) : (
            <div className="empty-state">
              <Icon name="Fingerprint" size={24} />
              <p>{text("你还没有添加通行密钥。", "You haven't added a passkey yet.")}</p>
            </div>
          )}
        </section>
        <section className="settings-section">
          <div className="section-heading">
            <h2>{text("双重验证", "Two-step verification")}</h2>
          </div>
          {row(
            "ShieldCheck",
            <>
              {text("验证器应用", "Authenticator app")}
              {data.totp && <span className="badge success">{text("已开启", "Enabled")}</span>}
            </>,
            data.forceMfa
              ? text("此账号必须开启双重验证。", "Two-step verification is required for this account.")
              : text(
                  "密码登录时，再输入验证器中的动态验证码。",
                  "Enter a code from your authenticator after your password.",
                ),
            data.totp ? (
              <Button
                variant={ButtonVariants.Secondary}
                disabled={busy || data.forceMfa}
                onClick={() => open({ type: "disable-mfa" })}
              >
                {text("关闭", "Disable")}
              </Button>
            ) : (
              button(text("开启", "Enable"), () => {
                open({ type: "setup-mfa" });
              })
            ),
          )}
        </section>
      </>
    );
  else if (section === "devices")
    content = (
      <>
        <div className="inline-note">
          {text(
            "这里显示 Kite Plus 账号的登录会话。退出会通知已接入的应用结束对应会话。",
            "These are Kite Plus account sessions. Signing out notifies connected apps to end their linked session.",
          )}
        </div>
        {[true, false].map((current) => (
          <section className="settings-section" key={String(current)}>
            <div className="section-heading">
              <h2>{current ? text("当前设备", "This device") : text("其他设备", "Other devices")}</h2>
            </div>
            {data.devices.filter((device) => device.current === current).length ? (
              data.devices
                .filter((device) => device.current === current)
                .map((device) => (
                  <div key={device.id}>
                    {row(
                      /mobile|iphone|android|ios/i.test(device.name) ? "Smartphone" : "Laptop",
                      <>
                        {device.name || text("浏览器会话", "Browser session")}
                        {current && <span className="badge success">{text("当前设备", "This device")}</span>}
                      </>,
                      <>
                        {text("会话更新：", "Session updated: ")}
                        {date(device.updatedAt)}
                        <br />
                        {text("创建于：", "Created: ")}
                        {date(device.createdAt)}
                      </>,
                      current ? (
                        <Link className="text-button muted" href="/logout">
                          {text("退出", "Sign out")}
                        </Link>
                      ) : (
                        button(text("退出登录", "Sign out"), () =>
                          open({ type: "revoke-device", id: device.id, name: device.name }),
                        )
                      ),
                    )}
                  </div>
                ))
            ) : (
              <div className="empty-state">
                <Icon name="Laptop" size={24} />
                <p>{text("没有其他设备保持登录。", "No other devices are signed in.")}</p>
              </div>
            )}
          </section>
        ))}
        {data.devices.some((device) => !device.current) && (
          <div className="form-actions">
            {button(text("退出其他设备", "Sign out other devices"), () => open({ type: "revoke-all" }))}
          </div>
        )}
      </>
    );
  else
    content = (
      <>
        <div className="delete-panel">
          <h2>{text("这项操作无法撤销", "This action can't be undone")}</h2>
          <ul>
            <li>
              {text(
                "你的 Kite Plus 账号和认证方式将被删除。",
                "Your Kite Plus account and sign-in methods will be deleted.",
              )}
            </li>
            <li>
              {text(
                "Explore 将按其删除规则处理订阅和关联资料。",
                "Explore will handle your subscriptions and linked information under its deletion rules.",
              )}
            </li>
            <li>
              {text(
                "评论服务将按你在该服务设置的删除方式处理评论。",
                "Comments will be handled according to your deletion settings in that service.",
              )}
            </li>
            <li>
              {text("备份中的数据会随备份保留期结束而清除。", "Data in backups is removed as their retention period ends.")}
            </li>
          </ul>
        </div>
        <form
          className="delete-confirm"
          onSubmit={(event) => {
            event.preventDefault();
            if (confirmation === expected) open({ type: "delete-final" });
          }}
        >
          <TextInput
            label={text(`输入「${expected}」以确认`, `Type ${expected} to confirm`)}
            value={confirmation}
            onChange={(event) => setConfirmation(event.target.value)}
            autoComplete="off"
            spellCheck={false}
            disabled={busy}
          />
          <div className="form-actions">
            <Button type="submit" color={ButtonColors.Warn} disabled={busy || confirmation !== expected}>
              {text("删除我的账号", "Delete my account")}
            </Button>
            <Link className="text-button" href="/account">
              {text("保留账号", "Keep my account")}
            </Link>
          </div>
        </form>
      </>
    );
  const dialogTitles: Record<Dialog["type"], [string, string]> = {
    email: ["更换邮箱", "Change your email"],
    "email-verify": ["验证邮箱", "Verify your email"],
    password: ["修改密码", "Change your password"],
    passkey: ["添加通行密钥", "Add a passkey"],
    "remove-key": ["移除通行密钥？", "Remove this passkey?"],
    "setup-mfa": ["设置双重验证", "Set up two-step verification"],
    "disable-mfa": ["关闭双重验证？", "Disable two-step verification?"],
    "revoke-device": ["退出这个设备？", "Sign out this device?"],
    "revoke-all": ["退出其他设备？", "Sign out other devices?"],
    "delete-final": ["确认删除账号？", "Delete your account?"],
    reauth: ["验证你的身份", "Verify your identity"],
  };
  const field = (key: string, label: string, type = "text", autoComplete = "off") => (
    <TextInput
      label={label}
      type={type}
      value={values[key] || ""}
      onChange={change(key)}
      required
      disabled={busy}
      autoComplete={autoComplete}
      maxLength={type === "password" ? 256 : 200}
    />
  );
  const cancel = (
    <Button variant={ButtonVariants.Secondary} disabled={busy} onClick={() => void close()}>
      {text("取消", "Cancel")}
    </Button>
  );
  const submitButton = (label: string) => (
    <div className="modal-actions">
      {cancel}
      <Button type="submit" disabled={busy}>
        {busy && <Spinner />}
        {label}
      </Button>
    </div>
  );
  let body: ReactNode;
  if (modal?.type === "email")
    body = (
      <>
        <p>
          {text(
            "新邮箱将立即替换当前邮箱，完成验证后可用于登录和恢复账号。请确认地址无误。",
            "Your new email replaces the current address immediately. Verify it to use it for sign-in and recovery. Check the address carefully.",
          )}
        </p>
        <form onSubmit={submit} className="form-stack">
          {field("email", text("新邮箱", "New email"), "email", "email")}
          {submitButton(text("发送验证邮件", "Send verification email"))}
        </form>
      </>
    );
  else if (modal?.type === "email-verify")
    body = (
      <>
        <p>
          {text(`输入发送至 ${modal.name || data.email} 的验证码。`, `Enter the code sent to ${modal.name || data.email}.`)}
        </p>
        <form onSubmit={submit} className="form-stack">
          {field("code", text("邮箱验证码", "Email verification code"), "text", "one-time-code")}
          {submitButton(text("验证邮箱", "Verify email"))}
        </form>
        <button className="text-button muted" disabled={busy} onClick={() => void run({ type: "email-resend" })}>
          {text("重新发送验证邮件", "Resend verification email")}
        </button>
      </>
    );
  else if (modal?.type === "password")
    body = (
      <>
        <p>{passwordRequirements}</p>
        <form onSubmit={submit} className="form-stack">
          {field("current", text("当前密码", "Current password"), "password", "current-password")}
          {field("password", text("新密码", "New password"), "password", "new-password")}
          {field("confirm", text("确认新密码", "Confirm new password"), "password", "new-password")}
          {submitButton(text("修改密码", "Change password"))}
        </form>
      </>
    );
  else if (modal?.type === "passkey")
    body = (
      <>
        <p>
          {text(
            "浏览器将打开系统验证窗口。为通行密钥命名，便于识别。",
            "Your browser will open a system verification window. Name the passkey so you can recognize it.",
          )}
        </p>
        <form onSubmit={submit} className="form-stack">
          {field("name", text("通行密钥名称", "Passkey name"))}
          {submitButton(text("继续添加", "Continue"))}
        </form>
      </>
    );
  else if (modal?.type === "setup-mfa")
    body = (
      <>
        <p>
          {text(
            "使用验证器扫描二维码，或手动输入设置密钥。输入当前验证码完成设置。",
            "Scan the QR code with your authenticator, or enter the setup key manually. Enter its current code to finish setup.",
          )}
        </p>
        {setup.uri ? (
          <>
            <div className="kite-totp-qr">
              <QRCodeSVG value={setup.uri} size={180} marginSize={2} bgColor="#ffffff" fgColor="#0a0a0a" />
            </div>
            <div className="notice">
              <span>{text("设置密钥", "Setup key")}</span>
              <code className="kite-totp-secret">{setup.secret}</code>
            </div>
            <form onSubmit={submit} className="form-stack">
              <TextInput
                label={text("验证码", "Verification code")}
                value={values.code || ""}
                onChange={change("code")}
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                required
                disabled={busy}
              />
              {submitButton(text("验证并开启", "Verify and enable"))}
            </form>
          </>
        ) : (
          <div className="notice" role="status">
            {busy ? <Spinner /> : button(text("获取设置密钥", "Get setup key"), () => void run({ type: "totp-start" }))}
          </div>
        )}
      </>
    );
  else if (modal?.type === "reauth")
    body = (
      <>
        <p>
          {text(
            "继续操作前，请再次确认身份。验证在 10 分钟内有效。",
            "Confirm your identity before continuing. Verification stays valid for 10 minutes.",
          )}
        </p>
        {data.hasPassword && (
          <form onSubmit={submit} className="form-stack">
            {field("verifyPassword", text("当前密码", "Current password"), "password", "current-password")}
            {data.totp && (
              <TextInput
                label={text("验证器验证码", "Authenticator code")}
                value={values.verifyCode || ""}
                onChange={change("verifyCode")}
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                required
                disabled={busy}
              />
            )}
            {submitButton(text("验证身份", "Verify identity"))}
          </form>
        )}
        {data.passkeys.some((key) => key.state === "AUTH_FACTOR_STATE_READY") && (
          <Button variant={ButtonVariants.Secondary} disabled={busy} onClick={() => void verifyIdentity(true)}>
            <Icon name="Fingerprint" />
            {text("使用通行密钥验证", "Verify with passkey")}
          </Button>
        )}
      </>
    );
  else if (modal) {
    const descriptions: Partial<Record<Dialog["type"], [string, string]>> = {
      "remove-key": [
        "移除后，将无法再使用这枚通行密钥登录。",
        "You won't be able to sign in with this passkey after removing it.",
      ],
      "disable-mfa": [
        "关闭后，密码登录将不再要求验证器验证码。",
        "Password sign-in will no longer require an authenticator code.",
      ],
      "revoke-device": [
        "这个设备需要重新登录才能继续使用账号。",
        "This device will need to sign in again to use your account.",
      ],
      "revoke-all": [
        "其他设备的登录会话将结束；当前设备会保留。",
        "Other signed-in sessions will end. This device stays signed in.",
      ],
      "delete-final": [
        "你的 Kite Plus 账号将永久删除，关联服务会按各自规则处理账号数据。",
        "Your Kite Plus account will be permanently deleted. Connected services handle data under their own rules.",
      ],
    };
    const action = () => {
      if (modal.type === "remove-key") void run({ type: "passkey-remove", id: modal.id || "" });
      if (modal.type === "disable-mfa") void run({ type: "totp-remove" });
      if (modal.type === "revoke-device") void run({ type: "device-revoke", id: modal.id || "" });
      if (modal.type === "revoke-all") void run({ type: "devices-revoke" });
      if (modal.type === "delete-final") void run({ type: "delete", confirmation });
    };
    body = (
      <>
        <p>{text(...(descriptions[modal.type] || ["", ""]))}</p>
        <div className="notice">{modal.name || data.email}</div>
        <div className="modal-actions">
          {cancel}
          <Button
            color={
              ["remove-key", "disable-mfa", "delete-final"].includes(modal.type) ? ButtonColors.Warn : ButtonColors.Neutral
            }
            disabled={busy}
            onClick={action}
          >
            {busy && <Spinner />}
            {text("确认", "Confirm")}
          </Button>
        </div>
      </>
    );
  }
  return (
    <>
      <main className="account-main">
        <nav className="account-nav" aria-label={text("账号设置", "Account settings")}>
          <span className="nav-caption">{text("账号设置", "ACCOUNT SETTINGS")}</span>
          {navigation.map(([id, icon]) => (
            <Link
              key={id}
              className="nav-link"
              href={id === "profile" ? "/account" : `/account/${id}`}
              aria-current={section === id ? "page" : undefined}
            >
              <Icon name={icon} />
              {text(...titles[id])}
            </Link>
          ))}
          <div className="nav-separator" />
          <Link className="nav-link logout-nav" href="/logout">
            <Icon name="LogOut" />
            {text("退出账号", "Sign out")}
          </Link>
        </nav>
        <div className="account-content">
          <div className="page-heading">
            <h1>{text(...titles[section])}</h1>
            <p>{text(...descriptions[section])}</p>
          </div>
          {!data.emailVerified && section !== "profile" && (
            <div className="notice">
              {text("邮箱尚未验证，请先前往个人资料完成验证。", "Your email is unverified. Open your profile to verify it.")}{" "}
              <Link className="text-button" href="/account">
                {text("验证邮箱", "Verify email")}
              </Link>
            </div>
          )}
          {error && !modal && <Alert>{error}</Alert>}
          {content}
        </div>
      </main>
      {modal && (
        <dialog
          ref={dialogRef}
          className="modal kite-dialog"
          aria-labelledby="account-dialog-title"
          onCancel={(event) => {
            event.preventDefault();
            void close();
          }}
        >
          <button
            type="button"
            className="icon-button close-modal"
            disabled={busy}
            aria-label={text("关闭", "Close")}
            onClick={() => void close()}
          >
            <X size={18} strokeWidth={1.6} absoluteStrokeWidth />
          </button>
          <h2 id="account-dialog-title">{text(...dialogTitles[modal.type])}</h2>
          {error && <Alert>{error}</Alert>}
          {body}
        </dialog>
      )}
      {toast && (
        <div className="toast" role="status">
          <Icon name="Check" size={16} />
          {toast}
        </div>
      )}
    </>
  );
}
