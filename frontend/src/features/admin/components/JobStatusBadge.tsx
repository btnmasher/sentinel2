import { getJobStatusClass } from "../utils/formatters";
import HoverCard from "@/components/HoverCard";

const statusLabel = (status?: string) => {
  switch (status) {
    case "success":
      return "Success";
    case "failed":
      return "Failed";
    case "running":
      return "Running";
    case "partial":
      return "Partial";
    case "skipped":
      return "Skipped";
    case "canceled":
      return "Canceled";
    case "timeout":
      return "Timeout";
    default:
      return status ? status : "Unknown";
  }
};

type JobStatusBadgeProps = {
  status?: string;
  message?: string;
  className?: string;
};

export default function JobStatusBadge({
  status,
  message,
  className,
}: JobStatusBadgeProps) {
  if (!status) return null;
  const label = statusLabel(status);
  const details = message ? `${label}: ${message}` : label;

  return (
    <HoverCard
      trigger={
        <span
          className={`badge badge-xs cursor-help ${getJobStatusClass(status)} ${
            className ?? ""
          }`.trim()}
          tabIndex={0}
        >
          {label}
        </span>
      }
      className="hover-card-surface rounded-md p-2 text-xs max-w-96"
    >
      {details}
    </HoverCard>
  );
}
