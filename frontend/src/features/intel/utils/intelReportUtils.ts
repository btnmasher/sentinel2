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

export const normalizeIntelReport = (input: any): IntelReport | null => {
  const reportId = toNumber(input?.report_id ?? input?.id);
  const reportTime = toNumber(input?.report_time ?? input?.time);
  if (!reportId || !reportTime) return null;

  const recordId =
    input?.recordId ??
    input?.record_id ??
    (typeof input?.id === "string" ? input.id : undefined);

  return {
    recordId,
    id: reportId,
    time: reportTime,
    author: input?.author ?? "",
    text: input?.text ?? "",
    systems: decodeArray(input?.systems),
    regions: decodeArray(input?.regions),
  };
};
