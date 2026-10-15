import { KiteAuthFrame } from "@/components/kite-auth-frame";
import { StateMark } from "@/components/kite-state-mark";
import { getLocale } from "next-intl/server";
import Link from "next/link";
export default async function Page() {
  const zh = (await getLocale()) === "zh";
  return (
    <KiteAuthFrame>
      <section className="auth-card">
        <StateMark icon="Link2Off" />
        <div className="flow-heading">
          <h1>{zh ? "登录已过期" : "Your session has expired"}</h1>
          <p>{zh ? "为了保护你的账号，请重新登录后继续。" : "Sign in again to continue and keep your account secure."}</p>
        </div>
        <Link className="button primary full-width" href="/loginname">
          {zh ? "重新登录" : "Sign in again"}
        </Link>
      </section>
    </KiteAuthFrame>
  );
}
