import { useEffect, useMemo, useState } from "react";
import type { ComponentType } from "react";
import {
  AlertTriangle,
  Circle,
  CircleHelp,
  Clock3,
  Copy,
  Droplets,
  Flag,
  Globe2,
  MapPin,
  Moon,
  Swords,
  ShieldAlert,
  ShieldCheck,
  SquarePen,
  Trash2,
  Wrench,
  Building2,
} from "lucide-react";
import { useAllianceLogo, useCorporationLogo } from "@/hooks/useEveImage";
import { api } from "@/config/api";
import {
  hostilityByValue,
  replacementByValue,
  severityByValue,
  structureByValue,
  timerKindLabels,
} from "../config/timerOptions";
import {
  countdownTone,
  formatCountdown,
  formatReplacement,
  formatStageLabel,
  formatStanding,
  formatStructureType,
  formatTicker,
  formatTimerDateParts,
  isCountdownExpired,
  type FormattedDateParts,
  type TimerDateParts,
} from "../formatters";
import {
  replacementBadgeClass,
  severityBadgeClass,
  stageBadgeClass,
  standingBadgeClass,
  structureBadgeClassByTone,
} from "../utils/timerBadgeTones";
import { hostilityRowToneClass } from "../utils/timerRowTones";
import {
  TimerKind,
  TimerReplacementAction,
  TimerStandingType,
  TimerStageLabel,
  TimerStatus,
  TimerStructureType,
  type TimerRecord,
} from "../types";
import {
  normalizeTimerStanding,
  standingDefenderProgressClass,
  standingOwnerTextToneClass,
} from "../utils/timerStanding";

type TimerRowProps = {
  timer: TimerRecord;
  canManage: boolean;
  readOnly: boolean;
  use24Hour: boolean;
  nowMs: number;
  systemName: string;
  regionName: string;
  onEdit: (timer: TimerRecord) => void;
  onCancel: (timer: TimerRecord) => void;
  onUncancel: (timer: TimerRecord) => void;
  onCopy: (timer: TimerRecord, systemName: string, regionName: string) => void;
  onDeleteConfirm: (timer: TimerRecord) => void;
};

export default function TimerRow({
  timer,
  canManage,
  readOnly,
  use24Hour,
  nowMs,
  systemName,
  regionName,
  onEdit,
  onCancel,
  onUncancel,
  onCopy,
  onDeleteConfirm,
}: TimerRowProps) {
  const structure = structureByValue.get(timer.structure_type);
  const structureTone = structure?.tone || "gray";
  const StructureIcon = structure?.icon || CircleHelp;
  const standingMeta = hostilityByValue.get(timer.standing_type);
  const StandingIcon = standingMeta?.icon || Flag;
  const replacementMeta = replacementByValue.get(timer.replacement_action);
  const ReplacementIcon = replacementMeta?.icon || ShieldAlert;
  const severityMeta = severityByValue.get(timer.severity);
  const SeverityIcon = severityMeta?.icon || Circle;
  const stageIcon = stageBadgeIcon(timer.stage_label);
  const StageIcon = stageIcon ?? CircleHelp;
  const corpLogo = useCorporationLogo(
    timer.owner_corporation_id || undefined,
    32,
  );
  const allianceLogo = useAllianceLogo(
    timer.owner_alliance_id || undefined,
    32,
  );
  const rowToneClass = hostilityRowToneClass(timer.standing_type);
  const countdownText = formatCountdown(timer.expires_at, nowMs);
  const countdownClass = countdownTone(timer.expires_at, nowMs);
  const cancelled = timer.status === TimerStatus.Canceled;
  const expired = isCountdownExpired(timer.expires_at, nowMs);
  const staticDateParts = formatTimerDateParts(timer.expires_at, use24Hour);
  const kindLabel = timerKindLabels[timer.timer_kind] || timer.timer_kind;
  const includeKindInFallback =
    timer.stage_label !== TimerStageLabel.NotApplicable;
  const resolvedTitle =
    timer.title?.trim() ||
    `${systemName} ${formatStructureType(timer.structure_type)}${includeKindInFallback ? ` ${kindLabel}` : ""}`;

  return (
    <div className={`rounded-xl border bg-base-300/30 p-2.5 ${rowToneClass}`}>
      <div className="grid grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_auto] gap-x-2 gap-y-2">
        <TimerRowCountdown
          countdownClass={countdownClass}
          countdownText={countdownText}
          cancelled={cancelled}
          expired={expired}
          staticDateParts={staticDateParts}
          expiresAt={timer.expires_at}
          severity={timer.severity}
          severityLabel={severityMeta?.label || timer.severity}
          severityIcon={SeverityIcon}
        />
        <TimerRowActions
          timer={timer}
          canManage={canManage}
          readOnly={readOnly}
          systemName={systemName}
          regionName={regionName}
          onCopy={onCopy}
          onEdit={onEdit}
          onCancel={onCancel}
          onUncancel={onUncancel}
          onDeleteConfirm={onDeleteConfirm}
        />
        <TimerRowLocationOwner
          timer={timer}
          systemName={systemName}
          regionName={regionName}
          resolvedTitle={resolvedTitle}
          corpLogo={corpLogo}
          allianceLogo={allianceLogo}
        />
        <TimerRowBadges
          timer={timer}
          structureTone={structureTone}
          structureIcon={StructureIcon}
          stageIcon={StageIcon}
          standingIcon={StandingIcon}
          replacementIcon={ReplacementIcon}
        />
      </div>
    </div>
  );
}

type TimerRowCountdownProps = {
  countdownClass: string;
  countdownText: string;
  cancelled: boolean;
  expired: boolean;
  staticDateParts: TimerDateParts | null;
  expiresAt: string;
  severity: TimerRecord["severity"];
  severityLabel: string;
  severityIcon: ComponentType<{ className?: string }>;
};

function TimerRowCountdown({
  countdownClass,
  countdownText,
  cancelled,
  expired,
  staticDateParts,
  expiresAt,
  severity,
  severityLabel,
  severityIcon: SeverityIcon,
}: TimerRowCountdownProps) {
  return (
    <div className="min-w-0">
      <div className="flex items-end gap-1.5">
        <div
          className={`font-mono text-xl font-bold tracking-wide ${countdownClass}`}
        >
          {cancelled && !expired
            ? "CANCELLED"
            : expired
              ? "EXPIRED"
              : countdownText}
        </div>
        {!cancelled && !expired ? (
          <span className="timer-countdown-label">remaining</span>
        ) : null}
        <span
          className={`badge timer-row-badge ${severityBadgeClass(severity)} ${isCriticalSeverity(severity) ? "timer-badge-critical-pulse" : ""}`}
        >
          <SeverityIcon className="h-2.5 w-2.5" />
          {severityLabel}
        </span>
      </div>
      {staticDateParts ? (
        <div className="timer-static-time-row">
          <Clock3 className="timer-static-time-icon" />
          <StyledDateText parts={staticDateParts.local} />
          <span className="timer-static-time-paren">(</span>
          <StyledDateText
            parts={{
              ...staticDateParts.eve,
              timezone: "EVE TIME",
            }}
          />
          <span className="timer-static-time-paren">)</span>
        </div>
      ) : (
        <span className="timer-fallback-time">
          <Clock3 className="h-3.5 w-3.5" /> {expiresAt}
        </span>
      )}
    </div>
  );
}

type TimerRowActionsProps = Pick<
  TimerRowProps,
  | "timer"
  | "canManage"
  | "readOnly"
  | "onEdit"
  | "onCancel"
  | "onUncancel"
  | "onCopy"
  | "onDeleteConfirm"
> & {
  systemName: string;
  regionName: string;
};

function TimerRowActions({
  timer,
  canManage,
  readOnly,
  systemName,
  regionName,
  onCopy,
  onEdit,
  onCancel,
  onUncancel,
  onDeleteConfirm,
}: TimerRowActionsProps) {
  const isExpired = Date.parse(timer.expires_at) <= Date.now();
  const creator = timer.created_by_name?.trim();
  return (
    <div className="flex flex-wrap items-center justify-end gap-1.5 justify-self-end self-start">
      {creator ? (
        <span className="text-[10px] text-slate-500">Added by {creator}</span>
      ) : null}
      <button
        className="btn btn-xs btn-outline"
        onClick={() => onCopy(timer, systemName, regionName)}
      >
        <Copy className="h-3 w-3" />
        Copy
      </button>
      {canManage && !readOnly && (
        <button
          className="btn btn-xs btn-outline"
          onClick={() => onEdit(timer)}
        >
          <SquarePen className="h-3 w-3" />
          Edit
        </button>
      )}
      {canManage && !readOnly && timer.status === TimerStatus.Active && (
        <button
          className="btn btn-xs btn-outline"
          onClick={() => onCancel(timer)}
        >
          Cancel
        </button>
      )}
      {canManage &&
        !readOnly &&
        timer.status === TimerStatus.Canceled &&
        !isExpired && (
          <button
            className="btn btn-xs btn-outline"
            onClick={() => onUncancel(timer)}
          >
            Un-cancel
          </button>
        )}
      {canManage && !readOnly && (
        <button
          className="btn btn-xs btn-error btn-outline"
          onClick={() => onDeleteConfirm(timer)}
        >
          <Trash2 className="h-3 w-3" />
        </button>
      )}
    </div>
  );
}

type TimerRowLocationOwnerProps = {
  timer: TimerRecord;
  systemName: string;
  regionName: string;
  resolvedTitle: string;
  corpLogo: string | null;
  allianceLogo: string | null;
};

function TimerRowLocationOwner({
  timer,
  systemName,
  regionName,
  resolvedTitle,
  corpLogo,
  allianceLogo,
}: TimerRowLocationOwnerProps) {
  const hasSovCampaignProgress = showSovCampaignProgress(timer);
  const sovAttackers = useMemo(
    () => parseSovCampaignAttackers(timer.notes),
    [timer.notes],
  );
  const [attackerAllianceIDByKey, setAttackerAllianceIDByKey] = useState<
    Record<string, number>
  >({});
  const showSovOwnersVs =
    hasSovCampaignProgress &&
    timer.owner_alliance_name &&
    sovAttackers.length > 0;

  useEffect(() => {
    if (!showSovOwnersVs || sovAttackers.length === 0) {
      setAttackerAllianceIDByKey({});
      return;
    }
    let active = true;
    const load = async () => {
      const unresolved = sovAttackers.filter((attacker) => {
        if (attackerAllianceIDCache.has(attacker.key)) {
          return false;
        }
        return true;
      });
      if (unresolved.length > 0) {
        await Promise.all(
          unresolved.map(async (attacker) => {
            try {
              const response = await api.get<{
                entities: TimerEntitySearchItem[];
              }>(
                `/organizations/search?scope=alliance&query=${encodeURIComponent(attacker.query)}&limit=10`,
              );
              const matches = response.data.entities ?? [];
              const resolved = matches.find((entity) => {
                if (entity.type !== "alliance") return false;
                const sameName =
                  entity.name.trim().toLowerCase() ===
                  attacker.name.trim().toLowerCase();
                const sameTicker =
                  attacker.ticker.length > 0 &&
                  entity.ticker.trim().toLowerCase() ===
                    attacker.ticker.trim().toLowerCase();
                return sameName || sameTicker;
              });
              attackerAllianceIDCache.set(attacker.key, resolved?.id ?? 0);
            } catch {
              attackerAllianceIDCache.set(attacker.key, 0);
            }
          }),
        );
      }
      if (!active) return;
      const nextMap: Record<string, number> = {};
      for (const attacker of sovAttackers) {
        nextMap[attacker.key] = attackerAllianceIDCache.get(attacker.key) ?? 0;
      }
      setAttackerAllianceIDByKey(nextMap);
    };
    void load();
    return () => {
      active = false;
    };
  }, [showSovOwnersVs, sovAttackers]);

  return (
    <div className="min-w-0 self-stretch flex flex-col">
      <div>
        <div className="flex items-center gap-1.5 font-semibold">
          <MapPin className="h-4 w-4" />
          <span className="timer-system-name">{systemName}</span>
          <span className="timer-region-name">{regionName}</span>
          {timer.planet_name ? (
            <span className="timer-celestial-chip">
              <Globe2 className="h-3 w-3" />
              <span className="timer-celestial-chip-label">Planet</span>
              <span className="timer-celestial-chip-value">
                {timer.planet_name}
              </span>
            </span>
          ) : null}
          {timer.moon_name ? (
            <span className="timer-celestial-chip">
              <Moon className="h-3 w-3" />
              <span className="timer-celestial-chip-label">Moon</span>
              <span className="timer-celestial-chip-value">
                {timer.moon_name}
              </span>
            </span>
          ) : null}
        </div>
        <div className="timer-title-text">{resolvedTitle}</div>
        {hasSovCampaignProgress ? (
          <SovCampaignProgressBar timer={timer} attackers={sovAttackers} />
        ) : null}
        {timer.notes?.trim() && !hasSovCampaignProgress ? (
          <div className="timer-note-text mt-1 line-clamp-2">{timer.notes}</div>
        ) : null}
      </div>
      {showSovOwnersVs ? (
        <div className="mt-auto pt-1.5 flex items-center gap-2 text-xs">
          <span
            className={`inline-flex items-center gap-1.5 font-semibold ${standingOwnerTextToneClass(timer.standing_type)}`}
          >
            {allianceLogo ? (
              <img
                src={allianceLogo}
                alt="Defender alliance logo"
                className="h-3.5 w-3.5 rounded-[2px]"
                loading="lazy"
              />
            ) : null}
            {formatTicker(timer.owner_alliance_ticker)}{" "}
            {timer.owner_alliance_name}
          </span>
          <span className="inline-flex items-center text-slate-500 dark:text-slate-400">
            <Swords className="h-3 w-3" />
          </span>
          <span className="min-w-0 inline-flex flex-wrap items-center gap-1">
            {sovAttackers.map((attacker) => (
              <SovAttackerChip
                key={attacker.key}
                attacker={attacker}
                allianceID={attackerAllianceIDByKey[attacker.key] ?? 0}
              />
            ))}
          </span>
        </div>
      ) : timer.owner_corporation_name || timer.owner_alliance_name ? (
        <div className="mt-auto pt-1.5 flex flex-wrap items-center gap-1.5 text-xs text-slate-700 dark:text-slate-300">
          {timer.owner_corporation_name ? (
            <span
              className={`inline-flex items-center gap-1.5 font-semibold ${standingOwnerTextToneClass(timer.standing_type)}`}
            >
              {corpLogo ? (
                <img
                  src={corpLogo}
                  alt="Corporation logo"
                  className="h-3.5 w-3.5 rounded-[2px]"
                  loading="lazy"
                />
              ) : null}
              {formatTicker(timer.owner_corporation_ticker)}{" "}
              {timer.owner_corporation_name}
            </span>
          ) : null}
          {timer.owner_alliance_name ? (
            <span
              className={`inline-flex items-center gap-1.5 font-semibold ${standingOwnerTextToneClass(timer.standing_type)}`}
            >
              {allianceLogo ? (
                <img
                  src={allianceLogo}
                  alt="Alliance logo"
                  className="h-3.5 w-3.5 rounded-[2px]"
                  loading="lazy"
                />
              ) : null}
              {formatTicker(timer.owner_alliance_ticker)}{" "}
              {timer.owner_alliance_name}
            </span>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

type TimerRowBadgesProps = {
  timer: TimerRecord;
  structureTone:
    | "blue"
    | "yellow"
    | "green"
    | "purple"
    | "gray"
    | "red"
    | "lightblue";
  structureIcon: ComponentType<{ className?: string }>;
  stageIcon: ComponentType<{ className?: string }>;
  standingIcon: ComponentType<{ className?: string }>;
  replacementIcon: ComponentType<{ className?: string }>;
};

function TimerRowBadges({
  timer,
  structureTone,
  structureIcon: StructureIcon,
  stageIcon: StageIcon,
  standingIcon: StandingIcon,
  replacementIcon: ReplacementIcon,
}: TimerRowBadgesProps) {
  return (
    <div className="flex max-w-[36rem] flex-wrap items-center justify-end gap-1.5 self-end justify-self-end text-xs text-slate-700 dark:text-slate-300">
      <span
        className={`badge timer-row-badge ${standingBadgeClass(timer.standing_type)}`}
      >
        <StandingIcon className="h-2.5 w-2.5" />{" "}
        {formatStanding(timer.standing_type)}
      </span>
      <span
        className={`badge timer-row-badge ${structureBadgeClassByTone(structureTone)}`}
      >
        <StructureIcon className="h-2.5 w-2.5" />{" "}
        {formatStructureType(timer.structure_type)}
      </span>
      {showSkyhookFullnessBadge(timer) ? (
        <span className="badge timer-row-badge timer-skyhook-fullness-badge">
          <Droplets className="h-2.5 w-2.5" />{" "}
          {Math.round(timer.skyhook_fullness_pct)}% Full
        </span>
      ) : null}
      <span
        className={`badge timer-row-badge ${stageBadgeClass(timer.stage_label)}`}
      >
        <StageIcon className="h-2.5 w-2.5" />{" "}
        {formatStageLabel(timer.stage_label)}
      </span>
      {timer.replacement_action &&
      timer.replacement_action !== TimerReplacementAction.NotReplaceable ? (
        <span
          className={`badge timer-row-badge ${replacementBadgeClass(timer.replacement_action)}`}
        >
          <ReplacementIcon className="h-2.5 w-2.5" />{" "}
          {formatReplacement(timer.replacement_action)}
        </span>
      ) : null}
    </div>
  );
}

function showSkyhookFullnessBadge(timer: TimerRecord): boolean {
  const isSkyhookExtraction =
    timer.structure_type === TimerStructureType.OrbitalSkyhook &&
    timer.timer_kind === TimerKind.Extraction;
  return (
    isSkyhookExtraction &&
    Number.isFinite(timer.skyhook_fullness_pct) &&
    timer.skyhook_fullness_pct > 0
  );
}

function showSovCampaignProgress(timer: TimerRecord): boolean {
  if (
    timer.structure_type !== TimerStructureType.SovereigntyHub ||
    !timer.source_ref.startsWith("esi:sovereignty_campaign:")
  ) {
    return false;
  }
  return (
    Number.isFinite(timer.attackers_score_pct) &&
    Number.isFinite(timer.defender_score_pct) &&
    (timer.attackers_score_pct > 0 || timer.defender_score_pct > 0)
  );
}

function SovCampaignProgressBar({
  timer,
  attackers,
}: {
  timer: TimerRecord;
  attackers: ParsedSovAttacker[];
}) {
  const hostileProgressRed = "rgba(239,68,68,0.9)";
  const attackersPct = clampPercent(timer.attackers_score_pct);
  const defenderPct = clampPercent(timer.defender_score_pct);
  const defenderStanding = normalizeTimerStanding(timer.standing_type);
  const defendersHostile = defenderStanding === TimerStandingType.Hostile;
  const strongestAttackerStanding = mostFriendlyAttackerStanding(attackers);
  const bothHostile =
    defendersHostile && strongestAttackerStanding === TimerStandingType.Hostile;
  const attackerFallsToUnaligned =
    defendersHostile && strongestAttackerStanding === null;
  const attackersProgressClass =
    defendersHostile && strongestAttackerStanding
      ? standingDefenderProgressClass(strongestAttackerStanding)
      : "bg-red-500/90";
  const defenderWidthStyle = { width: `${defenderPct}%` };
  const defenderSegmentStyle = bothHostile
    ? {
        ...defenderWidthStyle,
        backgroundImage: `repeating-linear-gradient(-45deg, ${hostileProgressRed} 0px, ${hostileProgressRed} 6px, rgba(100,116,139,0.9) 6px, rgba(100,116,139,0.9) 12px)`,
      }
    : defenderWidthStyle;
  const attackerWidthStyle = { width: `${attackersPct}%` };
  const attackerSegmentStyle = bothHostile
    ? { ...attackerWidthStyle, backgroundColor: hostileProgressRed }
    : attackerFallsToUnaligned
      ? {
          ...attackerWidthStyle,
          backgroundImage:
            "repeating-linear-gradient(-45deg, rgba(167,139,250,0.95) 0px, rgba(167,139,250,0.95) 6px, rgba(100,116,139,0.95) 6px, rgba(100,116,139,0.95) 12px)",
        }
      : attackerWidthStyle;
  return (
    <div className="mt-1.5 w-1/2 min-w-[14rem] max-w-[30rem]">
      <div className="mb-1 flex items-center justify-between text-[11px]">
        <span
          className={`font-semibold ${standingOwnerTextToneClass(timer.standing_type)}`}
        >
          Defenders {defenderPct}%
        </span>
        <span className="font-semibold text-rose-300">
          Attackers {attackersPct}%
        </span>
      </div>
      <div className="h-2 overflow-hidden rounded bg-slate-700/70">
        <div className="flex h-full">
          <div
            className={standingDefenderProgressClass(timer.standing_type)}
            style={defenderSegmentStyle}
            title={`Defenders ${defenderPct}%`}
          />
          <div
            className={attackersProgressClass}
            style={attackerSegmentStyle}
            title={`Attackers ${attackersPct}%`}
          />
        </div>
      </div>
    </div>
  );
}

function extractSovCampaignAttackersLabel(notes: string): string {
  const trimmed = notes.trim();
  if (trimmed.length === 0) {
    return "";
  }
  const prefix = "attackers:";
  if (!trimmed.toLowerCase().startsWith(prefix)) {
    return "";
  }
  return trimmed.slice(prefix.length).trim();
}

type TimerEntitySearchItem = {
  type: string;
  id: number;
  name: string;
  ticker: string;
};

type ParsedSovAttacker = {
  key: string;
  ticker: string;
  name: string;
  query: string;
  hostility: TimerStandingType | null;
};

const attackerAllianceIDCache = new Map<string, number>();

function parseSovCampaignAttackers(notes: string): ParsedSovAttacker[] {
  const label = extractSovCampaignAttackersLabel(notes);
  if (!label) return [];
  return label
    .split(",")
    .map((part) => parseSovCampaignAttackerPart(part))
    .filter((value): value is ParsedSovAttacker => value !== null);
}

function parseSovCampaignAttackerPart(
  rawPart: string,
): ParsedSovAttacker | null {
  const part = rawPart.trim();
  if (!part) return null;
  const withHostility = part.match(
    /^(.*?)(?:\s+\((ours|friendly|neutral|complicated|hostile)\))?$/i,
  );
  const core = withHostility?.[1]?.trim() ?? part;
  const hostilityRaw = withHostility?.[2]?.trim().toLowerCase();
  const hostility = hostilityRaw
    ? (normalizeTimerStanding(hostilityRaw) as TimerStandingType)
    : null;
  const match = core.match(/^\[(.+?)\]\s+(.+)$/);
  if (!match) {
    const name = core;
    const key = `${name.toLowerCase()}|`;
    return {
      key,
      ticker: "",
      name,
      query: name,
      hostility,
    };
  }
  const ticker = match[1]?.trim() ?? "";
  const name = match[2]?.trim() ?? "";
  if (!name) return null;
  const key = `${name.toLowerCase()}|${ticker.toLowerCase()}`;
  return {
    key,
    ticker,
    name,
    query: ticker ? `${name} ${ticker}` : name,
    hostility,
  };
}

function attackerToneClass(standing: TimerStandingType | null): string {
  if (!standing) {
    return "text-violet-300";
  }
  return standingOwnerTextToneClass(standing);
}

function SovAttackerChip({
  attacker,
  allianceID,
}: {
  attacker: ParsedSovAttacker;
  allianceID: number;
}) {
  const logo = useAllianceLogo(allianceID || undefined, 32);
  return (
    <span
      className={`inline-flex max-w-[14rem] items-center gap-1 truncate font-semibold ${attackerToneClass(attacker.hostility)}`}
      title={`${attacker.ticker ? `[${attacker.ticker}] ` : ""}${attacker.name}`}
    >
      {logo ? (
        <img
          src={logo}
          alt="Attacker alliance logo"
          className="h-3.5 w-3.5 rounded-[2px]"
          loading="lazy"
        />
      ) : null}
      <span className="truncate">
        {attacker.ticker ? `[${attacker.ticker}] ` : ""}
        {attacker.name}
      </span>
    </span>
  );
}

function clampPercent(value: number): number {
  const rounded = Math.round(Number.isFinite(value) ? value : 0);
  return Math.max(0, Math.min(100, rounded));
}

function mostFriendlyAttackerStanding(
  attackers: ParsedSovAttacker[],
): TimerStandingType | null {
  let best: TimerStandingType | null = null;
  let bestRank = -1;
  for (const attacker of attackers) {
    if (!attacker.hostility) continue;
    const rank = standingFriendlinessRank(attacker.hostility);
    if (rank > bestRank) {
      best = attacker.hostility;
      bestRank = rank;
    }
  }
  return best;
}

function standingFriendlinessRank(standing: TimerStandingType): number {
  switch (standing) {
    case TimerStandingType.Ours:
      return 5;
    case TimerStandingType.Friendly:
      return 4;
    case TimerStandingType.Neutral:
      return 3;
    case TimerStandingType.Complicated:
      return 2;
    case TimerStandingType.Hostile:
      return 1;
    default:
      return 0;
  }
}

function isCriticalSeverity(value: string): boolean {
  return value.trim().toLowerCase() === "critical";
}

function StyledDateText({ parts }: { parts: FormattedDateParts }) {
  return (
    <span className="timer-styled-date">
      <span>{parts.year}</span>
      <span className="text-success/85">-</span>
      <span>{parts.month}</span>
      <span className="text-success/85">-</span>
      <span>{parts.day}</span>
      <span className="timer-styled-date-divider">|</span>
      <span>{parts.hour}</span>
      <span className="text-success/85">:</span>
      <span>{parts.minute}</span>
      <span className="text-success/85">:</span>
      <span>{parts.second}</span>
      {parts.suffix ? (
        <span className="timer-styled-date-suffix">{parts.suffix}</span>
      ) : null}
      {parts.timezone ? (
        <span className="timer-styled-date-suffix">{parts.timezone}</span>
      ) : null}
    </span>
  );
}

function stageBadgeIcon(stage: TimerStageLabel) {
  switch (stage) {
    case "armor":
      return ShieldCheck;
    case "reinforcement":
      return ShieldAlert;
    case "hull":
      return ShieldAlert;
    case "initial_vulnerability":
      return AlertTriangle;
    case "anchoring":
      return Building2;
    case "unanchoring":
      return Wrench;
    case "extraction_window":
      return Moon;
    default:
      return null;
  }
}
