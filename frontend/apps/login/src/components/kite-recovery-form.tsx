"use client";
import { resetPassword } from "@/lib/server/password";
import { useLocale } from "next-intl";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Alert } from "./alert";
import { Button } from "./button";
import { TextInput } from "./input";
import { Spinner } from "./spinner";
export function KiteRecoveryForm({
  initialLoginName,
  organization,
  requestId,
}: {
  initialLoginName?: string;
  organization?: string;
  requestId?: string;
}) {
  const zh = useLocale() === "zh";
  const router = useRouter();
  const [name, setName] = useState(initialLoginName || "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <form
      className="form-stack"
      onSubmit={async (event) => {
        event.preventDefault();
        if (busy || !name.trim()) return;
        setBusy(true);
        setError("");
        try {
          const response = await resetPassword({ loginName: name.trim(), organization, requestId });
          if (response && "error" in response) {
            setError(response.error || (zh ? "发送未完成。" : "Unable to send the email."));
            return;
          }
          const params = new URLSearchParams({ loginName: name.trim() });
          if (organization) params.set("organization", organization);
          if (requestId) params.set("requestId", requestId);
          router.push(`/password/reset-sent?${params}`);
        } catch {
          setError(zh ? "发送未完成，请稍后重试。" : "Unable to send the email. Try again later.");
        } finally {
          setBusy(false);
        }
      }}
    >
      <TextInput
        label={zh ? "邮箱或用户名" : "Email or username"}
        value={name}
        onChange={(event) => setName(event.target.value)}
        required
        disabled={busy}
        autoComplete="username"
        autoFocus
        maxLength={200}
      />
      {error && <Alert>{error}</Alert>}
      <Button type="submit" disabled={busy || !name.trim()}>
        {busy && <Spinner />}
        {zh ? "发送重置邮件" : "Send reset email"}
      </Button>
    </form>
  );
}
