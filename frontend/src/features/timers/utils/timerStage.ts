import { TimerStageLabel } from "../types";

export type TimerStageTone =
  | "blue"
  | "yellow"
  | "green"
  | "purple"
  | "gray"
  | "red"
  | "orange";

export function normalizeTimerStageLabel(value?: string): TimerStageLabel | "" {
  const cleaned = (value ?? "").trim().toLowerCase();
  switch (cleaned) {
    case TimerStageLabel.Armor:
      return TimerStageLabel.Armor;
    case TimerStageLabel.Hull:
      return TimerStageLabel.Hull;
    case TimerStageLabel.Reinforcement:
      return TimerStageLabel.Reinforcement;
    case TimerStageLabel.InitialVulnerability:
      return TimerStageLabel.InitialVulnerability;
    case TimerStageLabel.NotApplicable:
      return TimerStageLabel.NotApplicable;
    case TimerStageLabel.Anchoring:
      return TimerStageLabel.Anchoring;
    case TimerStageLabel.Unanchoring:
      return TimerStageLabel.Unanchoring;
    case TimerStageLabel.ExtractionWindow:
      return TimerStageLabel.ExtractionWindow;
    case TimerStageLabel.Custom:
      return TimerStageLabel.Custom;
    default:
      return "";
  }
}

export function timerStageTone(value?: string): TimerStageTone {
  switch (normalizeTimerStageLabel(value)) {
    case TimerStageLabel.Armor:
      return "yellow";
    case TimerStageLabel.Hull:
      return "red";
    case TimerStageLabel.Reinforcement:
      return "red";
    case TimerStageLabel.InitialVulnerability:
      return "purple";
    case TimerStageLabel.Anchoring:
      return "blue";
    case TimerStageLabel.Unanchoring:
      return "orange";
    case TimerStageLabel.ExtractionWindow:
      return "green";
    default:
      return "gray";
  }
}
