import { format, intervalToDuration, isValid, parseISO, addDays, startOfDay } from "date-fns";
import { structureByValue } from "./config/timerOptions";
import type {
  TimerReplacementAction,
  TimerStageLabel,
  TimerStandingType,
  TimerStructureType,
} from "./types";

export type FormattedDateParts = {
  year: string;
  month: string;
  day: string;
  hour: string;
  minute: string;
  second: string;
  suffix?: string;
  timezone?: string;
};

export type TimerDateParts = {
  local: FormattedDateParts;
  eve: FormattedDateParts;
};

export function formatTicker(ticker: string): string {
  const trimmed = ticker.trim();
  if (!trimmed) return "";
  return `[${trimmed}]`;
}

export function formatStructureType(value: TimerStructureType | string): string {
  const structured = structureByValue.get(value as TimerStructureType);
  if (structured) return structured.label;
  return (
    value
      .split("_")
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(" ") || "Custom"
  );
}

export function formatStanding(value: TimerStandingType): string {
  if (value === "ours") return "Ours";
  if (value === "complicated") return "It's Complicated";
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export function formatStageLabel(value: TimerStageLabel | ""): string {
  if (!value) return "-";
  if (value === "structure") return "Hull";
  return value
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function formatReplacement(value: TimerReplacementAction): string {
  switch (value) {
    case "not_replaceable":
      return "No Replacement";
    case "logi_replacement":
      return "Logi Replacement";
    case "corp_replacement":
      return "Corp Replacement";
    case "alliance_replacement":
      return "Alliance Replacement";
    default:
      return "Replacement Unknown";
  }
}

export function formatDateParts(
  value: Date,
  use24Hour: boolean,
  includeTimezone: boolean,
): FormattedDateParts {
  const formatString = use24Hour
    ? "yyyy-MM-dd-HH-mm-ss"
    : "yyyy-MM-dd-hh-mm-ss-aaa";
  const parts = format(value, formatString).split("-");
  const timezone = includeTimezone
    ? new Intl.DateTimeFormat(undefined, { timeZoneName: "short" })
        .formatToParts(value)
        .find((part) => part.type === "timeZoneName")?.value
    : undefined;

  return {
    year: parts[0] ?? "0000",
    month: parts[1] ?? "00",
    day: parts[2] ?? "00",
    hour: parts[3] ?? "00",
    minute: parts[4] ?? "00",
    second: parts[5] ?? "00",
    suffix: parts[6] ? parts[6].toUpperCase() : undefined,
    timezone: timezone?.toUpperCase(),
  };
}

export function formatTimerDateParts(
  value: string,
  use24Hour: boolean,
): TimerDateParts | null {
  const date = parseISO(value);
  if (!isValid(date)) return null;
  const local = formatDateParts(date, use24Hour, true);
  const eveDisplay = new Date(
    date.getUTCFullYear(),
    date.getUTCMonth(),
    date.getUTCDate(),
    date.getUTCHours(),
    date.getUTCMinutes(),
    date.getUTCSeconds(),
    0,
  );
  const eve = formatDateParts(eveDisplay, use24Hour, false);
  return { local, eve };
}

export function formatCountdown(value: string, nowMs: number): string {
  const targetMs = Date.parse(value);
  if (Number.isNaN(targetMs)) return "--:--:--";
  if (targetMs <= nowMs) return "00:00:00";
  const duration = intervalToDuration({ start: nowMs, end: targetMs });
  const years = duration.years ?? 0;
  const months = duration.months ?? 0;
  const days = duration.days ?? 0;
  const hours = duration.hours ?? 0;
  const minutes = duration.minutes ?? 0;
  const seconds = duration.seconds ?? 0;
  const hh = String(hours).padStart(2, "0");
  const mm = String(minutes).padStart(2, "0");
  const ss = String(seconds).padStart(2, "0");
  const prefix = [
    years > 0 ? `${years}y` : "",
    months > 0 ? `${months}mo` : "",
    days > 0 ? `${days}d` : "",
  ]
    .filter(Boolean)
    .join(" ");
  if (prefix) return `${prefix} ${hh}:${mm}:${ss}`;
  return `${hh}:${mm}:${ss}`;
}

export function countdownTone(value: string, nowMs: number): string {
  const targetMs = Date.parse(value);
  if (Number.isNaN(targetMs)) return "timer-countdown-neutral";
  const remainingMs = targetMs - nowMs;
  if (remainingMs <= 0) return "timer-countdown-critical";
  if (remainingMs <= 15 * 60 * 1000) return "timer-countdown-critical";
  if (remainingMs <= 60 * 60 * 1000) return "timer-countdown-warning";
  return "timer-countdown-healthy";
}

export function isCountdownExpired(value: string, nowMs: number): boolean {
  const targetMs = Date.parse(value);
  if (Number.isNaN(targetMs)) return false;
  return targetMs <= nowMs;
}

export function validISO(value: string): string {
  if (!value) return "";
  const date = parseISO(value);
  if (!isValid(date)) return "";
  return date.toISOString();
}

export function isoToEveDisplayDate(value: string): Date | null {
  const date = parseISO(value);
  if (!isValid(date)) return null;
  return new Date(
    date.getUTCFullYear(),
    date.getUTCMonth(),
    date.getUTCDate(),
    date.getUTCHours(),
    date.getUTCMinutes(),
    date.getUTCSeconds(),
    0,
  );
}

export function eveDisplayDateToISO(value: Date): string {
  if (Number.isNaN(value.getTime())) return "";
  return new Date(
    Date.UTC(
      value.getFullYear(),
      value.getMonth(),
      value.getDate(),
      value.getHours(),
      value.getMinutes(),
      value.getSeconds(),
      0,
    ),
  ).toISOString();
}

export function nextUTCMidnightISO(): string {
  const now = new Date();
  const eveDisplayNow = new Date(
    now.getUTCFullYear(),
    now.getUTCMonth(),
    now.getUTCDate(),
    now.getUTCHours(),
    now.getUTCMinutes(),
    now.getUTCSeconds(),
    0,
  );
  return eveDisplayDateToISO(addDays(startOfDay(eveDisplayNow), 1));
}

export function formatUTCDateTime(value: string): string {
  const date = parseISO(value);
  if (!isValid(date)) return value;
  const yyyy = String(date.getUTCFullYear());
  const mm = String(date.getUTCMonth() + 1).padStart(2, "0");
  const dd = String(date.getUTCDate()).padStart(2, "0");
  const hh = String(date.getUTCHours()).padStart(2, "0");
  const mi = String(date.getUTCMinutes()).padStart(2, "0");
  const ss = String(date.getUTCSeconds()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd} ${hh}:${mi}:${ss}`;
}

export function toUnixSeconds(value: string): number | null {
  const ms = Date.parse(value);
  if (Number.isNaN(ms)) return null;
  return Math.floor(ms / 1000);
}
