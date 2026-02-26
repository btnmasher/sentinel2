import type {
  ContextTone,
  ReplacementTone,
  StructureTone,
} from "../config/timerOptions";
import { toneButtonClass as sharedToneButtonClass } from "@/features/shared";

export function toneButtonClass(tone: StructureTone, active: boolean): string {
  return sharedToneButtonClass(tone, active);
}

export function contextToneClass(tone: ContextTone, active: boolean): string {
  return sharedToneButtonClass(tone, active);
}

export function replacementToneClass(
  tone: ReplacementTone,
  active: boolean,
): string {
  return sharedToneButtonClass(tone, active);
}

export function severityToneClass(
  tone: StructureTone,
  active: boolean,
): string {
  return sharedToneButtonClass(tone, active);
}
