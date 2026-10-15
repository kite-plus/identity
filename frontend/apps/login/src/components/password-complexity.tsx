import { Translated } from "@/components/translated";
import { lowerCaseValidator, numberValidator, symbolValidator, upperCaseValidator } from "@/helpers/validators";
import { PasswordComplexitySettings } from "@zitadel/proto/zitadel/settings/v2/password_settings_pb";
import { Check, X } from "lucide-react";
import { useTranslations } from "next-intl";

type Props = {
  passwordComplexitySettings: PasswordComplexitySettings;
  password: string;
  equals: boolean;
};

function CheckIcon({ title }: { title: string }) {
  return <Check size={14} strokeWidth={1.6} absoluteStrokeWidth className="mr-2 text-green-600" aria-label={title} />;
}

function CrossIcon({ title }: { title: string }) {
  return <X size={14} strokeWidth={1.6} absoluteStrokeWidth className="mr-2 text-neutral-400" aria-label={title} />;
}

function renderIcon(matched: boolean, t: ReturnType<typeof useTranslations>) {
  return matched ? <CheckIcon title={t("complexity.matches")} /> : <CrossIcon title={t("complexity.doesNotMatch")} />;
}
const desc = "text-14px leading-4 text-input-light-label dark:text-input-dark-label";

export function PasswordComplexity({ passwordComplexitySettings, password, equals }: Props) {
  const t = useTranslations("password");
  const hasMinLength = password?.length >= passwordComplexitySettings.minLength;
  const hasSymbol = symbolValidator(password);
  const hasNumber = numberValidator(password);
  const hasUppercase = upperCaseValidator(password);
  const hasLowercase = lowerCaseValidator(password);

  return (
    <div className="mb-4 grid grid-cols-2 gap-x-8 gap-y-2">
      {passwordComplexitySettings.minLength != undefined ? (
        <div className="flex flex-row items-center" data-testid="length-check">
          {renderIcon(hasMinLength, t)}
          <span className={desc}>
            <Translated
              i18nKey="complexity.length"
              namespace="password"
              data={{ minLength: passwordComplexitySettings.minLength.toString() }}
            />
          </span>
        </div>
      ) : null}
      {passwordComplexitySettings.requiresSymbol && (
        <div className="flex flex-row items-center" data-testid="symbol-check">
          {renderIcon(hasSymbol, t)}
          <span className={desc}>
            <Translated i18nKey="complexity.hasSymbol" namespace="password" />
          </span>
        </div>
      )}
      {passwordComplexitySettings.requiresNumber && (
        <div className="flex flex-row items-center" data-testid="number-check">
          {renderIcon(hasNumber, t)}
          <span className={desc}>
            <Translated i18nKey="complexity.hasNumber" namespace="password" />
          </span>
        </div>
      )}
      {passwordComplexitySettings.requiresUppercase && (
        <div className="flex flex-row items-center" data-testid="uppercase-check">
          {renderIcon(hasUppercase, t)}
          <span className={desc}>
            <Translated i18nKey="complexity.hasUppercase" namespace="password" />
          </span>
        </div>
      )}
      {passwordComplexitySettings.requiresLowercase && (
        <div className="flex flex-row items-center" data-testid="lowercase-check">
          {renderIcon(hasLowercase, t)}
          <span className={desc}>
            <Translated i18nKey="complexity.hasLowercase" namespace="password" />
          </span>
        </div>
      )}
      <div className="flex flex-row items-center" data-testid="equal-check">
        {renderIcon(equals, t)}
        <span className={desc}>
          <Translated i18nKey="complexity.equals" namespace="password" />
        </span>
      </div>
    </div>
  );
}
