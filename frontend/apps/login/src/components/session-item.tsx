"use client";

import { ArrowRightIcon, XIcon } from "@/components/kite-icons";
import { handleServerActionResponse } from "@/lib/client-utils";
import { sendLoginname } from "@/lib/server/loginname";
import { clearSession, continueWithSession, ContinueWithSessionCommand } from "@/lib/server/session";
import { Timestamp, timestampDate } from "@zitadel/client";
import { Session } from "@zitadel/proto/zitadel/session/v2/session_pb";
import moment from "moment";
import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Alert } from "./alert";
import { AutoSubmitForm } from "./auto-submit-form";
import { getInitials } from "./avatar";

export function isSessionPrimaryFactorAndLifetimeValid(session: Partial<Session>): {
  valid: boolean;
  verifiedAt?: Timestamp;
} {
  const validPassword = session?.factors?.password?.verifiedAt;
  const validPasskey = session?.factors?.webAuthN?.verifiedAt;
  const validIDP = session?.factors?.intent?.verifiedAt;

  const stillValid = session.expirationDate ? timestampDate(session.expirationDate) > new Date() : true;

  const verifiedAt = validPassword || validPasskey || validIDP;
  const valid = !!((validPassword || validPasskey || validIDP) && stillValid);

  return { valid, verifiedAt };
}

export function SessionItem({ session, reload, requestId }: { session: Session; reload: () => void; requestId?: string }) {
  const currentLocale = useLocale();
  moment.locale(currentLocale === "zh" ? "zh-cn" : currentLocale);
  const t = useTranslations("error");
  const ta = useTranslations("accounts");

  const [loading, setLoading] = useState<boolean>(false);

  /**
   * Returns true when the session was removed (server-side and from the cookie).
   * On failure the error is shown and the card must stay in the list.
   */
  async function clearSessionId(id: string): Promise<boolean> {
    setLoading(true);
    setError(null);
    try {
      const response = await clearSession({ sessionId: id });
      if (response && "error" in response && response.error) {
        setError(response.error);
        return false;
      }
      return true;
    } catch {
      setError(t("couldNotClearSession"));
      return false;
    } finally {
      setLoading(false);
    }
  }

  const { valid, verifiedAt } = isSessionPrimaryFactorAndLifetimeValid(session);
  const [samlData, setSamlData] = useState<{ url: string; fields: Record<string, string> } | null>(null);

  const [error, setError] = useState<string | null>(null);

  const router = useRouter();

  const user = session.factors?.user;
  const name = user?.displayName || user?.loginName || "";

  async function choose() {
    if (valid && user) {
      const sessionPayload: ContinueWithSessionCommand = session;
      if (requestId) {
        sessionPayload.requestId = requestId;
      }
      setLoading(true);
      try {
        const callbackResponse = await continueWithSession(sessionPayload);
        handleServerActionResponse(callbackResponse, router, setSamlData, (e) => setError(e));
      } catch {
        setError(ta("failed"));
      } finally {
        setLoading(false);
      }
    } else if (user) {
      setLoading(true);
      try {
        const res = await sendLoginname({
          loginName: user.loginName,
          organization: user.organizationId,
          requestId: requestId,
        });
        handleServerActionResponse(res, router, setSamlData, (e) => setError(e));
      } catch {
        setError(ta("failed"));
      } finally {
        setLoading(false);
      }
    }
  }

  return (
    <>
      {samlData && <AutoSubmitForm url={samlData.url} fields={samlData.fields} />}
      <div className={`account-choice${valid ? "" : "stale"}`}>
        <button type="button" className="account-choice-main" onClick={choose} disabled={loading}>
          <span className="seal" aria-hidden="true">
            {getInitials(user?.displayName ?? "", user?.loginName ?? "")
              .charAt(0)
              .toUpperCase()}
          </span>
          <span className="account-choice-text">
            <b>{name}</b>
            {user?.displayName && <small>{user.loginName}</small>}
            {valid ? (
              <small className="account-choice-state">
                {ta("verified")} {verifiedAt && moment(timestampDate(verifiedAt)).fromNow()}
              </small>
            ) : (
              verifiedAt && (
                <small className="account-choice-state">
                  {ta("expired")} {session.expirationDate && moment(timestampDate(session.expirationDate)).fromNow()}
                </small>
              )
            )}
          </span>
          <span className="account-choice-go">
            {valid ? ta("continue") : ta("signInAgain")}
            <ArrowRightIcon className="icon" aria-hidden="true" />
          </span>
        </button>
        <button
          type="button"
          className="account-choice-remove"
          aria-label={ta("remove", { name })}
          title={ta("remove", { name })}
          disabled={loading}
          onClick={async () => {
            if (await clearSessionId(session.id)) {
              reload();
            }
          }}
        >
          <XIcon className="icon" aria-hidden="true" />
        </button>
      </div>
      {error && <Alert>{error}</Alert>}
    </>
  );
}
