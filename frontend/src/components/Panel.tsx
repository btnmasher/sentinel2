import type { ReactNode } from "react";

type PanelProps = {
  title: string;
  hint?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
};

export default function Panel({
  title,
  hint,
  actions,
  children,
  className,
  bodyClassName,
}: PanelProps) {
  return (
    <section
      className={`card border border-slate-800 bg-base-200/70 ${className ?? ""}`.trim()}
    >
      <div className={`card-body space-y-4 ${bodyClassName ?? ""}`.trim()}>
        <header className="space-y-1">
          <div className="flex items-center justify-between gap-3">
            <h3 className="font-display text-lg">{title}</h3>
            {actions}
          </div>
          {hint ? <p className="text-xs text-slate-400">{hint}</p> : null}
        </header>
        {children}
      </div>
    </section>
  );
}
