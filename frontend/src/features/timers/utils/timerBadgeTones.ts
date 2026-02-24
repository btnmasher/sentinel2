import {
  hostilityByValue,
  replacementByValue,
  severityByValue,
  structureByValue,
} from "../config/timerOptions";
import { normalizeTimerStanding } from "./timerStanding";
import { normalizeTimerSeverity } from "./timerSeverity";
import { timerStageTone } from "./timerStage";

export type TimerBadgeTone =
  | "blue"
  | "yellow"
  | "green"
  | "purple"
  | "gray"
  | "red"
  | "orange"
  | "lightblue";

export function badgeToneClass(tone: TimerBadgeTone): string {
  return `timer-badge-tone-${tone}`;
}

export function standingBadgeClass(standing: string): string {
  const value = normalizeTimerStanding(standing);
  return badgeToneClass(toBadgeTone(hostilityByValue.get(value)?.tone));
}

export function severityBadgeClass(severity: string): string {
  const value = normalizeTimerSeverity(severity);
  return badgeToneClass(toBadgeTone(severityByValue.get(value as never)?.tone));
}

export function stageBadgeClass(stageLabel?: string): string {
  return badgeToneClass(toBadgeTone(timerStageTone(stageLabel)));
}

export function structureBadgeClassByType(structureType?: string): string {
  const tone = structureByValue.get((structureType ?? "custom") as never)?.tone;
  if (!tone) return badgeToneClass("gray");
  return badgeToneClass(tone);
}

export function structureBadgeClassByTone(tone: string): string {
  if (tone === "lightblue") return badgeToneClass("lightblue");
  if (
    tone === "blue" ||
    tone === "yellow" ||
    tone === "green" ||
    tone === "purple" ||
    tone === "gray" ||
    tone === "red"
  ) {
    return badgeToneClass(tone);
  }
  return badgeToneClass("gray");
}

export function replacementBadgeClass(value: string): string {
  const tone = replacementByValue.get(value as never)?.tone;
  if (!tone) return "badge-ghost";
  return badgeToneClass(toBadgeTone(tone));
}

function toBadgeTone(
  tone: string | undefined,
  fallback: TimerBadgeTone = "gray",
): TimerBadgeTone {
  switch (tone) {
    case "lightblue":
      return "lightblue";
    case "blue":
    case "yellow":
    case "green":
    case "purple":
    case "gray":
    case "red":
    case "orange":
      return tone;
    default:
      return fallback;
  }
}
