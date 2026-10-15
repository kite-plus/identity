"use client";
import { Fingerprint, KeyRound } from "lucide-react";
import { useLocale } from "next-intl";
export enum AuthenticationMethod {
  Passkey = "passkey",
  Password = "password",
}
export const methods = [AuthenticationMethod.Passkey, AuthenticationMethod.Password];
export function AuthenticationMethodRadio({
  selected,
  selectionChanged,
}: {
  selected: AuthenticationMethod;
  selectionChanged: (value: AuthenticationMethod) => void;
}) {
  const zh = useLocale() === "zh";
  return (
    <fieldset className="kite-method-choices">
      <legend className="sr-only">{zh ? "登录方式" : "Sign-in method"}</legend>
      {methods.map((method) => {
        const Icon = method === AuthenticationMethod.Passkey ? Fingerprint : KeyRound;
        return (
          <label key={method} className={`kite-method-choice ${selected === method ? "selected" : ""}`}>
            <input
              type="radio"
              name="authentication-method"
              value={method}
              checked={selected === method}
              onChange={() => selectionChanged(method)}
              data-testid={`${method}-radio`}
            />
            <Icon size={20} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
            <span>{method === AuthenticationMethod.Passkey ? (zh ? "通行密钥" : "Passkey") : zh ? "密码" : "Password"}</span>
          </label>
        );
      })}
    </fieldset>
  );
}
