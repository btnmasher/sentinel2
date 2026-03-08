import { useEffect, useMemo, useState } from "react";
import { api } from "@/config/api";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useUIStore } from "@/app/store/uiStore";
import { useAuthStore } from "@/app/store/authStore";
import useConfirm from "@/app/hooks/useConfirm";
import ListPagination from "@/components/ListPagination";
import Panel from "@/components/Panel";
import ShadowedScrollArea from "@/components/ShadowedScrollArea";
import { useListPagination } from "@/hooks/useListPagination";
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
import useTimerDebugTools from "../hooks/useTimerDebugTools";
import { useTimerModal } from "../hooks/useTimerModal";
import { useTimersStore } from "../store/useTimersStore";
import type { TimerRecord } from "../types";
import {
  TimerKind,
  TimerReplacementAction,
  TimerSeverity,
  TimerStatus,
  TimerStructureType,
} from "../types";

type TimerBoardTab =
  | "timers"
  | "sovereignty"
  | "logistics"
  | "mining"
  | "ratting";

export default function TimerBoard() {
  useTimerDebugTools();
  const setToast = useUIStore((s) => s.setToast);
  const canManage = useAuthStore((s) => s.isStaff || s.isAdmin);
  const timersReadOnly = useAppConfigStore((s) => s.timersReadOnly);
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
  const [activeTab, setActiveTab] = useState<TimerBoardTab>("timers");

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

  const activeTimers = useMemo(
    () =>
      sortedTimers.filter((timer) =>
        showInactive
          ? true
          : timer.status === TimerStatus.Active &&
            !isTimerExpired(timer, nowMs),
      ),
    [nowMs, showInactive, sortedTimers],
  );

  const tabCounts = useMemo(() => {
    let timers = 0;
    let sovereignty = 0;
    let logistics = 0;
    let mining = 0;
    let ratting = 0;
    for (const timer of activeTimers) {
      if (isSovereigntyTimer(timer)) {
        sovereignty++;
        continue;
      }
      if (isLogisticsTimer(timer)) {
        logistics++;
        continue;
      }
      if (isMiningTimer(timer)) {
        mining++;
        continue;
      }
      if (isRattingTimer(timer)) {
        ratting++;
        continue;
      }
      timers++;
    }
    return { timers, sovereignty, logistics, mining, ratting };
  }, [activeTimers]);

  const tabTimers = useMemo(() => {
    switch (activeTab) {
      case "sovereignty":
        return activeTimers.filter(isSovereigntyTimer);
      case "logistics":
        return activeTimers.filter(isLogisticsTimer);
      case "mining":
        return activeTimers.filter(isMiningTimer);
      case "ratting":
        return activeTimers.filter(isRattingTimer);
      case "timers":
      default:
        return activeTimers.filter((timer) => !isSpecializedTimer(timer));
    }
  }, [activeTab, activeTimers]);

  const filteredTimers = useMemo(() => {
    if (activeTab !== "timers") {
      return tabTimers;
    }
    const needle = searchQuery.trim().toLowerCase();
    return tabTimers.filter((timer) => {
      const resolvedSystemName =
        systemsById[timer.system_id]?.name || timer.system_name;
      const resolvedRegionName =
        getRegionName(timer.region_id, timer.region_name) || timer.region_name;
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
    activeTab,
    kindFilter,
    searchQuery,
    severityFilter,
    standingFilter,
    structureFilter,
    tabTimers,
    systemsById,
    getRegionName,
  ]);

  const {
    pageSize,
    setPageSize,
    pageIndex,
    setPageIndex,
    pagedItems: pagedTimers,
    showPaginationControls: showPaginationRow,
  } = useListPagination({
    items: filteredTimers,
    initialPageSize: 50,
    minItemsToShowControls: 25,
    resetDeps: [
      activeTab,
      searchQuery,
      standingFilter,
      kindFilter,
      structureFilter,
      severityFilter,
      showInactive,
    ],
  });

  const tabs: Array<{ id: TimerBoardTab; label: string; count: number }> = [
    { id: "timers", label: "Timers", count: tabCounts.timers },
    { id: "sovereignty", label: "Sovereignty", count: tabCounts.sovereignty },
    { id: "logistics", label: "Logistics", count: tabCounts.logistics },
    { id: "mining", label: "Mining", count: tabCounts.mining },
    { id: "ratting", label: "Ratting", count: tabCounts.ratting },
  ];

  return (
    <Panel
      className="h-[calc(100dvh-8.5rem)] min-h-[32rem] overflow-hidden"
      bodyClassName="!space-y-0 h-full min-h-0 grid grid-rows-[auto_minmax(0,1fr)] gap-4"
      title="Timer Board"
      actions={
        <TimerBoardHeaderActions
          showInactive={showInactive}
          setShowInactive={setShowInactive}
          use24Hour={use24Hour}
          setUse24Hour={setUse24Hour}
          onAddTimer={openCreateModal}
          readOnly={timersReadOnly}
        />
      }
    >
      <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)]">
        <div className="timer-board-tab-row">
          {tabs.map((tab) => {
            const active = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={`timer-board-tab-btn ${active ? "is-active btn btn-sm btn-primary" : "is-inactive"}`}
              >
                <span>{tab.label}</span>
                <span
                  className={`timer-board-tab-count ${active ? "is-active" : "is-inactive"}`}
                >
                  {tab.count}
                </span>
              </button>
            );
          })}
        </div>

        <div className="timer-board-tab-pane">
          {activeTab === "timers" ? (
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
          ) : null}

          {loading ? (
            <div className="text-sm text-slate-600 dark:text-slate-500">
              Loading timers...
            </div>
          ) : filteredTimers.length === 0 ? (
            <div className="flex min-h-[16rem] items-center justify-center text-center text-lg font-medium text-slate-500">
              No {activeTab === "timers" ? "" : `${activeTab} `}timers found.
            </div>
          ) : (
            <div
              className={
                showPaginationRow
                  ? "min-h-0 flex-1 overflow-hidden grid grid-rows-[auto_minmax(0,1fr)] gap-2"
                  : "min-h-0 flex-1 overflow-hidden"
              }
            >
              {showPaginationRow ? (
                <ListPagination
                  totalItems={filteredTimers.length}
                  pageSize={pageSize}
                  pageIndex={pageIndex}
                  onPageSizeChange={setPageSize}
                  onPageChange={setPageIndex}
                  minItemsToShow={25}
                />
              ) : null}
              <div className="min-h-0 overflow-hidden">
                <TimerRowsTicker>
                  {(nowMs) => (
                    <ShadowedScrollArea scrollClassName="pr-1">
                      <div className="space-y-2">
                        {pagedTimers.map((timer) => (
                          <TimerRow
                            key={timer.id}
                            timer={timer}
                            canManage={canManage}
                            readOnly={timersReadOnly}
                            use24Hour={use24Hour}
                            nowMs={nowMs}
                            systemName={
                              systemsById[timer.system_id]?.name ||
                              timer.system_name
                            }
                            regionName={
                              getRegionName(
                                timer.region_id,
                                timer.region_name,
                              ) || timer.region_name
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
            </div>
          )}
        </div>
      </div>
    </Panel>
  );
}

function isSovereigntyTimer(timer: TimerRecord): boolean {
  if (timer.structure_type === TimerStructureType.SovereigntyHub) {
    return true;
  }
  return (
    timer.structure_type === TimerStructureType.OrbitalSkyhook &&
    timer.timer_kind === TimerKind.Reinforcement
  );
}

function isLogisticsTimer(timer: TimerRecord): boolean {
  return (
    timer.structure_type === TimerStructureType.OrbitalSkyhook &&
    timer.timer_kind === TimerKind.Extraction
  );
}

function isMiningTimer(timer: TimerRecord): boolean {
  const isMiningStructure =
    timer.structure_type === TimerStructureType.UpwellRefineryTatara ||
    timer.structure_type === TimerStructureType.UpwellRefineryAthanor ||
    timer.structure_type === TimerStructureType.MetenoxMoonDrill;
  if (!isMiningStructure) {
    return false;
  }
  return timer.timer_kind === TimerKind.Extraction;
}

function isRattingTimer(timer: TimerRecord): boolean {
  return timer.structure_type === TimerStructureType.MercenaryDen;
}

function isSpecializedTimer(timer: TimerRecord): boolean {
  return (
    isSovereigntyTimer(timer) ||
    isLogisticsTimer(timer) ||
    isMiningTimer(timer) ||
    isRattingTimer(timer)
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
