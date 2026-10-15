import { KiteShell } from "@/components/kite-shell";
import { LanguageProvider } from "@/components/language-provider";
import { ThemeProvider } from "@/components/theme-provider";
import { LANGS } from "@/lib/i18n";
import { getServiceConfig } from "@/lib/service-url";
import { getAllowedLanguages, getLegalAndSupportSettings } from "@/lib/zitadel";
import "@/styles/globals.scss";
import "@/styles/kite.css";
import * as Tooltip from "@radix-ui/react-tooltip";
import type { Metadata } from "next";
import { getLocale } from "next-intl/server";
import { headers } from "next/headers";
import React, { Suspense } from "react";

export async function generateMetadata(): Promise<Metadata> {
  const zh = (await getLocale()) === "zh";
  return { title: zh ? "Kite Plus 账号" : "Kite Plus Account", robots: { index: false, follow: false } };
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const { serviceConfig } = getServiceConfig(await headers());
  let languages = LANGS.filter((language) => language.code === "zh" || language.code === "en");
  let links: { terms?: string; privacy?: string; help?: string } = {};
  try {
    const [settings, legal] = await Promise.all([
      getAllowedLanguages({ serviceConfig }),
      getLegalAndSupportSettings({ serviceConfig }),
    ]);
    if (settings.allowedLanguages?.length)
      languages = languages.filter((language) => settings.allowedLanguages.includes(language.code));
    if (legal) links = { terms: legal.tosLink, privacy: legal.privacyPolicyLink, help: legal.helpLink };
  } catch {
    /* Still render a usable page while the API is down. */
  }
  const locale = await getLocale();
  return (
    <html lang={locale} suppressHydrationWarning>
      <head />
      <body>
        <ThemeProvider>
          <Tooltip.Provider>
            <Suspense fallback={<div className="app" aria-busy="true" />}>
              <LanguageProvider>
                <KiteShell languages={languages} links={links}>
                  {children}
                </KiteShell>
              </LanguageProvider>
            </Suspense>
          </Tooltip.Provider>
        </ThemeProvider>
      </body>
    </html>
  );
}
