"use client";

import { BrandingSettings } from "@zitadel/proto/zitadel/settings/v2/branding_settings_pb";
import { useLocale } from "next-intl";
import { usePathname } from "next/navigation";
import { Children, ReactNode } from "react";
import { KiteAuthFrame } from "./kite-auth-frame";
import { StateMark } from "./kite-state-mark";
import { ThemeWrapper } from "./theme-wrapper";

export function DynamicTheme({
  branding,
  children,
}: {
  children: ReactNode | ((isSideBySide: boolean) => ReactNode);
  branding?: BrandingSettings;
}) {
  const pathname = usePathname();
  const locale = useLocale();
  const actualChildren = typeof children === "function" ? children(false) : children;
  const parts = Children.toArray(actualChildren);
  const symbol = pathname.includes("passkey")
    ? "Fingerprint"
    : pathname.includes("authenticator") || pathname.includes("mfa") || pathname.includes("otp") || pathname.includes("u2f")
      ? "ShieldCheck"
      : pathname.includes("verify")
        ? pathname.includes("success")
          ? "MailCheck"
          : "Mail"
        : pathname.includes("logout")
          ? "DoorOpen"
          : pathname.includes("password/set") || pathname.includes("password/change")
            ? "KeyRound"
            : pathname.includes("signedin")
              ? "ShieldCheck"
              : undefined;
  return (
    <ThemeWrapper branding={branding}>
      <KiteAuthFrame>
        <section className="auth-card">
          <div className="flow-context">
            <img src={`${process.env.NEXT_PUBLIC_BASE_PATH || ""}/kite.svg`} width={16} height={16} alt="" />
            <span>{locale === "zh" ? "使用 Kite Plus 账号继续" : "Continue with your Kite Plus account"}</span>
          </div>
          {symbol && (
            <StateMark icon={symbol} tone={pathname.includes("success") || pathname.includes("signedin") ? "success" : ""} />
          )}
          {parts.length > 1 ? (
            <>
              <div className="flow-heading">{parts[0]}</div>
              <div className="kite-auth-form">{parts.slice(1)}</div>
            </>
          ) : (
            actualChildren
          )}
        </section>
      </KiteAuthFrame>
    </ThemeWrapper>
  );
}
