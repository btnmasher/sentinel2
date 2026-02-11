type JobRunsEmptyStateProps = {
  hasFilters: boolean;
};

export default function JobRunsEmptyState({
  hasFilters,
}: JobRunsEmptyStateProps) {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center text-center">
      <div className="space-y-2">
        <p className="text-base font-semibold text-slate-200">No jobs found</p>
        {hasFilters && (
          <p className="text-xs text-slate-400">
            Clear filters to see more job runs.
          </p>
        )}
      </div>
    </div>
  );
}
