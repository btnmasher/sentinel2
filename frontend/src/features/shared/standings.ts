import { Tone, badgeToneClass } from "./tones";

export enum StandingType {
  Ours = "ours",
  Friendly = "friendly",
  Neutral = "neutral",
  Complicated = "complicated",
  Hostile = "hostile",
}

export function normalizeStanding(value: string): StandingType {
  const cleaned = value.trim().toLowerCase();
  switch (cleaned) {
    case StandingType.Ours:
      return StandingType.Ours;
    case StandingType.Friendly:
      return StandingType.Friendly;
    case StandingType.Complicated:
      return StandingType.Complicated;
    case StandingType.Hostile:
      return StandingType.Hostile;
    case StandingType.Neutral:
    default:
      return StandingType.Neutral;
  }
}

export function isNeutralStanding(value: string): boolean {
  return normalizeStanding(value) === StandingType.Neutral;
}

export function standingTone(value: string): Tone {
  switch (normalizeStanding(value)) {
    case StandingType.Ours:
      return Tone.Blue;
    case StandingType.Friendly:
      return Tone.Green;
    case StandingType.Complicated:
      return Tone.Yellow;
    case StandingType.Hostile:
      return Tone.Red;
    case StandingType.Neutral:
    default:
      return Tone.Gray;
  }
}

export function standingBadgeClass(value: string): string {
  return badgeToneClass(standingTone(value));
}

export function standingOwnerTextToneClass(value: string): string {
  switch (normalizeStanding(value)) {
    case StandingType.Ours:
      return "timer-owner-tone-ours";
    case StandingType.Friendly:
      return "timer-owner-tone-friendly";
    case StandingType.Complicated:
      return "timer-owner-tone-complicated";
    case StandingType.Hostile:
      return "timer-owner-tone-hostile";
    case StandingType.Neutral:
    default:
      return "timer-owner-tone-neutral";
  }
}

export function standingDefenderProgressClass(value: string): string {
  switch (normalizeStanding(value)) {
    case StandingType.Ours:
      return "bg-sky-500/90";
    case StandingType.Friendly:
      return "bg-emerald-500/90";
    case StandingType.Complicated:
      return "bg-amber-500/90";
    case StandingType.Hostile:
      return "bg-red-500/90";
    case StandingType.Neutral:
    default:
      return "bg-slate-400/90";
  }
}

export function hostilityRowToneClass(value: string): string {
  switch (normalizeStanding(value)) {
    case StandingType.Ours:
      return "border-sky-400/70 bg-sky-500/[0.08] dark:border-sky-700/45 dark:bg-sky-500/[0.04]";
    case StandingType.Friendly:
      return "border-emerald-400/70 bg-emerald-500/[0.08] dark:border-emerald-700/45 dark:bg-emerald-500/[0.04]";
    case StandingType.Complicated:
      return "border-amber-400/70 bg-amber-500/[0.08] dark:border-amber-700/45 dark:bg-amber-500/[0.04]";
    case StandingType.Hostile:
      return "border-red-400/70 bg-red-500/[0.08] dark:border-red-700/45 dark:bg-red-500/[0.04]";
    case StandingType.Neutral:
    default:
      return "border-slate-400/80 dark:border-slate-700/70";
  }
}
