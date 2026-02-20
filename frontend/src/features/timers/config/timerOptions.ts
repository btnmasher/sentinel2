import type { LucideIcon } from "lucide-react";
import {
  AlertTriangle,
  Antenna,
  Building2,
  Castle,
  Circle,
  CircleDot,
  CircleHelp,
  DollarSign,
  Factory,
  Flag,
  Handshake,
  House,
  Landmark,
  Moon,
  ShieldAlert,
  ShieldCheck,
  Swords,
  TowerControl,
  Wrench,
} from "lucide-react";
import {
  TimerContextSelection,
  TimerKind,
  TimerReplacementAction,
  TimerSeverity,
  TimerStageLabel,
  TimerStandingType,
  TimerStructureType,
} from "../types";

export type StructureTone =
  | "blue"
  | "yellow"
  | "green"
  | "purple"
  | "gray"
  | "red"
  | "lightblue";
export type ContextTone =
  | "blue"
  | "yellow"
  | "green"
  | "purple"
  | "gray"
  | "red"
  | "orange";
export type ReplacementTone = "blue" | "green" | "purple" | "gray";

export type StructureOption = {
  value: TimerStructureType;
  label: string;
  icon: LucideIcon;
  tone: StructureTone;
};

export type StructureGroup = {
  group: string;
  options: StructureOption[];
};

export type StageOption = { value: TimerStageLabel; label: string };
export type ContextOption = {
  value: TimerContextSelection;
  label: string;
  timerKind: TimerKind;
  stageLabel: TimerStageLabel;
  tone: ContextTone;
};

export const structureGroups: StructureGroup[] = [
  {
    group: "Citadels",
    options: [
      {
        value: TimerStructureType.UpwellCitadelKeepstar,
        label: "Keepstar",
        icon: Building2,
        tone: "blue",
      },
      {
        value: TimerStructureType.UpwellCitadelFortizar,
        label: "Fortizar",
        icon: Castle,
        tone: "blue",
      },
      {
        value: TimerStructureType.UpwellCitadelAstrahus,
        label: "Astrahus",
        icon: House,
        tone: "blue",
      },
    ],
  },
  {
    group: "Engineering Complexes",
    options: [
      {
        value: TimerStructureType.UpwellEngineeringSotiyo,
        label: "Sotiyo",
        icon: Factory,
        tone: "yellow",
      },
      {
        value: TimerStructureType.UpwellEngineeringAzbel,
        label: "Azbel",
        icon: Factory,
        tone: "yellow",
      },
      {
        value: TimerStructureType.UpwellEngineeringRaitaru,
        label: "Raitaru",
        icon: Factory,
        tone: "yellow",
      },
    ],
  },
  {
    group: "Refineries",
    options: [
      {
        value: TimerStructureType.UpwellRefineryTatara,
        label: "Tatara",
        icon: Factory,
        tone: "yellow",
      },
      {
        value: TimerStructureType.UpwellRefineryAthanor,
        label: "Athanor",
        icon: Factory,
        tone: "yellow",
      },
    ],
  },
  {
    group: "Navigation Structures",
    options: [
      {
        value: TimerStructureType.AnsiblexJumpBridge,
        label: "Ansiblex Jump Gate",
        icon: Circle,
        tone: "green",
      },
      {
        value: TimerStructureType.PharoluxCynoBeacon,
        label: "Pharolux Cyno Beacon",
        icon: CircleDot,
        tone: "green",
      },
      {
        value: TimerStructureType.TenebrexCynoJammer,
        label: "Tenebrex Cyno Jammer",
        icon: CircleDot,
        tone: "green",
      },
    ],
  },
  {
    group: "Sovereignty Structures",
    options: [
      {
        value: TimerStructureType.OrbitalSkyhook,
        label: "Orbital Skyhook",
        icon: TowerControl,
        tone: "purple",
      },
      {
        value: TimerStructureType.SovereigntyHub,
        label: "Sovereignty Hub",
        icon: Building2,
        tone: "purple",
      },
    ],
  },
  {
    group: "Miscellaneous",
    options: [
      {
        value: TimerStructureType.PlayerOwnedStarbase,
        label: "Player-owned Starbase (POS)",
        icon: Antenna,
        tone: "lightblue",
      },
      {
        value: TimerStructureType.MetenoxMoonDrill,
        label: "Metenox Moon Drill",
        icon: Moon,
        tone: "gray",
      },
      {
        value: TimerStructureType.MercenaryDen,
        label: "Mercenary Den",
        icon: ShieldAlert,
        tone: "red",
      },
      {
        value: TimerStructureType.CustomsOfficePoco,
        label: "Customs Office (POCO)",
        icon: DollarSign,
        tone: "gray",
      },
      {
        value: TimerStructureType.Custom,
        label: "Other/Misc",
        icon: CircleHelp,
        tone: "gray",
      },
    ],
  },
];

export const structureOptions = structureGroups.flatMap(
  (group) => group.options,
);
export const structureByValue = new Map(
  structureOptions.map((option) => [option.value, option]),
);

const upwellCoreStructures = new Set<TimerStructureType>([
  TimerStructureType.UpwellCitadelAstrahus,
  TimerStructureType.UpwellCitadelFortizar,
  TimerStructureType.UpwellCitadelKeepstar,
  TimerStructureType.UpwellEngineeringRaitaru,
  TimerStructureType.UpwellEngineeringAzbel,
  TimerStructureType.UpwellEngineeringSotiyo,
  TimerStructureType.UpwellRefineryAthanor,
  TimerStructureType.UpwellRefineryTatara,
]);

const navigationStructures = new Set<TimerStructureType>([
  TimerStructureType.AnsiblexJumpBridge,
  TimerStructureType.PharoluxCynoBeacon,
  TimerStructureType.TenebrexCynoJammer,
]);

const singleExitReinforcementStructures = new Set<TimerStructureType>([
  TimerStructureType.OrbitalSkyhook,
  TimerStructureType.MetenoxMoonDrill,
  TimerStructureType.MercenaryDen,
  TimerStructureType.SovereigntyHub,
  TimerStructureType.CustomsOfficePoco,
  TimerStructureType.PlayerOwnedStarbase,
]);

export const planetOnlyStructureTypes = new Set<TimerStructureType>([
  TimerStructureType.OrbitalSkyhook,
  TimerStructureType.MercenaryDen,
  TimerStructureType.CustomsOfficePoco,
]);
export const moonOnlyStructureTypes = new Set<TimerStructureType>([
  TimerStructureType.MetenoxMoonDrill,
]);

export const hostilityOptions = [
  {
    value: TimerStandingType.Ours,
    label: "Ours",
    icon: ShieldCheck,
    tone: "blue",
  },
  {
    value: TimerStandingType.Friendly,
    label: "Friendly",
    icon: Handshake,
    tone: "green",
  },
  {
    value: TimerStandingType.Neutral,
    label: "Neutral",
    icon: Flag,
    tone: "gray",
  },
  {
    value: TimerStandingType.Complicated,
    label: "It's Complicated",
    icon: AlertTriangle,
    tone: "yellow",
  },
  {
    value: TimerStandingType.Hostile,
    label: "Hostile",
    icon: Swords,
    tone: "red",
  },
] as const;

export const replacementOptions = [
  {
    value: TimerReplacementAction.NotReplaceable,
    label: "No Replacement",
    icon: ShieldAlert,
    tone: "gray",
  },
  {
    value: TimerReplacementAction.LogiReplacement,
    label: "Logi Replacement",
    icon: Wrench,
    tone: "blue",
  },
  {
    value: TimerReplacementAction.CorpReplacement,
    label: "Corp Replacement",
    icon: Building2,
    tone: "green",
  },
  {
    value: TimerReplacementAction.AllianceReplacement,
    label: "Alliance Replacement",
    icon: Landmark,
    tone: "purple",
  },
] as const;
export const hostilityByValue = new Map(
  hostilityOptions.map((option) => [option.value, option]),
);
export const replacementByValue = new Map(
  replacementOptions.map((option) => [option.value, option]),
);

export const severityOptions = [
  {
    value: TimerSeverity.Low,
    label: "Low",
    tone: "blue",
    icon: Circle,
  },
  {
    value: TimerSeverity.Medium,
    label: "Medium",
    tone: "green",
    icon: CircleDot,
  },
  {
    value: TimerSeverity.High,
    label: "High",
    tone: "yellow",
    icon: AlertTriangle,
  },
  {
    value: TimerSeverity.Critical,
    label: "Critical",
    tone: "red",
    icon: Swords,
  },
] as const;
export const severityByValue = new Map(
  severityOptions.map((option) => [option.value, option]),
);

export const timerKindLabels: Record<TimerKind, string> = {
  [TimerKind.Reinforcement]: "Reinforcement",
  [TimerKind.Anchoring]: "Anchoring",
  [TimerKind.Unanchoring]: "Unanchoring",
  [TimerKind.Extraction]: "Extraction",
  [TimerKind.Custom]: "Custom",
};

function reinforcementToneForStage(stage: TimerStageLabel): ContextTone {
  switch (stage) {
    case TimerStageLabel.Armor:
      return "yellow";
    case TimerStageLabel.Structure:
      return "red";
    case TimerStageLabel.InitialVulnerability:
      return "purple";
    default:
      return "gray";
  }
}

function reinforcementSelectionForStage(
  stage: TimerStageLabel,
): TimerContextSelection {
  switch (stage) {
    case TimerStageLabel.Armor:
      return TimerContextSelection.Armor;
    case TimerStageLabel.Structure:
      return TimerContextSelection.Hull;
    case TimerStageLabel.InitialVulnerability:
      return TimerContextSelection.InitialVulnerability;
    default:
      return TimerContextSelection.NotApplicable;
  }
}

export function stageOptionsFor(
  structureType: TimerStructureType | "",
  timerKind: TimerKind,
): StageOption[] {
  if (timerKind === TimerKind.Anchoring) {
    return [{ value: TimerStageLabel.Anchoring, label: "Anchoring" }];
  }
  if (timerKind === TimerKind.Unanchoring) {
    return [{ value: TimerStageLabel.Unanchoring, label: "Unanchoring" }];
  }
  if (timerKind === TimerKind.Extraction) {
    return [
      { value: TimerStageLabel.ExtractionWindow, label: "Extraction Window" },
      { value: TimerStageLabel.PickupWindow, label: "Pickup Window" },
    ];
  }
  if (timerKind === TimerKind.Custom) {
    return [{ value: TimerStageLabel.Custom, label: "Custom" }];
  }
  if (structureType === TimerStructureType.Custom) {
    return [{ value: TimerStageLabel.NotApplicable, label: "Not Applicable" }];
  }
  if (structureType && upwellCoreStructures.has(structureType)) {
    return [
      { value: TimerStageLabel.Armor, label: "Armor" },
      { value: TimerStageLabel.Structure, label: "Hull" },
      {
        value: TimerStageLabel.InitialVulnerability,
        label: "Initial Vulnerability",
      },
      { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
    ];
  }
  if (structureType && navigationStructures.has(structureType)) {
    return [
      { value: TimerStageLabel.Armor, label: "Armor" },
      { value: TimerStageLabel.Structure, label: "Hull" },
      {
        value: TimerStageLabel.InitialVulnerability,
        label: "Initial Vulnerability",
      },
      { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
    ];
  }
  if (structureType && singleExitReinforcementStructures.has(structureType)) {
    return [
      { value: TimerStageLabel.Structure, label: "Hull" },
      {
        value: TimerStageLabel.InitialVulnerability,
        label: "Initial Vulnerability",
      },
      { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
    ];
  }
  return [{ value: TimerStageLabel.NotApplicable, label: "Not Applicable" }];
}

export function contextOptionsFor(
  structureType: TimerStructureType | "",
): ContextOption[] {
  const reinforcement: ContextOption[] = stageOptionsFor(
    structureType,
    TimerKind.Reinforcement,
  ).map((stage) => ({
    value: reinforcementSelectionForStage(stage.value),
    label: stage.label,
    timerKind: TimerKind.Reinforcement,
    stageLabel: stage.value,
    tone: reinforcementToneForStage(stage.value),
  }));
  return [
    ...reinforcement,
    {
      value: TimerContextSelection.Anchoring,
      label: "Anchoring",
      timerKind: TimerKind.Anchoring,
      stageLabel: TimerStageLabel.Anchoring,
      tone: "blue",
    },
    {
      value: TimerContextSelection.Unanchoring,
      label: "Unanchoring",
      timerKind: TimerKind.Unanchoring,
      stageLabel: TimerStageLabel.Unanchoring,
      tone: "orange",
    },
    {
      value: TimerContextSelection.ExtractionWindow,
      label: "Extraction",
      timerKind: TimerKind.Extraction,
      stageLabel: TimerStageLabel.ExtractionWindow,
      tone: "green",
    },
    {
      value: TimerContextSelection.Custom,
      label: "Custom",
      timerKind: TimerKind.Custom,
      stageLabel: TimerStageLabel.Custom,
      tone: "gray",
    },
  ];
}

const timerContextMeta: Record<
  TimerContextSelection,
  { timerKind: TimerKind; stageLabel: TimerStageLabel }
> = {
  [TimerContextSelection.Armor]: {
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.Armor,
  },
  [TimerContextSelection.Hull]: {
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.Structure,
  },
  [TimerContextSelection.InitialVulnerability]: {
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.InitialVulnerability,
  },
  [TimerContextSelection.NotApplicable]: {
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.NotApplicable,
  },
  [TimerContextSelection.Anchoring]: {
    timerKind: TimerKind.Anchoring,
    stageLabel: TimerStageLabel.Anchoring,
  },
  [TimerContextSelection.Unanchoring]: {
    timerKind: TimerKind.Unanchoring,
    stageLabel: TimerStageLabel.Unanchoring,
  },
  [TimerContextSelection.ExtractionWindow]: {
    timerKind: TimerKind.Extraction,
    stageLabel: TimerStageLabel.ExtractionWindow,
  },
  [TimerContextSelection.PickupWindow]: {
    timerKind: TimerKind.Extraction,
    stageLabel: TimerStageLabel.PickupWindow,
  },
  [TimerContextSelection.Custom]: {
    timerKind: TimerKind.Custom,
    stageLabel: TimerStageLabel.Custom,
  },
};

export function timerFieldsFromContextSelection(
  selection: TimerContextSelection,
): {
  timerKind: TimerKind;
  stageLabel: TimerStageLabel;
} {
  return timerContextMeta[selection];
}

export function timerContextSelectionFromFields(
  timerKind: TimerKind | "",
  stageLabel: TimerStageLabel | "",
): TimerContextSelection | "" {
  if (!timerKind || !stageLabel) {
    return "";
  }
  const entry = (
    Object.entries(timerContextMeta) as Array<
      [
        TimerContextSelection,
        { timerKind: TimerKind; stageLabel: TimerStageLabel },
      ]
    >
  ).find(
    ([, meta]) =>
      meta.timerKind === timerKind && meta.stageLabel === stageLabel,
  );
  return entry?.[0] || "";
}
