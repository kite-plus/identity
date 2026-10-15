"use client";

import { Check, Eye, EyeOff } from "lucide-react";
import { useLocale } from "next-intl";
import { ChangeEvent, DetailedHTMLProps, forwardRef, InputHTMLAttributes, ReactNode, useId, useState } from "react";

export type TextInputProps = DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement> & {
  label: string;
  suffix?: string;
  error?: string | ReactNode;
  success?: string | ReactNode;
  onChange?: (value: ChangeEvent<HTMLInputElement>) => void;
  onBlur?: (value: ChangeEvent<HTMLInputElement>) => void;
  roundness?: string;
};

export const TextInput = forwardRef<HTMLInputElement, TextInputProps>(
  (
    { label, suffix, error, success, type = "text", id: suppliedId, className = "", roundness: _roundness, ...props },
    ref,
  ) => {
    const generatedId = useId();
    const id = suppliedId || generatedId;
    const locale = useLocale();
    const [shown, setShown] = useState(false);
    return (
      <div className="field">
        <div className="field-label">
          <label htmlFor={id}>{label}</label>
        </div>
        <div className={type === "password" ? "password-field" : "kite-input-wrap"}>
          <input
            ref={ref}
            {...props}
            id={id}
            type={type === "password" && shown ? "text" : type}
            className={`input ${className}`}
            aria-invalid={!!error}
            aria-describedby={error || success ? `${id}-feedback` : props["aria-describedby"]}
          />
          {type === "password" && (
            <button
              type="button"
              className="icon-button"
              aria-controls={id}
              aria-pressed={shown}
              aria-label={locale === "zh" ? (shown ? "隐藏密码" : "显示密码") : shown ? "Hide password" : "Show password"}
              onClick={() => setShown(!shown)}
            >
              {shown ? (
                <EyeOff size={18} strokeWidth={1.6} absoluteStrokeWidth />
              ) : (
                <Eye size={18} strokeWidth={1.6} absoluteStrokeWidth />
              )}
            </button>
          )}
          {suffix && <span className="kite-input-suffix">@{suffix}</span>}
        </div>
        {(error || success) && (
          <p id={`${id}-feedback`} className={error ? "field-error" : "field-hint"}>
            {success && <Check size={14} strokeWidth={1.6} />} {error || success}
          </p>
        )}
      </div>
    );
  },
);
TextInput.displayName = "TextInput";
