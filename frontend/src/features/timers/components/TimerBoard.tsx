import { useEffect, useMemo, useState } from "react";
import { api } from "@/config/api";
import { useUIStore } from "@/app/store/uiStore";
import { useAuthStore } from "@/app/store/authStore";
import useConfirm from "@/app/hooks/useConfirm";
import Panel from "@/components/Panel";
import ShadowedScrollArea from "@/components/ShadowedScrollArea";
import {
  useMapStore as useSharedMapStore,
  useRegionNames,
} from "@/features/map";
import TimerRow from "./TimerRow";
import TimerBoardFilters from "./TimerBoardFilters";
import TimerBoardHeaderActions from "./TimerBoardHeaderActions";
import {
  formatReplacement,
  formatUTCDateTime,
  formatStageLabel,
  formatStanding,
  formatStructureType,
  formatTicker,
  toUnixSeconds,
} from "../formatters";
import TimerRowsTicker from "./TimerRowsTicker";
import { severityByValue, timerKindLabels } from "../config/timerOptions";
import { useTimerModal } from "../hooks/useTimerModal";
import { useTimersStore } from "../store/useTimersStore";
import type { TimerRecord } from "../types";
import {
  TimerKind,
  TimerReplacementAction,
  TimerSeverity,
  TimerStructureType,
} from "../types";

export default function TimerBoard() {
  const setToast = useUIStore((s) => s.setToast);
  const canManage = useAuthStore((s) => s.isStaff || s.isAdmin);
  const requestConfirm = useConfirm();
  const systemsById = useSharedMapStore((s) => s.systems);
  const { getRegionName } = useRegionNames();
  const allTimers = useTimersStore((s) => s.timers);
  const loading = useTimersStore((s) => s.loading);
  const loadTimers = useTimersStore((s) => s.loadTimers);

  const [showInactive, setShowInactive] = useState(false);
  const [nowMs, setNowMs] = useState(() => Date.now());
  const [use24Hour, setUse24Hour] = useState(() => {
    if (typeof window === "undefined") return true;
    const stored = window.localStorage.getItem("timers:use24Hour");
    return stored ? stored === "true" : true;
  });

  const [searchQuery, setSearchQuery] = useState("");
  const [standingFilter, setStandingFilter] = useState<string[]>([]);
  const [kindFilter, setKindFilter] = useState<string[]>([]);
  const [structureFilter, setStructureFilter] = useState<string[]>([]);
  const [severityFilter, setSeverityFilter] = useState<string[]>([]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem("timers:use24Hour", String(use24Hour));
  }, [use24Hour]);

  useEffect(() => {
    const interval = window.setInterval(() => setNowMs(Date.now()), 15_000);
    return () => window.clearInterval(interval);
  }, []);

  const { openCreateModal, openEditModal } = useTimerModal({
    onSaved: () => loadTimers({ silent: true }),
  });

  const cancelTimer = async (timer: TimerRecord) => {
    try {
      await api.post(`/timers/${timer.id}/cancel`);
      setToast({ text: "Timer cancelled", color: "success" });
      await loadTimers({ silent: true });
    } catch {
      setToast({ text: "Failed to cancel timer", color: "error" });
    }
  };

  const uncancelTimer = async (timer: TimerRecord) => {
    try {
      await api.post(`/timers/${timer.id}/uncancel`);
      setToast({ text: "Timer reactivated", color: "success" });
      await loadTimers({ silent: true });
    } catch {
      setToast({ text: "Failed to reactivate timer", color: "error" });
    }
  };

  const deleteTimer = async (timer: TimerRecord) => {
    try {
      await api.delete(`/timers/${timer.id}`);
      setToast({ text: "Timer deleted", color: "success" });
      await loadTimers({ silent: true });
    } catch {
      setToast({ text: "Failed to delete timer", color: "error" });
    }
  };

  const confirmDeleteTimer = (timer: TimerRecord) => {
    requestConfirm({
      title: "Delete timer?",
      body: `This will permanently delete "${timer.title || timer.system_name}".`,
      confirmLabel: "Delete",
      cancelLabel: "Keep timer",
      tone: "danger",
      onConfirm: () => void deleteTimer(timer),
    });
  };

  const copyTimerMarkdown = async (
    timer: TimerRecord,
    systemName: string,
    regionName: string,
  ) => {
    const text = buildTimerMarkdown(timer, systemName, regionName);
    try {
      if (!navigator?.clipboard?.writeText) {
        throw new Error("clipboard unavailable");
      }
      await navigator.clipboard.writeText(text);
      setToast({ text: "Timer markdown copied", color: "success" });
    } catch {
      setToast({ text: "Failed to copy timer markdown", color: "error" });
    }
  };

  const sortedTimers = useMemo(
    () =>
      [...allTimers].sort(
        (a, b) => +new Date(a.expires_at) - +new Date(b.expires_at),
      ),
    [allTimers],
  );

  const filteredTimers = useMemo(() => {
    const needle = searchQuery.trim().toLowerCase();
    return sortedTimers
      .filter((timer) =>
        showInactive
          ? true
          : timer.status.toLowerCase() === "active" &&
            !isTimerExpired(timer, nowMs),
      )
      .filter((timer) => {
        const resolvedSystemName =
          systemsById[timer.system_id]?.name || timer.system_name;
        const resolvedRegionName =
          getRegionName(timer.region_id, timer.region_name) ||
          timer.region_name;
        if (
          standingFilter.length > 0 &&
          !standingFilter.includes(timer.standing_type)
        ) {
          return false;
        }
        if (kindFilter.length > 0 && !kindFilter.includes(timer.timer_kind)) {
          return false;
        }
        if (
          structureFilter.length > 0 &&
          !structureFilter.includes(timer.structure_type)
        ) {
          return false;
        }
        if (
          severityFilter.length > 0 &&
          !severityFilter.includes(timer.severity)
        ) {
          return false;
        }
        if (!needle) return true;
        const fields = [
          resolvedSystemName,
          resolvedRegionName,
          timer.title,
          timer.structure_type,
          timer.timer_kind,
          timer.stage_label,
          timer.replacement_action,
          timer.owner_corporation_name,
          timer.owner_corporation_ticker,
          timer.owner_alliance_name,
          timer.owner_alliance_ticker,
          timer.planet_name,
          timer.moon_name,
        ];
        return fields.some((value) => value.toLowerCase().includes(needle));
      });
  }, [
    kindFilter,
    searchQuery,
    severityFilter,
    showInactive,
    sortedTimers,
    standingFilter,
    structureFilter,
    systemsById,
    getRegionName,
    nowMs,
  ]);

  return (
    <Panel
      title="Timer Board"
      hint={
        canManage
          ? "Active timers are shown soonest first and mirrored on the map as indicators."
          : "You can view and submit timers. Editing/deleting are restricted to staff/admin."
      }
      actions={
        <TimerBoardHeaderActions
          showInactive={showInactive}
          setShowInactive={setShowInactive}
          use24Hour={use24Hour}
          setUse24Hour={setUse24Hour}
          onAddTimer={openCreateModal}
        />
      }
    >
      <TimerBoardFilters
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        standingFilter={standingFilter}
        setStandingFilter={setStandingFilter}
        kindFilter={kindFilter}
        setKindFilter={setKindFilter}
        structureFilter={structureFilter}
        setStructureFilter={setStructureFilter}
        severityFilter={severityFilter}
        setSeverityFilter={setSeverityFilter}
      />

      {loading ? (
        <div className="text-sm text-slate-600 dark:text-slate-500">
          Loading timers...
        </div>
      ) : filteredTimers.length === 0 ? (
        <div className="text-sm text-slate-600 dark:text-slate-500">
          No timers found.
        </div>
      ) : (
        <div className="h-[calc(100dvh-18rem)] min-h-[18rem] overflow-hidden">
          <TimerRowsTicker>
            {(nowMs) => (
              <ShadowedScrollArea scrollClassName="pr-1">
                <div className="space-y-2">
                  {filteredTimers.map((timer) => (
                    <TimerRow
                      key={timer.id}
                      timer={timer}
                      canManage={canManage}
                      use24Hour={use24Hour}
                      nowMs={nowMs}
                      systemName={
                        systemsById[timer.system_id]?.name || timer.system_name
                      }
                      regionName={
                        getRegionName(timer.region_id, timer.region_name) ||
                        timer.region_name
                      }
                      onEdit={openEditModal}
                      onCancel={cancelTimer}
                      onUncancel={uncancelTimer}
                      onCopy={copyTimerMarkdown}
                      onDeleteConfirm={confirmDeleteTimer}
                    />
                  ))}
                </div>
              </ShadowedScrollArea>
            )}
          </TimerRowsTicker>
        </div>
      )}
    </Panel>
  );
}

function isTimerExpired(timer: TimerRecord, nowMs: number): boolean {
  const expiresMs = Date.parse(timer.expires_at);
  if (!Number.isFinite(expiresMs)) return false;
  return expiresMs <= nowMs;
}

function buildTimerMarkdown(
  timer: TimerRecord,
  systemName: string,
  regionName: string,
): string {
  const structureLabel = formatStructureType(timer.structure_type);
  const stageLabel = formatStageLabel(timer.stage_label);
  const kindLabel = timerKindLabels[timer.timer_kind] || timer.timer_kind;
  const severityLabel =
    severityByValue.get(timer.severity as TimerSeverity)?.label || "Unknown";
  const standingLabel = formatStanding(timer.standing_type);
  const unix = toUnixSeconds(timer.expires_at);
  const title = timer.title?.trim() || `${systemName} Timer`;
  const ownerParts = [
    timer.owner_corporation_name
      ? `${formatTicker(timer.owner_corporation_ticker)} ${timer.owner_corporation_name}`.trim()
      : "",
    timer.owner_alliance_name
      ? `${formatTicker(timer.owner_alliance_ticker)} ${timer.owner_alliance_name}`.trim()
      : "",
  ].filter(Boolean);

  const lines = [
    `## ${title}`,
    `- System: **${systemName}** (${regionName})`,
    `- Priority: **${severityLabel}**`,
    `- Structure: **${structureLabel}**`,
    `- Stage: **${stageLabel}**`,
    `- Type: **${kindLabel}**`,
    `- Hostility: **${standingLabel}**`,
  ];
  if (
    timer.structure_type === TimerStructureType.OrbitalSkyhook &&
    timer.timer_kind === TimerKind.Extraction &&
    Number.isFinite(timer.skyhook_fullness_pct) &&
    timer.skyhook_fullness_pct > 0
  ) {
    lines.push(`- Skyhook Fullness: **${timer.skyhook_fullness_pct}%**`);
  }

  if (
    timer.replacement_action &&
    timer.replacement_action !== TimerReplacementAction.NotReplaceable
  ) {
    lines.push(
      `- Replacement: **${formatReplacement(timer.replacement_action)}**`,
    );
  }
  if (ownerParts.length > 0) {
    lines.push(`- Owner: ${ownerParts.join(" | ")}`);
  }
  if (unix !== null) {
    lines.push(`- Expires: <t:${unix}:R>`);
    lines.push(`- Local Time): <t:${unix}:F>`);
  } else {
    lines.push(`- Expires: ${timer.expires_at}`);
  }
  lines.push(`- EVE Time: ${formatUTCDateTime(timer.expires_at)}`);

  if (timer.notes?.trim()) {
    lines.push("");
    lines.push(`> ${timer.notes.trim()}`);
  }

  return lines.join("\n");
}
