"use client";

import { setLanguageCookie } from "@/lib/cookies";
import { Lang } from "@/lib/i18n";
import { Monitor, Moon, Sun } from "lucide-react";
import { useLocale } from "next-intl";
import { useTheme } from "next-themes";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ReactNode, useEffect, useState } from "react";

export function KiteShell({
  children,
  languages,
  links,
}: {
  children: ReactNode;
  languages: Lang[];
  links: { terms?: string; privacy?: string; help?: string };
}) {
  const locale = useLocale();
  const zh = locale === "zh";
  const router = useRouter();
  const settings = /^\/account(\/(security|devices|delete))?\/?$/.test(usePathname());
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  const [changing, setChanging] = useState(false);
  useEffect(() => setMounted(true), []);
  const ThemeIcon = mounted && theme === "dark" ? Moon : mounted && theme === "light" ? Sun : Monitor;
  return (
    <div className={`app ${settings ? "app-settings" : "app-auth"}`}>
      <header className="site-header">
        <div className="header-inner">
          <Link className="brand" href="/account">
            <img src={`${process.env.NEXT_PUBLIC_BASE_PATH || ""}/kite.svg`} width={28} height={28} alt="" />
            <span className="brand-name">Kite Plus</span>
            <span className="brand-sub">{zh ? "账号" : "Account"}</span>
          </Link>
          <div className="header-actions">
            <select
              className="language"
              aria-label={zh ? "界面语言" : "Language"}
              value={locale}
              disabled={changing}
              onChange={async (event) => {
                setChanging(true);
                try {
                  await setLanguageCookie(event.target.value);
                  router.refresh();
                } finally {
                  setChanging(false);
                }
              }}
            >
              {languages.map((language) => (
                <option key={language.code} value={language.code}>
                  {language.name}
                </option>
              ))}
            </select>
            <button
              className="icon-button"
              type="button"
              aria-label={zh ? "切换显示主题" : "Change display theme"}
              onClick={() => setTheme(theme === "light" ? "dark" : theme === "dark" ? "system" : "light")}
            >
              <ThemeIcon size={18} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
            </button>
          </div>
        </div>
      </header>
      <div className="kite-content">{children}</div>
      <footer className="site-footer">
        <span>Kite Plus {zh ? "账号" : "Account"}</span>
        <div className="footer-links">
          {links.terms && <a href={links.terms}>{zh ? "服务条款" : "Terms"}</a>}
          {links.privacy && <a href={links.privacy}>{zh ? "隐私政策" : "Privacy"}</a>}
          {links.help && <a href={links.help}>{zh ? "帮助" : "Help"}</a>}
        </div>
      </footer>
    </div>
  );
}
