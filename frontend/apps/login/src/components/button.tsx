import { ThemeableProps } from "@/lib/themeUtils";
import { clsx } from "clsx";
import { ButtonHTMLAttributes, DetailedHTMLProps, forwardRef } from "react";

export enum ButtonSizes {
  Small = "Small",
  Large = "Large",
}
export enum ButtonVariants {
  Primary = "Primary",
  Secondary = "Secondary",
  Destructive = "Destructive",
}
export enum ButtonColors {
  Neutral = "Neutral",
  Primary = "Primary",
  Warn = "Warn",
}
export type ButtonProps = DetailedHTMLProps<ButtonHTMLAttributes<HTMLButtonElement>, HTMLButtonElement> & {
  size?: ButtonSizes;
  variant?: ButtonVariants;
  color?: ButtonColors;
} & ThemeableProps;

export const getButtonClasses = (
  size: ButtonSizes,
  variant: ButtonVariants,
  color: ButtonColors,
  _roundness = "",
  _appearance = "",
) =>
  clsx("button", {
    primary: variant === ButtonVariants.Primary && color !== ButtonColors.Warn,
    outline: variant === ButtonVariants.Secondary && color !== ButtonColors.Warn,
    danger: variant === ButtonVariants.Destructive || (variant === ButtonVariants.Primary && color === ButtonColors.Warn),
    "danger-outline": variant === ButtonVariants.Secondary && color === ButtonColors.Warn,
  });

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      children,
      className = "",
      variant = ButtonVariants.Primary,
      size = ButtonSizes.Small,
      color = ButtonColors.Primary,
      roundness: _roundness,
      ...props
    },
    ref,
  ) => (
    <button type="button" ref={ref} className={`${getButtonClasses(size, variant, color)} ${className}`} {...props}>
      {children}
    </button>
  ),
);
Button.displayName = "Button";
