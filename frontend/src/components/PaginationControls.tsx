type PaginationControlsProps = {
  page: number;
  hasMore: boolean;
  loading?: boolean;
  onPrev: () => void;
  onNext: () => void;
  size?: "xs" | "sm";
  className?: string;
};

export default function PaginationControls({
  page,
  hasMore,
  loading = false,
  onPrev,
  onNext,
  size = "xs",
  className = "",
}: PaginationControlsProps) {
  const buttonClass =
    size === "sm" ? "btn btn-sm btn-outline" : "btn btn-xs btn-outline";
  const textClass = size === "sm" ? "text-sm" : "text-xs";

  return (
    <div className={`flex items-center gap-2 ${textClass} ${className}`.trim()}>
      <button
        className={buttonClass}
        disabled={page <= 1 || loading}
        onClick={onPrev}
      >
        Prev
      </button>
      <span className="inline-flex min-w-[8ch] justify-center text-slate-400 tabular-nums">
        Page {page}
      </span>
      <button
        className={buttonClass}
        disabled={!hasMore || loading}
        onClick={onNext}
      >
        Next
      </button>
    </div>
  );
}
