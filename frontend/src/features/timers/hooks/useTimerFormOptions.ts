import { useMemo } from "react";
import {
  contextOptionsFor,
  hostilityOptions,
  replacementOptions,
  severityOptions,
  structureGroups,
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
