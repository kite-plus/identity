"use client";
import { Button } from "@/components/button";
import { KiteAuthFrame } from "@/components/kite-auth-frame";
import { StateMark } from "@/components/kite-state-mark";
import { useLocale } from "next-intl";
import Link from "next/link";
export default function Error({ reset }: { error: Error; reset: () => void }) {
  const zh = useLocale() === "zh";
  return (
    <KiteAuthFrame>
      <section className="auth-card">
        <StateMark icon="CloudOff" />
        <div className="flow-heading">
          <h1>{zh ? "账号服务暂不可用" : "Account service unavailable"}</h1>
          <p>
            {zh
              ? "账号服务暂时无法响应。请稍后重试，你已打开的应用可以继续使用。"
              : "The account service isn't responding. Try again later. Apps you've already opened can continue working."}
          </p>
        </div>
        <div className="form-stack">
          <Button onClick={reset}>{zh ? "重新尝试" : "Try again"}</Button>
          <Link className="button outline" href="/loginname">
            {zh ? "返回登录" : "Back to sign-in"}
          </Link>
        </div>
      </section>
    </KiteAuthFrame>
  );
}
