import { Link2Off } from "lucide-react";
import React from "react";

function CheckBadge({ children }: { children?: React.ReactNode }) {
  return (
    <>
      <circle cx="25" cy="25" r="6" className="glyph-badge" />
      {children || <path d="m22.5 25 1.8 1.8 3.3-3.4" className="glyph-accent" />}
    </>
  );
}

export function StateGlyph({ icon, tone = "" }: { icon: string; tone?: string }) {
  let drawing;
  if (icon === "Mail" || icon === "MailCheck") {
    drawing = (
      <>
        <rect x="4" y="7" width="24" height="18" rx="4" className="glyph-body" />
        <path d="m5 10 9.1 6.5a3.2 3.2 0 0 0 3.8 0L27 10" className="glyph-accent" />
        <path d="m5.5 23 6.2-5M26.5 23l-6.2-5" className="glyph-detail" />
        {icon === "MailCheck" && (
          <CheckBadge>
            {tone === "success" ? null : <path d="M22.6 27.2 27 22.8m-4.2 0H27V27" className="glyph-accent" />}
          </CheckBadge>
        )}
      </>
    );
  } else if (icon === "KeyRound") {
    drawing = (
      <>
        <circle cx="20.8" cy="11.4" r="7.3" className="glyph-body" />
        <circle cx="22.7" cy="9.5" r="1.5" className="glyph-dot" />
        <path d="m15.4 16.5-11 11M10 22l-3-3m-.2 6.2-2.8-2.8" className="glyph-accent" />
        {tone === "success" && <CheckBadge />}
      </>
    );
  } else if (icon === "ShieldCheck") {
    drawing = (
      <>
        <path
          d="M16 3.5c3.3 2.5 6.8 3.7 10.5 4.2v8.1c0 6.2-4 10.1-10.5 13.1C9.5 25.9 5.5 22 5.5 15.8V7.7C9.2 7.2 12.7 6 16 3.5Z"
          className="glyph-body"
        />
        <path d="m11.2 15.8 3.3 3.3 6.3-6.3" className="glyph-accent" />
      </>
    );
  } else if (icon === "CloudOff") {
    drawing = (
      <>
        <path d="M8.2 25a6.7 6.7 0 0 1-1-13.3A9.2 9.2 0 0 1 24 10.5 7.3 7.3 0 0 1 24 25H8.2Z" className="glyph-body" />
        <path d="m5.5 5.5 21 21" className="glyph-knockout" />
        <path d="m5.5 5.5 21 21" className="glyph-accent" />
      </>
    );
  } else if (icon === "DoorOpen") {
    drawing = (
      <>
        <path d="M7 4h17a2 2 0 0 1 2 2v22H7V4Z" className="glyph-detail" />
        <path d="M7 4 20 7v22L7 26V4Z" className="glyph-body" />
        <path d="M21.5 16h7m-3-3 3 3-3 3" className="glyph-accent" />
        <circle cx="16" cy="16" r="1" className="glyph-dot" />
      </>
    );
  } else if (icon === "UserRoundMinus") {
    drawing = (
      <>
        <circle cx="13" cy="10" r="5.5" className="glyph-body" />
        <path d="M4 27v-2.5a8.8 8.8 0 0 1 8.8-8.8h.4a8.8 8.8 0 0 1 8.8 8.8V27" className="glyph-detail" />
        <CheckBadge>
          <path d="M22.5 25h5" className="glyph-accent" />
        </CheckBadge>
      </>
    );
  } else if (icon === "Fingerprint") {
    drawing = (
      <>
        <ellipse cx="16" cy="15" rx="8" ry="10" fill="var(--glyph-fill)" />
        <path
          d="M5 13a11 11 0 0 1 22 0M5 17v-1M8 24c.8-2.8 1-5.8 1-10a7 7 0 0 1 14 0c0 4 0 7-.7 9M12 28c1.1-4.1 1.5-8.1 1.5-14a2.5 2.5 0 0 1 5 0c0 5.8-.2 9.8-1.7 14M5 22c.3-1.4.5-2.7.5-4.2M24 28c.9-2.4 1.4-5.1 1.6-8.1"
          className="glyph-accent"
        />
      </>
    );
  } else if (icon === "Link2Off") {
    return (
      <div className="state-glyph">
        <span className="link-wash"></span>
        <Link2Off size={32} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
      </div>
    );
  } else {
    drawing = (
      <>
        <circle cx="16" cy="16" r="11" className="glyph-body" />
        <path d="m10.5 16 3.7 3.7 7.3-7.3" className="glyph-accent" />
      </>
    );
  }
  return (
    <svg
      className="state-glyph"
      viewBox="0 0 32 32"
      width="32"
      height="32"
      fill="none"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {drawing}
    </svg>
  );
}

export function StateMark({ icon, tone = "" }: { icon: string; tone?: string }) {
  return (
    <div className={`state-mark ${tone}`} data-symbol={icon}>
      <StateGlyph icon={icon} tone={tone} />
    </div>
  );
}
