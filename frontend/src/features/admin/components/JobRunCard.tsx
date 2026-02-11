import {
  formatDateTime,
  formatDuration,
  getJobStatusClass,
} from "../utils/formatters";
import type { JobRunEntry } from "../types";
import JobStatusBadge from "./JobStatusBadge";

const sortSteps = (steps: JobRunEntry[]) =>
  steps.slice().sort((a, b) => {
    const aTime = Date.parse(a.started_at || "");
    const bTime = Date.parse(b.started_at || "");
    if (Number.isNaN(aTime) && Number.isNaN(bTime)) {
      return 0;
    }
    if (Number.isNaN(aTime)) return -1;
    if (Number.isNaN(bTime)) return 1;
    return aTime - bTime;
  });

const resolveDuration = (parent: JobRunEntry, now: number) =>
  formatDuration(
    parent.duration_ms && parent.duration_ms > 0
      ? parent.duration_ms
      : parent.completed_at && parent.started_at
        ? Date.parse(parent.completed_at) - Date.parse(parent.started_at)
        : parent.status === "running" && parent.started_at
          ? now - Date.parse(parent.started_at)
          : undefined,
  );

type JobRunCardProps = {
  parent: JobRunEntry;
  steps: JobRunEntry[];
  now: number;
  onCancel: (jobId: string) => void;
};

export default function JobRunCard({
  parent,
  steps,
  now,
  onCancel,
}: JobRunCardProps) {
  return (
    <li className="rounded-lg border border-slate-800/70 bg-base-300/50 px-3 py-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-semibold">{parent.kind}</span>
          {parent.step && <span className="text-slate-400">{parent.step}</span>}
          <JobStatusBadge status={parent.status} error={parent.error} />
        </div>
        <span className="text-slate-400">{resolveDuration(parent, now)}</span>
      </div>
      <div className="mt-1 text-slate-400">
        <span>{parent.trigger || "manual"}</span>
        {(parent.actor_display_name || parent.actor_id) && (
          <>
            {" · by "}
            <span>{parent.actor_display_name || parent.actor_id}</span>
          </>
        )}
        {" · "}
        {formatDateTime(parent.started_at)}
      </div>
      {parent.job_id && (
        <div className="text-slate-500">Job {parent.job_id}</div>
      )}
      {steps.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {sortSteps(steps).map((step) => (
            <span
              key={step.id}
              className={`badge badge-xs ${getJobStatusClass(step.status || "")}`}
              title={
                step.status === "skipped" && step.error
                  ? `Skipped: ${step.error}`
                  : step.status
                    ? step.status[0].toUpperCase() + step.status.slice(1)
                    : step.error
                      ? `Error: ${step.error}`
                      : undefined
              }
            >
              {step.step}
            </span>
          ))}
        </div>
      )}
      {parent.status === "running" && parent.job_id && (
        <div className="mt-2">
          <button
            className="btn btn-xs btn-danger"
            onClick={() => onCancel(parent.job_id || "")}
          >
            Cancel
          </button>
        </div>
      )}
      {parent.error && parent.status !== "skipped" && (
        <p className="mt-1 text-red-400">Error: {parent.error}</p>
      )}
    </li>
  );
}
