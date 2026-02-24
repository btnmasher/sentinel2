type PaginationBaseProps = {
  pageLabel: string;
  onPrev: () => void;
  onNext: () => void;
  prevDisabled?: boolean;
  nextDisabled?: boolean;
  size?: "xs" | "sm";
  className?: string;
};

export default function PaginationBase({
  pageLabel,
  onPrev,
  onNext,
  prevDisabled = false,
  nextDisabled = false,
  size = "xs",
  className = "",
}: PaginationBaseProps) {
  const buttonClass =
    size === "sm" ? "btn btn-sm btn-outline" : "btn btn-xs btn-outline";
  const textClass = size === "sm" ? "text-sm" : "text-xs";

  return (
    <div className={`flex items-center gap-2 ${textClass} ${className}`.trim()}>
      <button className={buttonClass} disabled={prevDisabled} onClick={onPrev}>
        Prev
      </button>
      <span className="inline-flex min-w-[8ch] justify-center text-slate-400 tabular-nums">
        {pageLabel}
      </span>
      <button className={buttonClass} disabled={nextDisabled} onClick={onNext}>
        Next
      </button>
    </div>
  );
}
