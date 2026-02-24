import { normalizeTimerStanding } from "./timerStanding";

export function hostilityRowToneClass(standing: string): string {
  const value = normalizeTimerStanding(standing);
  switch (value) {
    case "ours":
      return "border-sky-400/70 bg-sky-500/[0.08] dark:border-sky-700/45 dark:bg-sky-500/[0.04]";
    case "friendly":
      return "border-emerald-400/70 bg-emerald-500/[0.08] dark:border-emerald-700/45 dark:bg-emerald-500/[0.04]";
    case "complicated":
      return "border-amber-400/70 bg-amber-500/[0.08] dark:border-amber-700/45 dark:bg-amber-500/[0.04]";
    case "hostile":
      return "border-red-400/70 bg-red-500/[0.08] dark:border-red-700/45 dark:bg-red-500/[0.04]";
    case "neutral":
    default:
      return "border-slate-400/80 dark:border-slate-700/70";
  }
}
