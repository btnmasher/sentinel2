export enum TimerStructureType {
  UpwellCitadelKeepstar = "upwell_citadel_keepstar",
  UpwellCitadelFortizar = "upwell_citadel_fortizar",
  UpwellCitadelAstrahus = "upwell_citadel_astrahus",
  UpwellEngineeringSotiyo = "upwell_engineering_sotiyo",
  UpwellEngineeringAzbel = "upwell_engineering_azbel",
  UpwellEngineeringRaitaru = "upwell_engineering_raitaru",
  UpwellRefineryTatara = "upwell_refinery_tatara",
  UpwellRefineryAthanor = "upwell_refinery_athanor",
  AnsiblexJumpBridge = "ansiblex_jump_bridge",
  PharoluxCynoBeacon = "pharolux_cyno_beacon",
  TenebrexCynoJammer = "tenebrex_cyno_jammer",
  OrbitalSkyhook = "orbital_skyhook",
  SovereigntyHub = "sovereignty_hub",
  PlayerOwnedStarbase = "player_owned_starbase",
  MetenoxMoonDrill = "metenox_moon_drill",
  MercenaryDen = "mercenary_den",
  CustomsOfficePoco = "customs_office_poco",
  Custom = "custom",
}

export enum TimerStandingType {
  Ours = "ours",
  Friendly = "friendly",
  Neutral = "neutral",
  Complicated = "complicated",
  Hostile = "hostile",
}

export enum TimerKind {
  Reinforcement = "reinforcement",
  Anchoring = "anchoring",
  Unanchoring = "unanchoring",
  Extraction = "extraction",
  Custom = "custom",
}

export enum TimerStageLabel {
  Armor = "armor",
  Structure = "structure",
  InitialVulnerability = "initial_vulnerability",
  NotApplicable = "not_applicable",
  Anchoring = "anchoring",
  Unanchoring = "unanchoring",
  ExtractionWindow = "extraction_window",
  PickupWindow = "pickup_window",
  Custom = "custom",
}

export enum TimerContextSelection {
  Armor = "armor",
  Hull = "hull",
  InitialVulnerability = "initial_vulnerability",
  NotApplicable = "not_applicable",
  Anchoring = "anchoring",
  Unanchoring = "unanchoring",
  ExtractionWindow = "extraction_window",
  PickupWindow = "pickup_window",
  Custom = "custom",
}

export enum TimerReplacementAction {
  NotReplaceable = "not_replaceable",
  LogiReplacement = "logi_replacement",
  CorpReplacement = "corp_replacement",
  AllianceReplacement = "alliance_replacement",
}

export enum TimerSeverity {
  Low = "low",
  Medium = "medium",
  High = "high",
  Critical = "critical",
}

export enum TimerStatus {
  Active = "active",
  Canceled = "canceled",
}

export type TimerRecord = {
  id: string;
  title: string;
  system_id: number;
  system_name: string;
  region_id: number;
  region_name: string;
  standing_type: TimerStandingType;
  timer_kind: TimerKind;
  structure_type: TimerStructureType;
  stage_label: TimerStageLabel;
  planet_id: number;
  planet_name: string;
  moon_id: number;
  moon_name: string;
  owner_corporation_id: number;
  owner_corporation_name: string;
  owner_corporation_ticker: string;
  owner_alliance_id: number;
  owner_alliance_name: string;
  owner_alliance_ticker: string;
  skyhook_fullness_pct: number;
  stage: number;
  total_stages: number;
  severity: TimerSeverity;
  status: TimerStatus;
  expires_at: string;
  source: string;
  source_ref: string;
  notes: string;
  raw_text: string;
  replacement_action: TimerReplacementAction;
  created_by: string;
  canceled_by: string;
  canceled_at: string;
  created: string;
  updated: string;
};

export type ParseTimerResponse = {
  title: string;
  system: string;
  system_id: number;
  timer_kind: TimerKind;
  standing_type: TimerStandingType;
  expires_at: string;
  raw_extract: string;
};

export type TimerSystemOption = {
  id: number;
  name: string;
  region_id: number;
  region: string;
};

export type TimerEntityOption = {
  type: "corporation" | "alliance" | string;
  id: number;
  name: string;
  ticker: string;
  parent_alliance?: {
    id: number;
    name: string;
    ticker: string;
  };
};

export type TimerMoonOption = {
  id: number;
  name: string;
  system_id: number;
};

export type TimerPlanetOption = {
  id: number;
  name: string;
  system_id: number;
};

export type ListTimersResponse = {
  timers: TimerRecord[];
};

export type TimerFormStep = 1 | 2 | 3 | 4;

export type TimerForm = {
  raw_text: string;
  expires_at: string;
  system_id: number;
  system: string;
  structure_type: TimerStructureType | "";
  planet_id: number;
  planet_name: string;
  moon_id: number;
  moon_name: string;
  owner_corporation_id: number;
  owner_corporation_name: string;
  owner_corporation_ticker: string;
  owner_alliance_id: number;
  owner_alliance_name: string;
  owner_alliance_ticker: string;
  standing_type: TimerStandingType | "";
  timer_kind: TimerKind | "";
  stage_label: TimerStageLabel | "";
  context_selection: TimerContextSelection | "";
  replacement_action: TimerReplacementAction | "";
  skyhook_fullness_pct: string;
  severity: TimerSeverity | "";
  title: string;
  notes: string;
  other_structure_note: string;
  timer_kind_note: string;
};
