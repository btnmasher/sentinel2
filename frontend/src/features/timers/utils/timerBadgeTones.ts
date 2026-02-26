import {
  replacementByValue,
  severityByValue,
  structureByValue,
} from "../config/timerOptions";
import {
  Tone,
  badgeToneClass,
  standingBadgeClass,
  toToneOrDefault,
} from "@/features/shared";
import { normalizeTimerSeverity } from "./timerSeverity";
import { timerStageTone } from "./timerStage";

export { badgeToneClass, standingBadgeClass };

export function severityBadgeClass(severity: string): string {
  const value = normalizeTimerSeverity(severity);
  return badgeToneClass(
    toToneOrDefault(severityByValue.get(value as never)?.tone),
  );
}

export function stageBadgeClass(stageLabel?: string): string {
  return badgeToneClass(toToneOrDefault(timerStageTone(stageLabel)));
}

export function structureBadgeClassByType(structureType?: string): string {
  const tone = structureByValue.get((structureType ?? "custom") as never)?.tone;
  if (!tone) return badgeToneClass(Tone.Gray);
  return badgeToneClass(tone as Tone);
}

export function structureBadgeClassByTone(tone: string): string {
  if (tone === Tone.LightBlue) return badgeToneClass(Tone.LightBlue);
  if (
    tone === Tone.Blue ||
    tone === Tone.Yellow ||
    tone === Tone.Green ||
    tone === Tone.Purple ||
    tone === Tone.Gray ||
    tone === Tone.Red
  ) {
    return badgeToneClass(tone);
  }
  return badgeToneClass(Tone.Gray);
}

export function replacementBadgeClass(value: string): string {
  const tone = replacementByValue.get(value as never)?.tone;
  if (!tone) return "badge-ghost";
  return badgeToneClass(toToneOrDefault(tone));
}
