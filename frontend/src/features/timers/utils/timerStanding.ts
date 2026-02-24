import { TimerStandingType } from "../types";

export function normalizeTimerStanding(value: string): TimerStandingType {
  const cleaned = value.trim().toLowerCase();
  switch (cleaned) {
    case TimerStandingType.Ours:
      return TimerStandingType.Ours;
    case TimerStandingType.Friendly:
      return TimerStandingType.Friendly;
    case TimerStandingType.Complicated:
      return TimerStandingType.Complicated;
    case TimerStandingType.Hostile:
      return TimerStandingType.Hostile;
    case TimerStandingType.Neutral:
    default:
      return TimerStandingType.Neutral;
  }
}

export function isNeutralTimerStanding(value: string): boolean {
  return normalizeTimerStanding(value) === TimerStandingType.Neutral;
}

export function standingOwnerTextToneClass(value: string): string {
  switch (normalizeTimerStanding(value)) {
    case TimerStandingType.Ours:
      return "timer-owner-tone-ours";
    case TimerStandingType.Friendly:
      return "timer-owner-tone-friendly";
    case TimerStandingType.Complicated:
      return "timer-owner-tone-complicated";
    case TimerStandingType.Hostile:
      return "timer-owner-tone-hostile";
    case TimerStandingType.Neutral:
    default:
      return "timer-owner-tone-neutral";
  }
}

export function standingDefenderProgressClass(value: string): string {
  switch (normalizeTimerStanding(value)) {
    case TimerStandingType.Ours:
      return "bg-sky-500/90";
    case TimerStandingType.Friendly:
      return "bg-emerald-500/90";
    case TimerStandingType.Complicated:
      return "bg-amber-500/90";
    case TimerStandingType.Hostile:
      return "bg-red-500/90";
    case TimerStandingType.Neutral:
    default:
      return "bg-slate-400/90";
  }
}
