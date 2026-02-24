import { useEffect, useMemo, useState } from "react";
import { useShallow } from "zustand/shallow";
import useConfirm from "@/app/hooks/useConfirm";
import CursorPagination from "@/components/CursorPagination";
import Panel from "@/components/Panel";
import ShadowedScrollArea from "@/components/ShadowedScrollArea";
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
    includeHidden,
    jobKindExclusions,
    setJobKinds,
    setIncludeHidden,
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
      includeHidden: s.includeHidden,
      jobKindExclusions: s.jobKindExclusions,
      setJobKinds: s.setJobKinds,
      setIncludeHidden: s.setIncludeHidden,
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
      { id: "sov_campaign_sync", label: "Sovereignty Campaign Sync" },
      { id: "skyhook_sync", label: "Structure Notifications Sync" },
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
    includeHidden ||
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
    <Panel
      title="Job Runs"
      hint="Latest cron and admin jobs."
      className="h-full min-h-0"
      bodyClassName="h-full min-h-0 flex flex-col gap-1.5"
    >
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
        <label className="inline-flex items-center gap-1.5 rounded-md border border-slate-700/70 px-2 py-1 text-xs">
          <input
            type="checkbox"
            className="toggle toggle-xs toggle-primary"
            checked={includeHidden}
            onChange={(event) => {
              setIncludeHidden(event.target.checked);
              void loadJobs(1, { silent: false });
            }}
          />
          Show hidden jobs
        </label>
      </div>
      {jobLoading && <p className="text-xs text-slate-400">Loading jobs…</p>}
      <div className="flex-1 min-h-0 overflow-hidden">
        {!jobLoading && jobRuns.length === 0 ? (
          <JobRunsEmptyState hasFilters={hasFilters} />
        ) : (
          jobRuns.length > 0 && (
            <ShadowedScrollArea>
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
            </ShadowedScrollArea>
          )
        )}
      </div>
      {(jobPage > 1 || jobHasMore) && (
        <CursorPagination
          page={jobPage}
          hasMore={jobHasMore}
          loading={jobLoading}
          onPrev={() =>
            void loadJobs(Math.max(1, jobPage - 1), { silent: false })
          }
          onNext={() => void loadJobs(jobPage + 1, { silent: false })}
        />
      )}
    </Panel>
  );
}
