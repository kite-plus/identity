import { CircleAlert, Info } from "lucide-react";
import { ReactNode } from "react";
export enum AlertType {
  ALERT,
  INFO,
}
export function Alert({ children, type = AlertType.ALERT }: { children: ReactNode; type?: AlertType }) {
  const Icon = type === AlertType.ALERT ? CircleAlert : Info;
  return (
    <div
      className={type === AlertType.ALERT ? "alert" : "notice kite-notice"}
      role={type === AlertType.ALERT ? "alert" : "status"}
    >
      <Icon size={16} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
      <span>{children}</span>
    </div>
  );
}
