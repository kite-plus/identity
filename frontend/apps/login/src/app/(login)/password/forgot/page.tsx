import { KiteAuthFrame } from "@/components/kite-auth-frame";
import { KiteRecoveryForm } from "@/components/kite-recovery-form";
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
        <StateMark icon="KeyRound" />
        <div className="flow-heading">
          <h1>{zh ? "找回密码" : "Reset your password"}</h1>
          <p>
            {zh
              ? "输入账号邮箱或用户名，我们会发送重置邮件。"
              : "Enter your email or username and we'll send a reset email."}
          </p>
        </div>
        <KiteRecoveryForm
          initialLoginName={params.loginName}
          organization={params.organization}
          requestId={params.requestId}
        />
        <Link className="back-link" href={`/loginname?${new URLSearchParams(params)}`}>
          {zh ? "返回登录" : "Back to sign-in"}
        </Link>
      </section>
    </KiteAuthFrame>
  );
}
