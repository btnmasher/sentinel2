import type { ReactNode } from "react";

type AccordionCardProps = {
  title: string;
  subtitle?: ReactNode;
  open: boolean;
  onToggle: () => void;
  children: ReactNode;
  className?: string;
  titleClassName?: string;
  subtitleClassName?: string;
  contentClassName?: string;
};

export default function AccordionCard({
  title,
  subtitle,
  open,
  onToggle,
  children,
  className,
  titleClassName,
  subtitleClassName,
  contentClassName,
}: AccordionCardProps) {
  return (
    <details
      open={open}
      className={`collapse collapse-arrow accordion-card bg-base-200/70 border border-slate-800 ${
        open ? "collapse-open" : ""
      } ${className ?? ""}`}
    >
      <summary
        className="collapse-title flex items-center px-4 min-h-0"
        onClick={(event) => {
          event.preventDefault();
          onToggle();
        }}
      >
        <div className="flex flex-col justify-center">
          <h3
            className={`font-display leading-none ${
              titleClassName ?? ""
            }`.trim()}
          >
            {title}
          </h3>
          {open && subtitle && (
            <div
              className={`text-xs text-slate-400 mt-1 ${
                subtitleClassName ?? ""
              }`.trim()}
            >
              {subtitle}
            </div>
          )}
        </div>
      </summary>
      <div className={`collapse-content px-4 pb-4 ${contentClassName ?? ""}`}>
        {children}
      </div>
    </details>
  );
}
