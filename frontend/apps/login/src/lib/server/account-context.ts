import "server-only";

import { hasFreshAccountAuthentication } from "@/lib/account-policy";
import { AccountDevice } from "@/lib/account-types";
import { Cookie, getMostRecentSessionCookie, updateSessionCookie } from "@/lib/cookies";
import { applyCustomHeaders } from "@/lib/custom-headers";
import { getServiceConfig } from "@/lib/service-url";
import { hasVerifiedPrimaryFactor, isSessionValid } from "@/lib/session";
import { getSession, ServiceConfig, setSession } from "@/lib/zitadel";
import { Code, ConnectError, fromJson, JsonObject, timestampDate } from "@zitadel/client";
import { RequestChallengesSchema } from "@zitadel/proto/zitadel/session/v2/challenge_pb";
import { Session } from "@zitadel/proto/zitadel/session/v2/session_pb";
import { ChecksSchema } from "@zitadel/proto/zitadel/session/v2/session_service_pb";
import { headers } from "next/headers";

export class AccountAccessError extends Error {
  constructor(
    public reason: "login" | "fresh" | "unverified" | "invalid" | "unavailable" | "expired",
    public status = 0,
  ) {
    super(reason);
  }
}
export interface AccountContext {
  cookie: Cookie;
  session: Session;
  userId: string;
  serviceConfig: ServiceConfig;
  domain: string;
}

export async function requireAccountContext({ fresh = false }: { fresh?: boolean } = {}): Promise<AccountContext> {
  const cookie = await getMostRecentSessionCookie();
  if (!cookie) throw new AccountAccessError("login");
  const requestHeaders = await headers();
  const { serviceConfig } = getServiceConfig(requestHeaders);
  let session: Session | undefined;
  try {
    session = (await getSession({ serviceConfig, sessionId: cookie.id, sessionToken: cookie.token })).session;
  } catch (error) {
    if (error instanceof ConnectError && (error.code === Code.NotFound || error.code === Code.Unauthenticated))
      throw new AccountAccessError("expired");
    throw new AccountAccessError("unavailable");
  }
  if (session?.expirationDate && timestampDate(session.expirationDate).getTime() <= Date.now())
    throw new AccountAccessError("expired");
  if (!session?.factors?.user?.id || !hasVerifiedPrimaryFactor(session).valid) throw new AccountAccessError("login");
  if (!(await isSessionValid({ serviceConfig, session, checkEmailVerification: false })))
    throw new AccountAccessError("login");
  if (fresh && !hasFreshAccountAuthentication(session)) throw new AccountAccessError("fresh");
  const publicHost = serviceConfig.publicHost || requestHeaders.get("host") || "";
  const domain = new URL(`http://${publicHost}`).hostname;
  return { cookie, session, userId: session.factors.user.id, serviceConfig, domain };
}

export async function accountRequest<T>(context: AccountContext, path: string, method = "GET", body?: unknown): Promise<T> {
  const requestHeaders = new Headers({
    Authorization: `Bearer ${context.cookie.token}`,
    "Content-Type": "application/json",
  });
  if (context.serviceConfig.instanceHost) requestHeaders.set("x-zitadel-instance-host", context.serviceConfig.instanceHost);
  if (context.serviceConfig.publicHost) requestHeaders.set("x-zitadel-public-host", context.serviceConfig.publicHost);
  applyCustomHeaders({ set: (key, value) => requestHeaders.set(key, value), remove: (key) => requestHeaders.delete(key) });
  const response = await fetch(`${context.serviceConfig.baseUrl}${path}`, {
    method,
    headers: requestHeaders,
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
    signal: AbortSignal.timeout(15000),
  });
  if (!response.ok) {
    const failure = (await response.json().catch(() => ({}))) as { message?: string };
    if (response.status === 404 && typeof failure.message === "string" && failure.message.includes("membership not found")) {
      throw new AccountAccessError("invalid", 403);
    }
    throw new AccountAccessError(response.status >= 500 ? "unavailable" : "invalid", response.status);
  }
  return response.json() as Promise<T>;
}

interface SessionData {
  id: string;
  creationDate?: string;
  changeDate?: string;
  expirationDate?: string;
  factors?: { user?: { id: string }; password?: unknown; webAuthN?: { userVerified?: boolean }; intent?: unknown };
  userAgent?: { description?: string };
}
export async function listAccountSessions(context: AccountContext): Promise<SessionData[]> {
  const result: SessionData[] = [];
  for (let offset = 0; offset < 10000; offset += 100) {
    const page = await accountRequest<{ sessions?: SessionData[]; details?: { totalResult?: string } }>(
      context,
      "/v2/sessions/search",
      "POST",
      {
        query: { offset: String(offset), limit: 100 },
        queries: [{ userIdQuery: { id: context.userId } }],
      },
    );
    result.push(...(page.sessions || []).filter((session) => session.factors?.user?.id === context.userId));
    if (
      (page.sessions || []).length < 100 ||
      (page.details?.totalResult !== undefined && offset + 100 >= Number(page.details.totalResult))
    )
      return result;
  }
  throw new AccountAccessError("unavailable");
}
function describeAccountDevice(description?: string): string {
  if (!description) return "";
  const parts = description
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean);
  if (parts.length < 2) return description;
  const browser = parts[0].replace(/^Mobile Safari$/, "Safari");
  const system = parts.find((part) => /^(Mac OS|macOS|Windows|iOS|Android|Linux|Ubuntu|Chrome OS)$/i.test(part));
  return system ? `${browser} · ${system === "Mac OS" ? "macOS" : system}` : browser;
}
export async function listAccountDevices(context: AccountContext): Promise<AccountDevice[]> {
  const sessions = await listAccountSessions(context);
  return sessions
    .filter(
      (session) =>
        (!session.expirationDate || Date.parse(session.expirationDate) > Date.now()) &&
        (session.factors?.password || session.factors?.intent || session.factors?.webAuthN?.userVerified),
    )
    .map((session) => ({
      id: session.id,
      name: describeAccountDevice(session.userAgent?.description),
      createdAt: session.creationDate,
      updatedAt: session.changeDate,
      current: session.id === context.cookie.id,
    }))
    .sort((left, right) => Date.parse(right.updatedAt || "0") - Date.parse(left.updatedAt || "0"));
}
export async function updateAccountSession(context: AccountContext, body: Record<string, unknown>) {
  // A user token demands MFA right after a second factor is added, so the
  // signed session is updated through the login client instead.
  const remaining = context.session.expirationDate
    ? Math.max(1, Math.floor((timestampDate(context.session.expirationDate).getTime() - Date.now()) / 1000))
    : 30 * 24 * 60 * 60;
  const result = await setSession({
    serviceConfig: context.serviceConfig,
    sessionId: context.cookie.id,
    sessionToken: context.cookie.token,
    checks: body.checks ? fromJson(ChecksSchema, body.checks as JsonObject) : undefined,
    challenges: body.challenges ? fromJson(RequestChallengesSchema, body.challenges as JsonObject) : undefined,
    lifetime: { $typeName: "google.protobuf.Duration", seconds: BigInt(remaining), nanos: 0 },
  });
  const cookie = { ...context.cookie, token: result.sessionToken, changeTs: String(Date.now()) };
  await updateSessionCookie({ id: cookie.id, session: cookie });
  context.cookie = cookie;
  return result;
}
