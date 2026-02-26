import {
  isNeutralStanding,
  normalizeStanding,
  standingDefenderProgressClass,
  standingOwnerTextToneClass,
} from "@/features/shared";
import { TimerStandingType } from "../types";

export function normalizeTimerStanding(value: string): TimerStandingType {
  return normalizeStanding(value) as TimerStandingType;
}

export function isNeutralTimerStanding(value: string): boolean {
  return isNeutralStanding(value);
}

export { standingDefenderProgressClass, standingOwnerTextToneClass };
