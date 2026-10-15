import { CheckIcon } from "@/components/kite-icons";
import { clsx } from "clsx";
import { Fingerprint, KeyRound, Mail, ShieldCheck, Smartphone, Usb } from "lucide-react";
import Link from "next/link";
import { ReactNode } from "react";
import { BadgeState, StateBadge } from "./state-badge";
import { Translated } from "./translated";

const cardClasses = (alreadyAdded: boolean) =>
  clsx(
    "relative bg-background-light-400 dark:bg-background-dark-400 group block space-y-1.5 rounded-md px-5 py-3  border border-divider-light dark:border-divider-dark transition-all ",
    alreadyAdded ? "opacity-50 cursor-default" : "hover:shadow-lg hover:dark:bg-white/10",
  );

const LinkWrapper = ({ alreadyAdded, children, link }: { alreadyAdded: boolean; children: ReactNode; link: string }) => {
  return !alreadyAdded ? (
    <Link href={link} className={cardClasses(alreadyAdded)}>
      {children}
    </Link>
  ) : (
    <div className={cardClasses(alreadyAdded)}>{children}</div>
  );
};

export const TOTP = (alreadyAdded: boolean, link: string) => {
  return (
    <LinkWrapper key={link} alreadyAdded={alreadyAdded} link={link}>
      <div className={clsx("flex items-center font-medium", alreadyAdded ? "opacity-50" : "")}>
        <ShieldCheck className="mr-3" size={20} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />{" "}
        <Translated i18nKey="methods.totp" namespace="authenticator" />
      </div>
      {alreadyAdded && (
        <>
          <Setup />
        </>
      )}
    </LinkWrapper>
  );
};

export const U2F = (alreadyAdded: boolean, link: string) => {
  return (
    <LinkWrapper key={link} alreadyAdded={alreadyAdded} link={link}>
      <div className={clsx("flex items-center font-medium", alreadyAdded ? "" : "")}>
        <Usb className="mr-3" size={20} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
        <Translated i18nKey="methods.u2f" namespace="authenticator" />
      </div>
      {alreadyAdded && (
        <>
          <Setup />
        </>
      )}
    </LinkWrapper>
  );
};

export const EMAIL = (alreadyAdded: boolean, link: string) => {
  return (
    <LinkWrapper key={link} alreadyAdded={alreadyAdded} link={link}>
      <div className={clsx("flex items-center font-medium", alreadyAdded ? "" : "")}>
        <Mail className="mr-3" size={20} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />

        <Translated i18nKey="methods.otpEmail" namespace="authenticator" />
      </div>
      {alreadyAdded && (
        <>
          <Setup />
        </>
      )}
    </LinkWrapper>
  );
};

export const SMS = (alreadyAdded: boolean, link: string) => {
  return (
    <LinkWrapper key={link} alreadyAdded={alreadyAdded} link={link}>
      <div className={clsx("flex items-center font-medium", alreadyAdded ? "" : "")}>
        <Smartphone className="mr-3" size={20} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
        <Translated i18nKey="methods.otpSms" namespace="authenticator" />
      </div>
      {alreadyAdded && (
        <>
          <Setup />
        </>
      )}
    </LinkWrapper>
  );
};

export const PASSKEYS = (alreadyAdded: boolean, link: string) => {
  return (
    <LinkWrapper key={link} alreadyAdded={alreadyAdded} link={link}>
      <div className={clsx("flex items-center font-medium", alreadyAdded ? "" : "")}>
        <Fingerprint className="mr-3" size={20} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
        <Translated i18nKey="methods.passkey" namespace="authenticator" />
      </div>
      {alreadyAdded && (
        <>
          <Setup />
        </>
      )}
    </LinkWrapper>
  );
};

export const PASSWORD = (alreadyAdded: boolean, link: string) => {
  return (
    <LinkWrapper key={link} alreadyAdded={alreadyAdded} link={link}>
      <div className={clsx("flex items-center font-medium", alreadyAdded ? "" : "")}>
        <KeyRound className="mr-3" size={20} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
        <Translated i18nKey="methods.password" namespace="authenticator" />
      </div>
      {alreadyAdded && (
        <>
          <Setup />
        </>
      )}
    </LinkWrapper>
  );
};

function Setup() {
  return (
    <div className="absolute top-0 right-2 transform">
      <StateBadge evenPadding={true} state={BadgeState.Success}>
        <CheckIcon className="h-4 w-4" />
      </StateBadge>
    </div>
  );
}
