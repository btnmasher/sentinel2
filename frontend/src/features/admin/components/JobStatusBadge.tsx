import { getJobStatusClass } from "../utils/formatters";

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
  error?: string;
  className?: string;
};

export default function JobStatusBadge({
  status,
  error,
  className,
}: JobStatusBadgeProps) {
  if (!status) return null;
  const label = statusLabel(status);
  const title =
    status === "skipped" && error
      ? `Skipped: ${error}`
      : error
        ? `${label}: ${error}`
        : label;

  return (
    <span
      className={`badge badge-xs ${getJobStatusClass(status)} ${
        className ?? ""
      }`.trim()}
      title={title}
    >
      {label}
    </span>
  );
}
