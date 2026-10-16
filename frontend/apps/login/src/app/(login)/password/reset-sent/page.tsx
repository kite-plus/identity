import { KiteAuthFrame } from "@/components/kite-auth-frame";
import { StateMark } from "@/components/kite-state-mark";
import { getLocale } from "next-intl/server";
import Link from "next/link";
export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{ loginName?: string; organization?: string; requestId?: string }>;
}) {
  const params = await searchParams;
  const zh = (await getLocale()) === "zh";
  // The reset page never says whether an account exists, so this is where
  // a reader without one (often an old Explore password account) learns why.
  const email = params.loginName?.includes("@") ? params.loginName : undefined;
  const signUp = new URLSearchParams();
  if (email) signUp.set("email", email);
  if (params.organization) signUp.set("organization", params.organization);
  if (params.requestId) signUp.set("requestId", params.requestId);
  return (
    <KiteAuthFrame>
      <section className="auth-card">
        <StateMark icon="MailCheck" />
        <div className="flow-heading">
          <h1>{zh ? "检查你的邮箱" : "Check your inbox"}</h1>
          <p>
            {zh
              ? "如果账号存在，我们已向对应邮箱发送重置说明。请点击邮件中的链接，或输入验证码继续。"
              : "If this account exists, we've sent reset instructions to its email. Open the link in the email or enter the code to continue."}
          </p>
        </div>
        <div className="form-stack">
          <Link className="button primary full-width" href={`/password/set?${new URLSearchParams(params)}`}>
            {zh ? "输入邮件验证码" : "Enter email code"}
          </Link>
          <Link className="button full-width outline" href={`/password/forgot?${new URLSearchParams(params)}`}>
            {zh ? "重新发送或更换账号" : "Resend or use another account"}
          </Link>
        </div>
        <div className="reset-help">
          <h2>{zh ? "没有收到邮件？" : "No email?"}</h2>
          <ul>
            <li>{zh ? "看看垃圾邮件箱，邮件可能要等一两分钟。" : "Check your spam folder; it can take a minute or two."}</li>
            <li>
              {zh
                ? "这个邮箱可能还没有 Kite Plus 账号。以前在 Explore 用密码注册的读者，用同一个邮箱注册 Kite Plus 账号，再登录 Explore，订阅都还在。"
                : "This email may not have a Kite Plus account yet. If you signed up on Explore with a password, sign up here with the same email, then sign in to Explore: your follows are still there."}
            </li>
          </ul>
          <Link className="text-button" href={`/register?${signUp}`}>
            {email
              ? zh
                ? "用这个邮箱注册"
                : "Sign up with this email"
              : zh
                ? "注册 Kite Plus 账号"
                : "Create a Kite Plus account"}
          </Link>
        </div>
      </section>
    </KiteAuthFrame>
  );
}
