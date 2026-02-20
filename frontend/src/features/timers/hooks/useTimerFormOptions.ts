import { useMemo } from "react";
import {
  contextOptionsFor,
  hostilityOptions,
  replacementOptions,
  severityOptions,
  structureGroups,
  type ContextTone,
  type ReplacementTone,
  type StructureTone,
} from "../config/timerOptions";
import type { TimerStructureType } from "../types";

export function useTimerFormOptions(structureType: TimerStructureType | "") {
  const contextOptions = useMemo(
    () => contextOptionsFor(structureType),
    [structureType],
  );
  return {
    contextOptions,
    hostilityOptions,
    replacementOptions,
    severityOptions,
    structureGroups,
  };
}

export function toneButtonClass(tone: StructureTone, active: boolean): string {
  return active
    ? `timer-tone-btn-${tone}-active`
    : `btn-outline timer-tone-btn-${tone}-idle`;
}

export function contextToneClass(tone: ContextTone, active: boolean): string {
  return active
    ? `timer-tone-btn-${tone}-active`
    : `btn-outline timer-tone-btn-${tone}-idle`;
}

export function replacementToneClass(
  tone: ReplacementTone,
  active: boolean,
): string {
  return active
    ? `timer-tone-btn-${tone}-active`
    : `btn-outline timer-tone-btn-${tone}-idle`;
}

export function severityToneClass(
  tone: StructureTone,
  active: boolean,
): string {
  return active
    ? `timer-tone-btn-${tone}-active`
    : `btn-outline timer-tone-btn-${tone}-idle`;
}
