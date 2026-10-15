"use client";
import { resolveLocalizedLegalLink } from "@/lib/legal-links";
import { LegalAndSupportSettings } from "@zitadel/proto/zitadel/settings/v2/legal_settings_pb";
import { CircleHelp } from "lucide-react";
import { useLocale } from "next-intl";
import Link from "next/link";
import { useState } from "react";
import { Checkbox } from "./checkbox";
import { Translated } from "./translated";

type Props = {
  legal: LegalAndSupportSettings;
  onChange: (allAccepted: boolean) => void;
};

type AcceptanceState = {
  tosAccepted: boolean;
  privacyPolicyAccepted: boolean;
};

export function PrivacyPolicyCheckboxes({ legal, onChange }: Props) {
  const locale = useLocale();
  const [acceptanceState, setAcceptanceState] = useState<AcceptanceState>({
    tosAccepted: false,
    privacyPolicyAccepted: false,
  });
  const helpLink = resolveLocalizedLegalLink(legal?.helpLink, locale);
  const tosLink = resolveLocalizedLegalLink(legal?.tosLink, locale);
  const privacyPolicyLink = resolveLocalizedLegalLink(legal?.privacyPolicyLink, locale);

  // Helper function to check if all required checkboxes are accepted
  const checkAllAccepted = (newState: AcceptanceState) => {
    const hasTosLink = !!tosLink;
    const hasPrivacyLink = !!privacyPolicyLink;

    // Check that all required checkboxes are accepted
    return (!hasTosLink || newState.tosAccepted) && (!hasPrivacyLink || newState.privacyPolicyAccepted);
  };

  return (
    <>
      <p className="text-text-light-secondary-500 dark:text-text-dark-secondary-500 mt-4 flex flex-row items-center text-sm">
        <Translated i18nKey="agreeTo" namespace="register" />
        {helpLink && (
          <span>
            <Link href={helpLink} target="_blank" aria-label="Open help in a new tab" data-testid="help-link">
              <CircleHelp size={16} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
            </Link>
          </span>
        )}
      </p>
      {tosLink && (
        <div className="mt-4 flex items-center">
          <Checkbox
            className="mr-4"
            checked={acceptanceState.tosAccepted}
            value={"tos"}
            onChangeVal={(checked: boolean) => {
              const newState = {
                ...acceptanceState,
                tosAccepted: checked,
              };
              setAcceptanceState(newState);
              onChange(checkAllAccepted(newState));
            }}
            data-testid="tos-checkbox"
          />

          <div className="mr-4 w-[28rem]">
            <p className="text-text-light-500 dark:text-text-dark-500 text-sm">
              <Link href={tosLink} className="underline" target="_blank" data-testid="tos-link">
                <Translated i18nKey="termsOfService" namespace="register" />
              </Link>
            </p>
          </div>
        </div>
      )}
      {privacyPolicyLink && (
        <div className="mt-4 flex items-center">
          <Checkbox
            className="mr-4"
            checked={acceptanceState.privacyPolicyAccepted}
            value={"privacypolicy"}
            onChangeVal={(checked: boolean) => {
              const newState = {
                ...acceptanceState,
                privacyPolicyAccepted: checked,
              };
              setAcceptanceState(newState);
              onChange(checkAllAccepted(newState));
            }}
            data-testid="privacy-policy-checkbox"
          />

          <div className="mr-4 w-[28rem]">
            <p className="text-text-light-500 dark:text-text-dark-500 text-sm">
              <Link href={privacyPolicyLink} className="underline" target="_blank" data-testid="privacy-policy-link">
                <Translated i18nKey="privacyPolicy" namespace="register" />
              </Link>
            </p>
          </div>
        </div>
      )}
    </>
  );
}
