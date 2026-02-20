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
  ShieldAlert,
  ShieldCheck,
  SquarePen,
  Trash2,
  Wrench,
  Building2,
} from "lucide-react";
import { useAllianceLogo, useCorporationLogo } from "@/hooks/useEveImage";
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
  TimerStageLabel,
  TimerStandingType,
  TimerStatus,
  TimerStructureType,
  type TimerRecord,
} from "../types";

type TimerRowProps = {
  timer: TimerRecord;
  canManage: boolean;
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
    <div className={`rounded-xl border bg-base-300/30 p-3 ${rowToneClass}`}>
      <div className="grid grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_auto] gap-x-3 gap-y-3">
        <TimerRowCountdown
          countdownClass={countdownClass}
          countdownText={countdownText}
          cancelled={cancelled}
          expired={expired}
          staticDateParts={staticDateParts}
          expiresAt={timer.expires_at}
        />
        <TimerRowActions
          timer={timer}
          canManage={canManage}
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
          severityIcon={SeverityIcon}
          severityLabel={severityMeta?.label || timer.severity}
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
};

function TimerRowCountdown({
  countdownClass,
  countdownText,
  cancelled,
  expired,
  staticDateParts,
  expiresAt,
}: TimerRowCountdownProps) {
  return (
    <div className="min-w-0">
      <div className="flex items-end gap-2">
        <div
          className={`font-mono text-2xl font-bold tracking-wide ${countdownClass}`}
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
  systemName,
  regionName,
  onCopy,
  onEdit,
  onCancel,
  onUncancel,
  onDeleteConfirm,
}: TimerRowActionsProps) {
  const isExpired = Date.parse(timer.expires_at) <= Date.now();
  return (
    <div className="flex gap-2 justify-self-end self-start">
      <button
        className="btn btn-xs btn-outline"
        onClick={() => onCopy(timer, systemName, regionName)}
      >
        <Copy className="h-3 w-3" />
        Copy
      </button>
      {canManage && (
        <button
          className="btn btn-xs btn-outline"
          onClick={() => onEdit(timer)}
        >
          <SquarePen className="h-3 w-3" />
          Edit
        </button>
      )}
      {canManage && timer.status === TimerStatus.Active && (
        <button
          className="btn btn-xs btn-outline"
          onClick={() => onCancel(timer)}
        >
          Cancel
        </button>
      )}
      {canManage && timer.status === TimerStatus.Canceled && !isExpired && (
        <button
          className="btn btn-xs btn-outline"
          onClick={() => onUncancel(timer)}
        >
          Un-cancel
        </button>
      )}
      {canManage && (
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
  return (
    <div className="min-w-0 self-stretch flex flex-col">
      <div>
        <div className="flex items-center gap-2 font-semibold">
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
        {timer.notes?.trim() ? (
          <div className="timer-note-text mt-1 line-clamp-2">{timer.notes}</div>
        ) : null}
      </div>
      {(timer.owner_corporation_name || timer.owner_alliance_name) && (
        <div className="mt-auto pt-2 flex flex-wrap items-center gap-2 text-xs text-slate-700 dark:text-slate-300">
          {timer.owner_corporation_name ? (
            <span
              className={`inline-flex items-center gap-1.5 font-semibold ${ownerTextToneClass(timer.standing_type)}`}
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
              className={`inline-flex items-center gap-1.5 font-semibold ${ownerTextToneClass(timer.standing_type)}`}
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
      )}
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
  severityIcon: ComponentType<{ className?: string }>;
  severityLabel: string;
};

function TimerRowBadges({
  timer,
  structureTone,
  structureIcon: StructureIcon,
  stageIcon: StageIcon,
  standingIcon: StandingIcon,
  replacementIcon: ReplacementIcon,
  severityIcon: SeverityIcon,
  severityLabel,
}: TimerRowBadgesProps) {
  return (
    <div className="flex max-w-[36rem] flex-wrap items-center justify-end gap-2 self-end justify-self-end text-xs text-slate-700 dark:text-slate-300">
      <span
        className={`badge timer-row-badge ${severityBadgeClass(timer.severity)}`}
      >
        <SeverityIcon className="h-3 w-3" />
        {severityLabel}
      </span>
      <span
        className={`badge timer-row-badge ${standingBadgeClass(timer.standing_type)}`}
      >
        <StandingIcon className="h-3 w-3" />{" "}
        {formatStanding(timer.standing_type)}
      </span>
      <span
        className={`badge timer-row-badge ${structureBadgeClassByTone(structureTone)}`}
      >
        <StructureIcon className="h-3 w-3" />{" "}
        {formatStructureType(timer.structure_type)}
      </span>
      {showSkyhookFullnessBadge(timer) ? (
        <span className="badge timer-row-badge timer-skyhook-fullness-badge">
          <Droplets className="h-3 w-3" />{" "}
          {Math.round(timer.skyhook_fullness_pct)}% Full
        </span>
      ) : null}
      <span
        className={`badge timer-row-badge ${stageBadgeClass(timer.stage_label)}`}
      >
        <StageIcon className="h-3 w-3" /> {formatStageLabel(timer.stage_label)}
      </span>
      {timer.replacement_action &&
      timer.replacement_action !== TimerReplacementAction.NotReplaceable ? (
        <span
          className={`badge timer-row-badge ${replacementBadgeClass(timer.replacement_action)}`}
        >
          <ReplacementIcon className="h-3 w-3" />{" "}
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

function ownerTextToneClass(standing: TimerStandingType): string {
  switch (standing) {
    case "ours":
      return "timer-owner-tone-ours";
    case "friendly":
      return "timer-owner-tone-friendly";
    case "complicated":
      return "timer-owner-tone-complicated";
    case "hostile":
      return "timer-owner-tone-hostile";
    case "neutral":
    default:
      return "timer-owner-tone-neutral";
  }
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
    case "structure":
      return ShieldAlert;
    case "initial_vulnerability":
      return AlertTriangle;
    case "anchoring":
      return Building2;
    case "unanchoring":
      return Wrench;
    case "extraction_window":
    case "pickup_window":
      return Moon;
    default:
      return null;
  }
}
