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
import { Tone } from "@/features/shared";

export type StructureTone =
  | Tone.Blue
  | Tone.Yellow
  | Tone.Green
  | Tone.Purple
  | Tone.Gray
  | Tone.Red
  | Tone.LightBlue;
export type ContextTone =
  | Tone.Blue
  | Tone.Yellow
  | Tone.Green
  | Tone.Purple
  | Tone.Gray
  | Tone.Red
  | Tone.Orange;
export type ReplacementTone = Tone.Blue | Tone.Green | Tone.Purple | Tone.Gray;

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

export type HostilityOption = {
  value: TimerStandingType;
  label: string;
  icon: LucideIcon;
  tone: StructureTone;
};

export type ReplacementOption = {
  value: TimerReplacementAction;
  label: string;
  icon: LucideIcon;
  tone: ReplacementTone;
};

export type SeverityOption = {
  value: TimerSeverity;
  label: string;
  tone: StructureTone;
  icon: LucideIcon;
};

export const structureGroups: StructureGroup[] = [
  {
    group: "Citadels",
    options: [
      {
        value: TimerStructureType.UpwellCitadelKeepstar,
        label: "Keepstar",
        icon: Building2,
        tone: Tone.Blue,
      },
      {
        value: TimerStructureType.UpwellCitadelFortizar,
        label: "Fortizar",
        icon: Castle,
        tone: Tone.Blue,
      },
      {
        value: TimerStructureType.UpwellCitadelAstrahus,
        label: "Astrahus",
        icon: House,
        tone: Tone.Blue,
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
        tone: Tone.Yellow,
      },
      {
        value: TimerStructureType.UpwellEngineeringAzbel,
        label: "Azbel",
        icon: Factory,
        tone: Tone.Yellow,
      },
      {
        value: TimerStructureType.UpwellEngineeringRaitaru,
        label: "Raitaru",
        icon: Factory,
        tone: Tone.Yellow,
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
        tone: Tone.Yellow,
      },
      {
        value: TimerStructureType.UpwellRefineryAthanor,
        label: "Athanor",
        icon: Factory,
        tone: Tone.Yellow,
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
        tone: Tone.Green,
      },
      {
        value: TimerStructureType.PharoluxCynoBeacon,
        label: "Pharolux Cyno Beacon",
        icon: CircleDot,
        tone: Tone.Green,
      },
      {
        value: TimerStructureType.TenebrexCynoJammer,
        label: "Tenebrex Cyno Jammer",
        icon: CircleDot,
        tone: Tone.Green,
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
        tone: Tone.Purple,
      },
      {
        value: TimerStructureType.SovereigntyHub,
        label: "Sovereignty Hub",
        icon: Building2,
        tone: Tone.Purple,
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
        tone: Tone.LightBlue,
      },
      {
        value: TimerStructureType.MetenoxMoonDrill,
        label: "Metenox Moon Drill",
        icon: Moon,
        tone: Tone.Gray,
      },
      {
        value: TimerStructureType.MercenaryDen,
        label: "Mercenary Den",
        icon: ShieldAlert,
        tone: Tone.Red,
      },
      {
        value: TimerStructureType.CustomsOfficePoco,
        label: "Customs Office (POCO)",
        icon: DollarSign,
        tone: Tone.Gray,
      },
      {
        value: TimerStructureType.Custom,
        label: "Other/Misc",
        icon: CircleHelp,
        tone: Tone.Gray,
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

const upwellDualReinforcementStructures = new Set<TimerStructureType>([
  TimerStructureType.UpwellCitadelFortizar,
  TimerStructureType.UpwellCitadelKeepstar,
  TimerStructureType.UpwellEngineeringSotiyo,
  TimerStructureType.UpwellEngineeringAzbel,
  TimerStructureType.UpwellRefineryTatara,
]);

const upwellSingleReinforcementStructures = new Set<TimerStructureType>([
  TimerStructureType.UpwellCitadelAstrahus,
  TimerStructureType.UpwellEngineeringRaitaru,
  TimerStructureType.UpwellRefineryAthanor,
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

export function isSingleStageReinforcementStructure(
  structureType: TimerStructureType | "",
): boolean {
  return (
    structureType !== "" &&
    (singleExitReinforcementStructures.has(structureType) ||
      upwellSingleReinforcementStructures.has(structureType) ||
      navigationStructures.has(structureType))
  );
}

const anchoringTimerStructures = new Set<TimerStructureType>([
  TimerStructureType.UpwellCitadelKeepstar,
  TimerStructureType.UpwellCitadelFortizar,
  TimerStructureType.UpwellCitadelAstrahus,
  TimerStructureType.UpwellEngineeringSotiyo,
  TimerStructureType.UpwellEngineeringAzbel,
  TimerStructureType.UpwellEngineeringRaitaru,
  TimerStructureType.UpwellRefineryTatara,
  TimerStructureType.UpwellRefineryAthanor,
  TimerStructureType.AnsiblexJumpBridge,
  TimerStructureType.PharoluxCynoBeacon,
  TimerStructureType.TenebrexCynoJammer,
  TimerStructureType.OrbitalSkyhook,
  TimerStructureType.MetenoxMoonDrill,
  TimerStructureType.CustomsOfficePoco,
  TimerStructureType.PlayerOwnedStarbase,
]);

const unanchoringTimerStructures = new Set<TimerStructureType>([
  ...anchoringTimerStructures,
]);

const extractionTimerStructures = new Set<TimerStructureType>([
  TimerStructureType.UpwellRefineryAthanor,
  TimerStructureType.UpwellRefineryTatara,
  TimerStructureType.OrbitalSkyhook,
  TimerStructureType.MetenoxMoonDrill,
  TimerStructureType.MercenaryDen,
]);

function supportsAnchoringTimer(
  structureType: TimerStructureType | "",
): boolean {
  return (
    structureType === TimerStructureType.Custom ||
    (structureType !== "" && anchoringTimerStructures.has(structureType))
  );
}

function supportsUnanchoringTimer(
  structureType: TimerStructureType | "",
): boolean {
  return (
    structureType === TimerStructureType.Custom ||
    (structureType !== "" && unanchoringTimerStructures.has(structureType))
  );
}

function supportsExtractionTimer(
  structureType: TimerStructureType | "",
): boolean {
  return structureType !== "" && extractionTimerStructures.has(structureType);
}

export const planetOnlyStructureTypes = new Set<TimerStructureType>([
  TimerStructureType.OrbitalSkyhook,
  TimerStructureType.MercenaryDen,
  TimerStructureType.CustomsOfficePoco,
]);
export const moonOnlyStructureTypes = new Set<TimerStructureType>([
  TimerStructureType.MetenoxMoonDrill,
]);

export const hostilityOptions: HostilityOption[] = [
  {
    value: TimerStandingType.Ours,
    label: "Ours",
    icon: ShieldCheck,
    tone: Tone.Blue,
  },
  {
    value: TimerStandingType.Friendly,
    label: "Friendly",
    icon: Handshake,
    tone: Tone.Green,
  },
  {
    value: TimerStandingType.Neutral,
    label: "Neutral",
    icon: Flag,
    tone: Tone.Gray,
  },
  {
    value: TimerStandingType.Complicated,
    label: "It's Complicated",
    icon: AlertTriangle,
    tone: Tone.Yellow,
  },
  {
    value: TimerStandingType.Hostile,
    label: "Hostile",
    icon: Swords,
    tone: Tone.Red,
  },
];

export const replacementOptions: ReplacementOption[] = [
  {
    value: TimerReplacementAction.NotReplaceable,
    label: "No Replacement",
    icon: ShieldAlert,
    tone: Tone.Gray,
  },
  {
    value: TimerReplacementAction.LogiReplacement,
    label: "Logi Replacement",
    icon: Wrench,
    tone: Tone.Blue,
  },
  {
    value: TimerReplacementAction.CorpReplacement,
    label: "Corp Replacement",
    icon: Building2,
    tone: Tone.Green,
  },
  {
    value: TimerReplacementAction.AllianceReplacement,
    label: "Alliance Replacement",
    icon: Landmark,
    tone: Tone.Purple,
  },
];
export const hostilityByValue = new Map(
  hostilityOptions.map((option) => [option.value, option]),
);
export const replacementByValue = new Map(
  replacementOptions.map((option) => [option.value, option]),
);

export const severityOptions: SeverityOption[] = [
  {
    value: TimerSeverity.Low,
    label: "Low",
    tone: Tone.Blue,
    icon: Circle,
  },
  {
    value: TimerSeverity.Medium,
    label: "Medium",
    tone: Tone.Green,
    icon: CircleDot,
  },
  {
    value: TimerSeverity.High,
    label: "High",
    tone: Tone.Yellow,
    icon: AlertTriangle,
  },
  {
    value: TimerSeverity.Critical,
    label: "Critical",
    tone: Tone.Red,
    icon: Swords,
  },
];
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
      return Tone.Yellow;
    case TimerStageLabel.Hull:
    case TimerStageLabel.Reinforcement:
      return Tone.Red;
    case TimerStageLabel.InitialVulnerability:
      return Tone.Purple;
    case TimerStageLabel.Anchoring:
      return Tone.Blue;
    case TimerStageLabel.Unanchoring:
      return Tone.Orange;
    case TimerStageLabel.ExtractionWindow:
      return Tone.Green;
    default:
      return Tone.Gray;
  }
}

function reinforcementSelectionForStage(
  stage: TimerStageLabel,
): TimerContextSelection {
  switch (stage) {
    case TimerStageLabel.Armor:
      return TimerContextSelection.Armor;
    case TimerStageLabel.Hull:
      return TimerContextSelection.Hull;
    case TimerStageLabel.Reinforcement:
      return TimerContextSelection.Reinforcement;
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
    if (!supportsAnchoringTimer(structureType)) {
      return [
        { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
      ];
    }
    return [{ value: TimerStageLabel.Anchoring, label: "Anchoring" }];
  }
  if (timerKind === TimerKind.Unanchoring) {
    if (!supportsUnanchoringTimer(structureType)) {
      return [
        { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
      ];
    }
    return [{ value: TimerStageLabel.Unanchoring, label: "Unanchoring" }];
  }
  if (timerKind === TimerKind.Extraction) {
    return [
      { value: TimerStageLabel.ExtractionWindow, label: "Extraction Window" },
    ];
  }
  if (timerKind === TimerKind.Custom) {
    return [{ value: TimerStageLabel.Custom, label: "Custom" }];
  }
  if (structureType === TimerStructureType.Custom) {
    return [{ value: TimerStageLabel.NotApplicable, label: "Not Applicable" }];
  }
  if (structureType && upwellDualReinforcementStructures.has(structureType)) {
    return [
      { value: TimerStageLabel.Armor, label: "Armor" },
      { value: TimerStageLabel.Hull, label: "Hull" },
      {
        value: TimerStageLabel.InitialVulnerability,
        label: "Initial Vulnerability",
      },
      { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
    ];
  }
  if (structureType && navigationStructures.has(structureType)) {
    return [
      { value: TimerStageLabel.Reinforcement, label: "Reinforcement" },
      {
        value: TimerStageLabel.InitialVulnerability,
        label: "Initial Vulnerability",
      },
      { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
    ];
  }
  if (structureType && upwellSingleReinforcementStructures.has(structureType)) {
    return [
      { value: TimerStageLabel.Reinforcement, label: "Reinforcement" },
      {
        value: TimerStageLabel.InitialVulnerability,
        label: "Initial Vulnerability",
      },
      { value: TimerStageLabel.NotApplicable, label: "Not Applicable" },
    ];
  }
  if (structureType && singleExitReinforcementStructures.has(structureType)) {
    return [
      { value: TimerStageLabel.Reinforcement, label: "Reinforcement" },
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
  const anchoringOptions: ContextOption[] = supportsAnchoringTimer(
    structureType,
  )
    ? [
        {
          value: TimerContextSelection.Anchoring,
          label: "Anchoring",
          timerKind: TimerKind.Anchoring,
          stageLabel: TimerStageLabel.Anchoring,
          tone: Tone.Blue,
        },
      ]
    : [];
  const unanchoringOptions: ContextOption[] = supportsUnanchoringTimer(
    structureType,
  )
    ? [
        {
          value: TimerContextSelection.Unanchoring,
          label: "Unanchoring",
          timerKind: TimerKind.Unanchoring,
          stageLabel: TimerStageLabel.Unanchoring,
          tone: Tone.Orange,
        },
      ]
    : [];
  const extractionOptions: ContextOption[] = supportsExtractionTimer(
    structureType,
  )
    ? [
        {
          value: TimerContextSelection.ExtractionWindow,
          label: "Extraction",
          timerKind: TimerKind.Extraction,
          stageLabel: TimerStageLabel.ExtractionWindow,
          tone: Tone.Green,
        },
      ]
    : [];
  return [
    ...reinforcement,
    ...anchoringOptions,
    ...unanchoringOptions,
    ...extractionOptions,
    {
      value: TimerContextSelection.Custom,
      label: "Custom",
      timerKind: TimerKind.Custom,
      stageLabel: TimerStageLabel.Custom,
      tone: Tone.Gray,
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
    stageLabel: TimerStageLabel.Hull,
  },
  [TimerContextSelection.Reinforcement]: {
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.Reinforcement,
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
