import {
  formatDateTime,
  formatDuration,
  getJobStatusClass,
} from "../utils/formatters";
import type { JobRun } from "../types";
import JobStatusBadge from "./JobStatusBadge";
import HoverCard from "@/components/HoverCard";

const sortSteps = (steps: JobRun[]) =>
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

const isStructuredStep = (step: JobRun) => {
  const label = (step.step || "").trim();
  if (!label) return false;
  return !label.includes(" ");
};

const resolveDuration = (parent: JobRun, now: number) =>
  formatDuration(
    parent.duration_ms && parent.duration_ms > 0
      ? parent.duration_ms
      : parent.completed_at && parent.started_at
        ? Date.parse(parent.completed_at) - Date.parse(parent.started_at)
        : parent.status === "running" && parent.started_at
          ? now - Date.parse(parent.started_at)
          : undefined,
  );

const stepStatusDetails = (step: JobRun) =>
  step.status === "skipped" && step.message
    ? `Skipped: ${step.message}`
    : step.status
      ? step.status[0].toUpperCase() + step.status.slice(1)
      : step.message
        ? `Message: ${step.message}`
        : "";

const isFailureStatus = (status?: string) =>
  status === "failed" || status === "timeout" || status === "canceled";

function JobStepStatusBadge({ step }: { step: JobRun }) {
  const details = stepStatusDetails(step);
  const badge = (
    <span
      className={`badge badge-xs ${getJobStatusClass(step.status || "")} ${
        details ? "cursor-help" : ""
      }`}
      tabIndex={details ? 0 : -1}
    >
      {step.step}
    </span>
  );

  if (!details) return badge;

  return (
    <HoverCard
      trigger={badge}
      className="hover-card-surface rounded-md p-2 text-xs max-w-96"
    >
      {details}
    </HoverCard>
  );
}

type JobRunCardProps = {
  parent: JobRun;
  steps: JobRun[];
  now: number;
  onCancel: (jobId: string) => void;
};

export default function JobRunCard({
  parent,
  steps,
  now,
  onCancel,
}: JobRunCardProps) {
  const visibleSteps = sortSteps(steps).filter(isStructuredStep);
  return (
    <li className="rounded-lg border border-slate-800/70 bg-base-300/50 px-3 py-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-semibold">{parent.kind}</span>
          <JobStatusBadge
            status={parent.status}
            message={
              isFailureStatus(parent.status) ? undefined : parent.message
            }
          />
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
      {visibleSteps.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {visibleSteps.map((step) => (
            <JobStepStatusBadge key={step.id} step={step} />
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
      {isFailureStatus(parent.status) && parent.message && (
        <p className="mt-1 text-red-400">Error: {parent.message}</p>
      )}
    </li>
  );
}
