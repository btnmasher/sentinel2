import type { SearchResult } from "../types";

export const formatDateTime = (value?: string) => {
  if (!value) return "—";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString();
};

export const formatSessionStatus = (value: string) => {
  if (!value) return "Active";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  if (parsed.getFullYear() <= 1971) return "Active";
  if (parsed.getTime() > Date.now()) {
    return `Active (revokes at ${parsed.toLocaleString()})`;
  }
  return `Revoked ${parsed.toLocaleString()}`;
};

export const formatDuration = (value?: number) => {
  if (value === undefined || value === null) return "—";
  if (value < 1000) return `${value} ms`;
  return `${(value / 1000).toFixed(1)} s`;
};

export const buildSearchLabel = (result: SearchResult) =>
  result.main_name ? `${result.name} (${result.main_name})` : result.name;

export const hasSearchMain = (result?: SearchResult | null) =>
  Boolean(result?.main_name || result?.is_main);

export const getJobStatusClass = (status?: string) => {
  switch (status) {
    case "success":
      return "badge-success";
    case "failed":
      return "badge-error";
    case "running":
      return "badge-info";
    case "partial":
      return "badge-warning";
    case "skipped":
      return "badge-ghost";
    case "canceled":
      return "badge-ghost";
    default:
      return "badge-ghost";
  }
};
