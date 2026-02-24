import PaginationBase from "./PaginationBase";

type CursorPaginationProps = {
  page: number;
  hasMore: boolean;
  loading?: boolean;
  onPrev: () => void;
  onNext: () => void;
  size?: "xs" | "sm";
  className?: string;
};

export default function CursorPagination({
  page,
  hasMore,
  loading = false,
  onPrev,
  onNext,
  size = "xs",
  className = "",
}: CursorPaginationProps) {
  return (
    <PaginationBase
      pageLabel={`Page ${page}`}
      onPrev={onPrev}
      onNext={onNext}
      prevDisabled={page <= 1 || loading}
      nextDisabled={!hasMore || loading}
      size={size}
      className={className}
    />
  );
}
