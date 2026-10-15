import { DynamicTheme } from "@/components/dynamic-theme";
import { UserPlusIcon } from "@/components/kite-icons";
import { SessionsList } from "@/components/sessions-list";
import { Translated } from "@/components/translated";
import { getAllSessions } from "@/lib/cookies";
import { getServiceConfig } from "@/lib/service-url";
import { getAuthRequest, getBrandingSettings, getDefaultOrg, listSessions, ServiceConfig } from "@/lib/zitadel";
import { create } from "@zitadel/client";
import { Organization } from "@zitadel/proto/zitadel/org/v2/org_pb";
import { Session, SessionSchema } from "@zitadel/proto/zitadel/session/v2/session_pb";
import { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { headers } from "next/headers";
import Link from "next/link";

export async function generateMetadata(props: {
  searchParams: Promise<Record<string | number | symbol, string | undefined>>;
}): Promise<Metadata> {
  const t = await getTranslations("accounts");
  const { requestId } = await props.searchParams;
  return { title: requestId?.startsWith("oidc_") ? t("confirmTitle") : t("title") };
}

async function loadSessions({ serviceConfig, organization }: { serviceConfig: ServiceConfig; organization?: string }) {
  // Deliberately no cleanup: expired cookie entries are still listed (as invalid
  // accounts) so the user can re-authenticate with one click, like Login V1.
  const sessionCookies = await getAllSessions();

  if (!sessionCookies || !sessionCookies.length) {
    console.info("No session cookie found.");
    return [];
  }

  // listSessions is a plain search: ids of terminated or unknown sessions are
  // simply absent from the response, it does not fail. Transport failures are
  // deliberately not caught here, they must surface instead of downgrading
  // live sessions to invalid cards.
  const ids = sessionCookies.map((s) => s.id).filter((id) => !!id) as string[];
  let liveSessions: Session[] = [];
  if (ids.length) {
    const response = await listSessions({ serviceConfig, ids });
    liveSessions = response?.sessions ?? [];
  }

  // For cookie entries whose server-side session no longer exists (e.g. after
  // an RP-initiated logout) synthesize an invalid Session so the account stays
  // selectable and the user can re-authenticate with one click.
  const liveIds = new Set(liveSessions.map((s) => s.id));
  const synthesized: Session[] = sessionCookies
    .filter((c) => !!c.id && !!c.loginName && !liveIds.has(c.id))
    .map((c) =>
      create(SessionSchema, {
        id: c.id,
        factors: {
          user: {
            loginName: c.loginName,
            // No displayName: the cookie does not carry one, and reusing the
            // loginName would render it twice and break the avatar initials.
            organizationId: c.organization ?? "",
          },
        },
      }),
    );

  let sessions = [...liveSessions, ...synthesized];
  if (organization) {
    sessions = sessions.filter((s) => s.factors?.user?.organizationId === organization);
  }

  return sessions;
}

// The host the reader is signing in to, shown so they know where the
// chosen account goes. Only OIDC requests carry one.
async function requestingSite(serviceConfig: ServiceConfig, requestId?: string) {
  if (!requestId?.startsWith("oidc_")) return undefined;
  try {
    const { authRequest } = await getAuthRequest({ serviceConfig, authRequestId: requestId.replace("oidc_", "") });
    return authRequest?.redirectUri ? new URL(authRequest.redirectUri).host : undefined;
  } catch {
    return undefined;
  }
}

export default async function Page(props: { searchParams: Promise<Record<string | number | symbol, string | undefined>> }) {
  const searchParams = await props.searchParams;

  const requestId = searchParams?.requestId;
  const organization = searchParams?.organization;
  const orgDomain = searchParams?.orgDomain;

  const _headers = await headers();
  const { serviceConfig } = getServiceConfig(_headers);

  let defaultOrganization;
  if (!organization) {
    const org: Organization | null = await getDefaultOrg({ serviceConfig });
    if (org) {
      defaultOrganization = org.id;
    }
  }

  let sessions = await loadSessions({ serviceConfig, organization });

  const branding = await getBrandingSettings({ serviceConfig, organization: organization ?? defaultOrganization });

  const params = new URLSearchParams();

  if (requestId) {
    params.append("requestId", requestId);
  }

  if (organization) {
    params.append("organization", organization);
  }

  if (orgDomain) {
    params.append("orgDomain", orgDomain);
  }

  const t = await getTranslations("accounts");
  const site = await requestingSite(serviceConfig, requestId);

  return (
    <DynamicTheme branding={branding}>
      <div className="flex flex-col space-y-4">
        <h1>{site ? t("confirmTitle") : t("title")}</h1>
        <p className="ztdl-p">
          {site
            ? t.rich("confirmDescription", { site: () => <strong className="request-site">{site}</strong> })
            : t("description")}
        </p>
      </div>

      <div className="w-full">
        <SessionsList sessions={sessions} requestId={requestId} />
        <Link className="account-another" href={`/loginname?` + params}>
          <UserPlusIcon className="icon" aria-hidden="true" />
          <Translated i18nKey="addAnother" namespace="accounts" />
        </Link>
      </div>
    </DynamicTheme>
  );
}
