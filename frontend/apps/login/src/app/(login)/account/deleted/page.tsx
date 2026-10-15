import { KiteAuthFrame } from "@/components/kite-auth-frame";
import { StateMark } from "@/components/kite-state-mark";
import { getLocale } from "next-intl/server";
import Link from "next/link";
export default async function Page() {
  const zh = (await getLocale()) === "zh";
  return (
    <KiteAuthFrame>
      <section className="auth-card">
        <StateMark icon="UserRoundMinus" />
        <div className="flow-heading">
          <h1>{zh ? "账号已删除" : "Account deleted"}</h1>
          <p>
            {zh
              ? "你的 Kite Plus 账号已删除。关联服务会按各自规则处理账号数据。"
              : "Your Kite Plus account has been deleted. Connected services handle data under their own rules."}
          </p>
        </div>
        <Link className="button full-width outline" href="/loginname">
          {zh ? "返回登录" : "Back to sign-in"}
        </Link>
      </section>
    </KiteAuthFrame>
  );
}
