import type {
  ContextTone,
  ReplacementTone,
  StructureTone,
} from "../config/timerOptions";

export function toneButtonClass(tone: StructureTone, active: boolean): string {
  return timerToneButtonClass(tone, active);
}

export function contextToneClass(tone: ContextTone, active: boolean): string {
  return timerToneButtonClass(tone, active);
}

export function replacementToneClass(
  tone: ReplacementTone,
  active: boolean,
): string {
  return timerToneButtonClass(tone, active);
}

export function severityToneClass(
  tone: StructureTone,
  active: boolean,
): string {
  return timerToneButtonClass(tone, active);
}

function timerToneButtonClass(tone: string, active: boolean): string {
  return active
    ? `timer-tone-btn-${tone}-active`
    : `btn-outline timer-tone-btn-${tone}-idle`;
}
