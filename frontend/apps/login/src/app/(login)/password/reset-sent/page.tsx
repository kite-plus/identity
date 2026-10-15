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
      </section>
    </KiteAuthFrame>
  );
}
