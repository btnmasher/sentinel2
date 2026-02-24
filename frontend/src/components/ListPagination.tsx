import PaginationBase from "./PaginationBase";

type ListPaginationProps = {
  totalItems: number;
  pageSize: number;
  pageIndex: number;
  onPageSizeChange: (next: number) => void;
  onPageChange: (next: number) => void;
  minItemsToShow?: number;
  pageSizeOptions?: number[];
  className?: string;
};

export default function ListPagination({
  totalItems,
  pageSize,
  pageIndex,
  onPageSizeChange,
  onPageChange,
  minItemsToShow = 25,
  pageSizeOptions = [25, 50, 100],
  className = "",
}: ListPaginationProps) {
  if (totalItems < minItemsToShow) {
    return null;
  }

  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const start = totalItems === 0 ? 0 : pageIndex * pageSize + 1;
  const end = Math.min((pageIndex + 1) * pageSize, totalItems);

  return (
    <div
      className={`grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 text-xs text-slate-400 ${className}`.trim()}
    >
      <span>
        Showing {start}-{end} of {totalItems}
      </span>
      <div className="grid grid-flow-col auto-cols-max items-center justify-end gap-2">
        <label className="inline-flex items-center gap-1.5 whitespace-nowrap">
          <span>Per page</span>
          <select
            className="select select-xs bg-base-300/70"
            value={pageSize}
            onChange={(event) => {
              const next = Number(event.target.value);
              if (Number.isFinite(next) && next > 0) {
                onPageSizeChange(next);
              }
            }}
          >
            {pageSizeOptions.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </label>
        <PaginationBase
          pageLabel={`Page ${pageIndex + 1}/${totalPages}`}
          onPrev={() => onPageChange(Math.max(0, pageIndex - 1))}
          onNext={() => onPageChange(Math.min(totalPages - 1, pageIndex + 1))}
          prevDisabled={pageIndex === 0}
          nextDisabled={pageIndex >= totalPages - 1}
        />
      </div>
    </div>
  );
}
