import { DynamicTheme } from "@/components/dynamic-theme";
import { Translated } from "@/components/translated";
import { getServiceConfig } from "@/lib/service-url";
import { getBrandingSettings } from "@/lib/zitadel";
import Link from "next/link";
import { getLocale } from "next-intl/server";
import { headers } from "next/headers";

export default async function Page(props: { searchParams: Promise<any> }) {
  const zh = await getLocale() === "zh";
  const searchParams = await props.searchParams;

  const _headers = await headers();
  const { serviceConfig } = getServiceConfig(_headers);

  const { organization } = searchParams;

  const branding = await getBrandingSettings({ serviceConfig, organization });

  return (
    <DynamicTheme branding={branding}>
      <div className="flex flex-col space-y-4">
        <h1>
          <Translated i18nKey="success.title" namespace="logout" />
        </h1>
        <p className="ztdl-p mb-6 block">
          <Translated i18nKey="success.description" namespace="logout" />
        </p>
      </div>
      <Link className="button outline full-width" href="/loginname">{zh ? "重新登录" : "Sign in again"}</Link>
    </DynamicTheme>
  );
}
