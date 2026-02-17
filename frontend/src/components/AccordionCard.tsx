import type { ReactNode } from "react";

type AccordionCardProps = {
  title: ReactNode;
  subtitle?: ReactNode;
  open: boolean;
  onToggle: () => void;
  children: ReactNode;
};

export default function AccordionCard({
  title,
  subtitle,
  open,
  onToggle,
  children,
}: AccordionCardProps) {
  return (
    <details
      open={open}
      className={`collapse collapse-arrow accordion-card bg-base-200/70 border border-slate-800 ${
        open ? "collapse-open flex flex-1 flex-col min-h-0" : "min-h-0"
      }`}
    >
      <summary
        className="collapse-title shrink-0 flex items-center px-4 min-h-0"
        onClick={(event) => {
          event.preventDefault();
          onToggle();
        }}
      >
        <div className="flex flex-col justify-center">
          <h3 className="font-display leading-none">{title}</h3>
          {open && subtitle && (
            <div className="text-xs text-slate-400 mt-1">{subtitle}</div>
          )}
        </div>
      </summary>
      <div className="collapse-content min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        {children}
      </div>
    </details>
  );
}
