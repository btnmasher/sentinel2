export type TimerSeverityBucket =
  | "critical"
  | "high"
  | "medium"
  | "low"
  | "unknown";

export function normalizeTimerSeverity(value: string): TimerSeverityBucket {
  const cleaned = value.trim().toLowerCase();
  if (
    cleaned === "critical" ||
    cleaned === "high" ||
    cleaned === "medium" ||
    cleaned === "low"
  ) {
    return cleaned;
  }
  return "unknown";
}

export function timerSeverityRank(value: string): number {
  switch (normalizeTimerSeverity(value)) {
    case "critical":
      return 4;
    case "high":
      return 3;
    case "medium":
      return 2;
    case "low":
      return 1;
    default:
      return 0;
  }
}

export function timerSeverityDotColor(value: string): string {
  switch (normalizeTimerSeverity(value)) {
    case "critical":
      return "#ef4444";
    case "high":
      return "#f59e0b";
    case "medium":
      return "#22c55e";
    case "low":
      return "#0ea5e9";
    default:
      return "#64748b";
  }
}

export function timerSeverityTextToneClass(value: string): string {
  switch (normalizeTimerSeverity(value)) {
    case "critical":
      return "intel-status-text-stale";
    case "high":
      return "intel-status-text-warn";
    case "medium":
      return "intel-status-text-active";
    case "low":
      return "text-sky-300";
    default:
      return "text-slate-400";
  }
}
