import { structureByValue } from "../config/timerOptions";

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
  const value = standing.trim().toLowerCase();
  if (value === "ours") return badgeToneClass("blue");
  if (value === "friendly") return badgeToneClass("green");
  if (value === "complicated") return badgeToneClass("yellow");
  if (value === "hostile") return badgeToneClass("red");
  return badgeToneClass("gray");
}

export function severityBadgeClass(severity: string): string {
  const value = severity.trim().toLowerCase();
  if (value === "critical") return badgeToneClass("red");
  if (value === "high") return badgeToneClass("yellow");
  if (value === "medium") return badgeToneClass("green");
  if (value === "low") return badgeToneClass("blue");
  return badgeToneClass("gray");
}

export function stageBadgeClass(stageLabel?: string): string {
  const value = (stageLabel ?? "").trim().toLowerCase();
  if (value === "armor") return badgeToneClass("yellow");
  if (value === "structure") return badgeToneClass("red");
  if (value === "initial_vulnerability") return badgeToneClass("purple");
  if (value === "anchoring") return badgeToneClass("blue");
  if (value === "unanchoring") return badgeToneClass("orange");
  if (value === "extraction_window" || value === "pickup_window") {
    return badgeToneClass("green");
  }
  return badgeToneClass("gray");
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
  if (value === "logi_replacement") return badgeToneClass("blue");
  if (value === "corp_replacement") return badgeToneClass("green");
  if (value === "alliance_replacement") return badgeToneClass("purple");
  return "badge-ghost";
}
