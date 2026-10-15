"use client";

import { Compass, MessageSquare, Smartphone } from "lucide-react";
import { useLocale } from "next-intl";
import { ReactNode, SVGProps, useEffect, useRef } from "react";

// Seeded so the server and the browser draw the same night sky.
const STARS = (() => {
  let seed = 7;
  const next = () => (seed = (seed * 16807) % 2147483647) / 2147483647;
  return Array.from({ length: 90 }, () => ({
    cx: Math.round(next() * 1280),
    cy: Math.round(next() * 800),
    r: +(0.5 + next()).toFixed(1),
    opacity: +(0.15 + next() * 0.55).toFixed(2),
  }));
})();

function Cloud({ className, shape = "a", under = true }: { className: string; shape?: "a" | "b"; under?: boolean }) {
  const body =
    shape === "a" ? (
      <>
        <rect y="44" width="236" height="46" rx="23" />
        <circle cx="74" cy="48" r="38" />
        <circle cx="138" cy="36" r="48" />
        <circle cx="192" cy="58" r="32" />
      </>
    ) : (
      <>
        <rect y="44" width="236" height="46" rx="23" />
        <circle cx="62" cy="50" r="34" />
        <circle cx="126" cy="38" r="46" />
        <circle cx="186" cy="56" r="34" />
      </>
    );
  return (
    <svg className={`kite-cloud ${className}`} viewBox="-6 -16 256 120">
      {under && (
        <g className="cloud-under" transform="translate(10 9)">
          {body}
        </g>
      )}
      <g className="cloud-body">{body}</g>
    </svg>
  );
}

export function KiteFlying(props: SVGProps<SVGSVGElement>) {
  return (
    <svg className="kite-drawing" viewBox="0 8 64 64" {...props}>
      <g transform="translate(32 32) scale(.82) translate(-32 -32)" strokeLinejoin="round" strokeWidth="5">
        <path className="kite-a" d="M10 14.5 L27 21 L27 30 L10 23.5 Z" />
        <path className="kite-b" d="M10 32 L27 38.5 L27 49 L10 42.5 Z" />
        <path className="kite-c" d="M37 21 L54 14.5 L54 42.5 L37 49 Z" />
        <path className="kite-shine" d="M10 14.5 L27 21 L10 23.5 Z" />
        <path className="kite-shine dim" d="M37 21 L54 14.5 L37 49 Z" />
        <path className="kite-shine dim" d="M10 32 L27 38.5 L10 42.5 Z" />
        <path className="kite-tail" d="M10 44 C 6 50, 12 54, 7 60 S 9 68, 3 74" strokeWidth=".5" />
        <path className="kite-bow" d="M7.6 51.4 l2.6 -1.4 l.2 2.6 z" />
        <path className="kite-bow gold" d="M6.6 59.6 l2.6 -1.2 l.1 2.6 z" />
        <path className="kite-bow" d="M5.2 67.8 l2.6 -1.2 l.1 2.6 z" />
      </g>
    </svg>
  );
}

/** The sign-in pages' frame: the sky, an introduction, and a kite whose string is tied to the card. */
export function KiteAuthFrame({ children }: { children: ReactNode }) {
  const zh = useLocale() === "zh";
  const anchor = useRef<HTMLSpanElement>(null);
  const knot = useRef<HTMLSpanElement>(null);
  const line = useRef<SVGPathElement>(null);

  useEffect(() => {
    const update = () => {
      const path = line.current;
      const svg = path?.ownerSVGElement;
      const from = anchor.current;
      const to = knot.current;
      if (!path || !svg || !from || !to) return;
      if (!from.getClientRects().length || !to.getClientRects().length) {
        path.setAttribute("d", "");
        return;
      }
      const o = svg.getBoundingClientRect();
      const k = from.getBoundingClientRect();
      const t = to.getBoundingClientRect();
      const kx = k.left - o.left;
      const ky = k.top - o.top;
      const tx = t.left + t.width / 2 - o.left;
      const ty = t.top + t.height / 2 - o.top;
      // Leaves the knot almost level, then rises to the kite, like a line in the wind.
      const c1x = tx + (kx - tx) * 0.55;
      const c1y = ty + Math.min(40, Math.abs(kx - tx) * 0.12);
      const c2x = kx - (kx - tx) * 0.06;
      const c2y = ky + (ty - ky) * 0.45;
      path.setAttribute("d", `M${tx} ${ty} C${c1x} ${c1y}, ${c2x} ${c2y}, ${kx} ${ky}`);
    };
    const still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    let frame = 0;
    const loop = () => {
      update();
      frame = requestAnimationFrame(loop);
    };
    if (still) update();
    else loop();
    const observer = new ResizeObserver(update);
    observer.observe(document.body);
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, []);

  const places = [
    {
      icon: <Compass className="icon" strokeWidth={1.8} aria-hidden="true" />,
      tone: "",
      name: "Explore",
      text: zh ? "订阅独立博客，收藏想读的文章" : "Follow independent blogs and save posts to read",
    },
    {
      icon: <MessageSquare className="icon" strokeWidth={1.8} aria-hidden="true" />,
      tone: "red",
      name: zh ? "Kite 评论" : "Kite Comments",
      text: zh ? "在使用 Kite 评论的博客下留言" : "Comment on blogs that use Kite Comments",
    },
    {
      icon: <Smartphone className="icon" strokeWidth={1.8} aria-hidden="true" />,
      tone: "gold",
      name: zh ? "Kite 应用" : "Kite app",
      text: zh ? "iPhone 与 Android，即将上架" : "iPhone and Android, coming soon",
    },
  ];

  return (
    <>
      <div className="kite-sky" aria-hidden="true">
        <svg className="kite-stars" viewBox="0 0 1280 800" preserveAspectRatio="xMidYMid slice">
          {STARS.map((star, index) => (
            <circle key={index} {...star} />
          ))}
        </svg>
        <svg className="kite-moon" viewBox="-90 -90 180 180">
          <circle r="90" className="moon-glow" />
          <circle r="25" className="moon-face" />
          <circle cx="13" cy="-9" r="23" className="moon-bite" />
        </svg>
        <Cloud className="cloud-low" />
        <Cloud className="cloud-high" shape="b" under={false} />
        <Cloud className="cloud-right" shape="b" />
      </div>
      <main className="auth-main">
        <div className="auth-layout">
          <aside className="auth-intro">
            <p className="eyebrow">{zh ? "KITE PLUS 账号" : "KITE PLUS ACCOUNT"}</p>
            <h2 className="auth-intro-title">
              {zh ? (
                <>
                  一个账号，
                  <br />
                  <span className="wavy">通行</span> Kite Plus
                </>
              ) : (
                <>
                  One account,
                  <br />
                  <span className="wavy">all</span> of Kite Plus
                </>
              )}
            </h2>
            <p className="auth-intro-lede">
              {zh
                ? "在 Explore 订阅独立博客、在独立博客下留言、用 Kite 手机应用阅读，都用这一个账号登录。"
                : "Follow blogs on Explore, comment on independent blogs and read in the Kite app, all with this one account."}
            </p>
            <ul className="auth-places">
              {places.map((place) => (
                <li key={place.name}>
                  <span className={`seal ${place.tone}`}>{place.icon}</span>
                  <div>
                    <b>{place.name}</b>
                    <small>{place.text}</small>
                  </div>
                </li>
              ))}
            </ul>
          </aside>
          <div className="auth-wrap">
            <div className="kite-fly" aria-hidden="true">
              <KiteFlying />
              <span ref={anchor} className="kite-anchor" />
            </div>
            <span ref={knot} className="kite-eyelet" aria-hidden="true" />
            {children}
            <p className="auth-tagline">
              {zh
                ? "一个账号，通行 Explore、Kite 评论和 Kite 应用"
                : "One account for Explore, Kite Comments and the Kite app"}
            </p>
          </div>
        </div>
      </main>
      <svg className="kite-string" aria-hidden="true">
        <path ref={line} />
      </svg>
    </>
  );
}
