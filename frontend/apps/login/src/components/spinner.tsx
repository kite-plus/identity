import { LoaderCircle } from "lucide-react";
export function Spinner({ className = "" }: { className?: string }) {
  return (
    <LoaderCircle
      role="status"
      aria-label="Loading"
      size={16}
      strokeWidth={1.6}
      absoluteStrokeWidth
      className={`spinner ${className}`}
    />
  );
}
