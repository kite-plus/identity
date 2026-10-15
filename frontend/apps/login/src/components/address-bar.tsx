"use client";

import { LockKeyhole } from "lucide-react";
import { usePathname } from "next/navigation";
import { Fragment } from "react";

type Props = {
  domain: string;
};

export function AddressBar({ domain }: Props) {
  const pathname = usePathname();

  return (
    <div className="flex items-center space-x-2 overflow-hidden p-3.5 lg:px-5 lg:py-3">
      <div className="text-gray-600">
        <LockKeyhole size={16} strokeWidth={1.6} absoluteStrokeWidth aria-hidden="true" />
      </div>
      <div className="flex space-x-1 text-sm font-medium">
        <div className="max-w-[150px] overflow-hidden px-2 text-ellipsis text-gray-500">
          <span className="whitespace-nowrap">{domain}</span>
        </div>
        {pathname ? (
          <>
            <span className="text-gray-600">/</span>
            {pathname
              .split("/")
              .slice(1)
              .filter((s) => !!s)
              .map((segment) => {
                return (
                  <Fragment key={segment}>
                    <span>
                      <span
                        key={segment}
                        className="animate-[highlight_1s_ease-in-out_1] rounded-full px-1.5 py-0.5 text-gray-800 dark:text-gray-100"
                      >
                        {segment}
                      </span>
                    </span>

                    <span className="text-gray-600">/</span>
                  </Fragment>
                );
              })}
          </>
        ) : null}
      </div>
    </div>
  );
}
