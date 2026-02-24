import { useEffect, useMemo, useState } from "react";

type UseListPaginationOptions<T> = {
  items: readonly T[];
  initialPageSize?: number;
  minItemsToShowControls?: number;
  resetDeps?: readonly unknown[];
};

type PageRange = {
  start: number;
  end: number;
};

export function useListPagination<T>({
  items,
  initialPageSize = 50,
  minItemsToShowControls = 25,
  resetDeps = [],
}: UseListPaginationOptions<T>) {
  const [pageSize, setPageSize] = useState(initialPageSize);
  const [pageIndex, setPageIndex] = useState(0);

  useEffect(() => {
    setPageIndex(0);
  }, [...resetDeps]);

  const totalItems = items.length;
  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(totalItems / pageSize)),
    [totalItems, pageSize],
  );

  useEffect(() => {
    setPageIndex((prev) => Math.min(prev, totalPages - 1));
  }, [totalPages]);

  const pagedItems = useMemo(() => {
    const start = pageIndex * pageSize;
    return items.slice(start, start + pageSize);
  }, [items, pageIndex, pageSize]);

  const pageRange = useMemo<PageRange>(() => {
    if (totalItems === 0) {
      return { start: 0, end: 0 };
    }
    const start = pageIndex * pageSize + 1;
    const end = Math.min((pageIndex + 1) * pageSize, totalItems);
    return { start, end };
  }, [totalItems, pageIndex, pageSize]);

  return {
    pageSize,
    setPageSize,
    pageIndex,
    setPageIndex,
    totalItems,
    totalPages,
    pageRange,
    pagedItems,
    showPaginationControls: totalItems >= minItemsToShowControls,
  };
}
