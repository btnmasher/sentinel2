import type { JobRunGroup } from "../types";
import JobRunCard from "./JobRunCard";

type JobRunsListProps = {
  jobRuns: JobRunGroup[];
  now: number;
  onCancel: (jobId: string) => void;
};

export default function JobRunsList({ jobRuns, now, onCancel }: JobRunsListProps) {
  return (
    <ul className="space-y-2 text-xs min-h-0 overflow-auto">
      {jobRuns.map(({ parent, steps }) => (
        <JobRunCard
          key={parent.job_id || parent.id}
          parent={parent}
          steps={steps}
          now={now}
          onCancel={onCancel}
        />
      ))}
    </ul>
  );
}
