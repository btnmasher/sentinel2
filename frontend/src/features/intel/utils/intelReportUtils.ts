import type { IntelReport } from "../types";

export const decodeArray = <T>(value: unknown): T[] => {
  if (Array.isArray(value)) return value as T[];
  return [];
};

export const toNumber = (value: unknown): number => {
  if (typeof value === "number") return value;
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isNaN(parsed) ? 0 : parsed;
  }
  return 0;
};

const asRecord = (value: unknown): Record<string, unknown> =>
  value && typeof value === "object" ? (value as Record<string, unknown>) : {};

export const normalizeIntelReport = (input: unknown): IntelReport | null => {
  const source = asRecord(input);
  const reportId = toNumber(source.report_id ?? source.id);
  const reportTime = toNumber(source.report_time ?? source.time);
  if (!reportId || !reportTime) return null;

  const recordId =
    (typeof source.recordId === "string" ? source.recordId : undefined) ??
    (typeof source.record_id === "string" ? source.record_id : undefined) ??
    (typeof source.id === "string" ? source.id : undefined);

  return {
    recordId,
    id: reportId,
    time: reportTime,
    author: typeof source.author === "string" ? source.author : "",
    text: typeof source.text === "string" ? source.text : "",
    channel_id:
      (typeof source.channel_id === "string" ? source.channel_id : undefined) ??
      (typeof source.channel === "string" ? source.channel : undefined),
    systems: decodeArray(source.systems),
    regions: decodeArray(source.regions),
  };
};

const CLEAR_REPORT_PATTERN = /\b(clr|clear)\b/i;

export const isClearIntelReport = (
  report: Pick<IntelReport, "text" | "systems">,
) => {
  return report.systems.length > 0 && CLEAR_REPORT_PATTERN.test(report.text);
};
