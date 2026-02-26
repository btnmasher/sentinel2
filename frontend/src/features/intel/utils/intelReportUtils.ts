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
    meta:
      source.meta && typeof source.meta === "object"
        ? (source.meta as Record<string, unknown>)
        : undefined,
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

export type ZKillIntelMeta = {
  source: "zkill_feed";
  zkill: {
    killmail_id: number;
    url: string;
    display_text: string;
    killer_name: string;
    killer_alliance_id: number;
    killer_corporation_id: number;
    victim_name: string;
    victim_alliance_id: number;
    victim_corporation_id: number;
    victim_ship_name: string;
    killer_hostility: string;
    victim_hostility: string;
    involved_attackers: number;
    system_name: string;
  };
};

export const getZKillIntelMeta = (
  report: Pick<IntelReport, "meta">,
): ZKillIntelMeta | null => {
  const source = report.meta;
  if (!source || typeof source !== "object") return null;
  if (source.source !== "zkill_feed") return null;
  const zkill = source.zkill;
  if (!zkill || typeof zkill !== "object") return null;
  const payload = zkill as Record<string, unknown>;
  if (
    typeof payload.url !== "string" ||
    typeof payload.display_text !== "string" ||
    typeof payload.killer_hostility !== "string" ||
    typeof payload.victim_hostility !== "string" ||
    typeof payload.system_name !== "string"
  ) {
    return null;
  }
  return {
    source: "zkill_feed",
    zkill: {
      killmail_id: toNumber(payload.killmail_id),
      url: payload.url,
      display_text: payload.display_text,
      killer_name:
        typeof payload.killer_name === "string" ? payload.killer_name : "",
      killer_alliance_id: toNumber(payload.killer_alliance_id),
      killer_corporation_id: toNumber(payload.killer_corporation_id),
      victim_name:
        typeof payload.victim_name === "string" ? payload.victim_name : "",
      victim_alliance_id: toNumber(payload.victim_alliance_id),
      victim_corporation_id: toNumber(payload.victim_corporation_id),
      victim_ship_name:
        typeof payload.victim_ship_name === "string"
          ? payload.victim_ship_name
          : "",
      killer_hostility: payload.killer_hostility,
      victim_hostility: payload.victim_hostility,
      involved_attackers: toNumber(payload.involved_attackers),
      system_name: payload.system_name,
    },
  };
};
