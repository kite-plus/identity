import { timestampDate } from "@zitadel/client";
import { Session } from "@zitadel/proto/zitadel/session/v2/session_pb";
import { hasVerifiedPrimaryFactor } from "./session";

const FRESH_AUTHENTICATION_MS = 10 * 60 * 1000;

export function hasFreshAccountAuthentication(session: Session, now = Date.now()): boolean {
  if (!hasVerifiedPrimaryFactor(session).valid) return false;
  const factors = session.factors;
  const primary = [
    factors?.password?.verifiedAt,
    factors?.intent?.verifiedAt,
    factors?.webAuthN?.userVerified ? factors.webAuthN.verifiedAt : undefined,
  ].filter((value) => value !== undefined);
  const fresh = primary.some((value) => {
    const age = now - timestampDate(value).getTime();
    return age >= 0 && age < FRESH_AUTHENTICATION_MS;
  });
  if (!fresh) return false;
  if (
    factors?.webAuthN?.userVerified &&
    factors.webAuthN.verifiedAt &&
    now - timestampDate(factors.webAuthN.verifiedAt).getTime() < FRESH_AUTHENTICATION_MS
  )
    return true;
  const secondary = [
    factors?.totp?.verifiedAt,
    factors?.otpEmail?.verifiedAt,
    factors?.otpSms?.verifiedAt,
    !factors?.webAuthN?.userVerified ? factors?.webAuthN?.verifiedAt : undefined,
  ].filter((value) => value !== undefined);
  return (
    secondary.length === 0 ||
    secondary.some((value) => {
      const age = now - timestampDate(value).getTime();
      return age >= 0 && age < FRESH_AUTHENTICATION_MS;
    })
  );
}
