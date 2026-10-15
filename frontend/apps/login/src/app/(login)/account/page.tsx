import { KiteAccount } from "@/components/kite-account";
import { getAccountOverview } from "@/lib/server/account";
import { AccountAccessError } from "@/lib/server/account-context";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";
export default async function Page() {
  const data = await getAccountOverview().catch((error) => {
    if (error instanceof AccountAccessError && error.reason === "expired") redirect("/expired");
    if (error instanceof AccountAccessError && error.reason === "login") redirect("/loginname");
    throw error;
  });
  return <KiteAccount data={data} section="profile" />;
}
