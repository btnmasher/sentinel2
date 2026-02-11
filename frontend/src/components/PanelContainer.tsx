import type { ReactNode } from "react";

type PanelContainerProps = {
  title: string;
  subtitle?: string;
  children: ReactNode;
  className?: string;
};

export default function PanelContainer({
  title,
  subtitle,
  children,
  className,
}: PanelContainerProps) {
  return (
    <div className={`space-y-4 ${className ?? ""}`.trim()}>
      <div className="flex flex-col">
        <h2 className="text-lg font-display leading-none">{title}</h2>
        {subtitle && <p className="text-xs text-slate-400 mt-1">{subtitle}</p>}
      </div>
      {children}
    </div>
  );
}
