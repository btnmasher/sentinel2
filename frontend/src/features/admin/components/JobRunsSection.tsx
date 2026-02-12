import { useEffect, useMemo, useState } from "react";
import { useShallow } from "zustand/shallow";
import useConfirm from "@/app/hooks/useConfirm";
import PaginationControls from "@/components/PaginationControls";
import { useAdminJobRunsStore } from "../store/adminJobRunsStore";
import type { DateRange } from "react-day-picker";
import DateRangePicker from "@/components/DateRangePicker";
import SelectionDropdown from "@/components/SelectionDropdown";
import JobRunsList from "./JobRunsList";
import JobRunsEmptyState from "./JobRunsEmptyState";

const formatDate = (value?: Date) => {
  if (!value) return undefined;
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

export default function JobRunsSection() {
  const requestConfirm = useConfirm();
  const {
    jobRuns,
    jobLoading,
    jobPage,
    jobHasMore,
    loadJobs,
    setDateRange,
    startDate,
    endDate,
    jobKindExclusions,
    setJobKinds,
    subscribe,
    cancelJob,
  } = useAdminJobRunsStore(
    useShallow((s) => ({
      jobRuns: s.jobRuns,
      jobLoading: s.loading,
      jobPage: s.page,
      jobHasMore: s.hasMore,
      loadJobs: s.loadJobs,
      setDateRange: s.setDateRange,
      startDate: s.startDate,
      endDate: s.endDate,
      jobKindExclusions: s.jobKindExclusions,
      setJobKinds: s.setJobKinds,
      subscribe: s.subscribe,
      cancelJob: s.cancelJob,
    })),
  );
  const [now, setNow] = useState(Date.now());
  const rangeValue = useMemo<DateRange | undefined>(() => {
    if (!startDate && !endDate) return undefined;
    return {
      from: startDate ? new Date(`${startDate}T00:00:00`) : undefined,
      to: endDate ? new Date(`${endDate}T00:00:00`) : undefined,
    };
  }, [startDate, endDate]);
  const { startHour, endHour } = useAdminJobRunsStore(
    useShallow((s) => ({
      startHour: s.startHour,
      endHour: s.endHour,
    })),
  );
  const jobKindOptions = useMemo(
    () => [
      { id: "map_data_update", label: "Map Data Update" },
      { id: "map_data_step", label: "Map Data Step" },
      { id: "character_refresh", label: "Character Refresh" },
      { id: "cleanup", label: "Cleanup" },
    ],
    [],
  );
  const selectedJobKinds = useMemo(() => {
    if (jobKindOptions.length === 0) return [];
    const excluded = new Set(jobKindExclusions);
    return jobKindOptions
      .map((option) => option.id)
      .filter((id) => !excluded.has(id));
  }, [jobKindExclusions, jobKindOptions]);
  const hasFilters = Boolean(
    startDate ||
    endDate ||
    startHour !== undefined ||
    endHour !== undefined ||
    jobKindExclusions.length > 0,
  );

  useEffect(() => {
    void loadJobs(1);
    let cleanup: (() => void) | null = null;
    void subscribe().then((unsubscribe) => {
      cleanup = () => void unsubscribe();
    });
    return () => {
      cleanup?.();
    };
  }, [loadJobs, subscribe]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);
    return () => {
      window.clearInterval(interval);
    };
  }, []);

  return (
    <section className="card bg-base-200/70 border border-slate-800 h-full min-h-0">
      <div className="card-body h-full min-h-0 grid grid-rows-[auto_auto_auto_1fr_auto] gap-4">
        <div>
          <h2 className="font-display text-2xl">Job Runs</h2>
          <p className="text-xs text-slate-400">Latest cron and admin jobs.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
          <DateRangePicker
            value={rangeValue}
            startHour={startHour}
            endHour={endHour}
            showTimeSelect
            onClear={() => {
              setDateRange({
                startDate: undefined,
                endDate: undefined,
                startHour: undefined,
                endHour: undefined,
              });
              void loadJobs(1, { silent: false });
            }}
            onApply={(range, nextStartHour, nextEndHour) => {
              const nextStart = formatDate(range?.from);
              const nextEnd = formatDate(range?.to);
              setDateRange({
                startDate: nextStart,
                endDate: nextEnd,
                startHour: nextStartHour,
                endHour: nextEndHour,
              });
              void loadJobs(1, { silent: false });
            }}
          />
          <SelectionDropdown
            items={jobKindOptions}
            selected={selectedJobKinds}
            onChange={(next) => {
              const excluded = jobKindOptions
                .map((option) => option.id)
                .filter((id) => !next.includes(id));
              setJobKinds(excluded);
              void loadJobs(1, { silent: false });
            }}
            multi
            showTags
            coalesceAllTags
            label="Job types"
            buttonClassName="min-w-[160px]"
          />
        </div>
        <div>
          {jobLoading && (
            <p className="text-xs text-slate-400">Loading jobs…</p>
          )}
        </div>
        {!jobLoading && jobRuns.length === 0 ? (
          <JobRunsEmptyState hasFilters={hasFilters} />
        ) : (
          jobRuns.length > 0 && (
            <JobRunsList
              jobRuns={jobRuns}
              now={now}
              onCancel={(jobId) =>
                requestConfirm({
                  title: "Cancel Job",
                  body: `Cancel job ${jobId}?`,
                  onConfirm: () => void cancelJob(jobId),
                  confirmLabel: "Cancel job",
                  cancelLabel: "Keep running",
                  tone: "danger",
                })
              }
            />
          )
        )}
        <div>
          {(jobPage > 1 || jobHasMore) && (
            <PaginationControls
              page={jobPage}
              hasMore={jobHasMore}
              loading={jobLoading}
              onPrev={() =>
                void loadJobs(Math.max(1, jobPage - 1), { silent: false })
              }
              onNext={() => void loadJobs(jobPage + 1, { silent: false })}
            />
          )}
        </div>
      </div>
    </section>
  );
}
