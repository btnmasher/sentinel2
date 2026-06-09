import { useEffect, useMemo, useState } from "react";
import type { IntelReport } from "../types";
import { Swords } from "lucide-react";
import {
  colorForAge,
  useRegionNames,
  useMapStore,
  useOpenSystemContextMenu,
} from "@/features/map";
import { useAllianceLogo, useCorporationLogo } from "@/hooks/useEveImage";
import { useUIStore } from "@/features/ui";
import {
  getZKillIntelMeta,
  isClearIntelReport,
  splitIntelReportTextBySystems,
} from "../utils/intelReportUtils";
import { useSettingsStore } from "@/app/store/settingsStore";
import HoverCard from "@/components/HoverCard";
import {
  hostilityRowToneClass,
  normalizeStanding,
  standingOwnerTextToneClass,
  StandingType,
} from "@/features/shared";

type SplitTextChunk = string | { kind: "system"; text: string; systemId: number };

function timeSuffix(minutes: number) {
  if (minutes <= 0) return "now";
  if (minutes > 60) return `${Math.floor(minutes / 60)}h`;
  return `${minutes}m`;
}

export default function ReportItem({
  log,
  channelNames,
}: {
  log: IntelReport;
  channelNames: Record<string, string>;
}) {
  const mapRegions = useMapStore((s) => s.mapRegions);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const setContextMenu = useUIStore((s) => s.setContextMenu);
  const openSystemContextMenu = useOpenSystemContextMenu();
  const { getRegionName } = useRegionNames();
  const threatTimings = useSettingsStore((s) => s.settings.intel.threatTimings);

  const [timePassed, setTimePassed] = useState(0);

  useEffect(() => {
    const intelAge = Math.max(
      0,
      Math.floor((Date.now() - log.time * 1000) / 60000),
    );
    setTimePassed(intelAge);
    const timer = setInterval(() => {
      setTimePassed((prev) => prev + 1);
    }, 60000);
    return () => clearInterval(timer);
  }, [log.time]);

  const timeColor = useMemo(
    () => colorForAge(timePassed * 60, threatTimings),
    [threatTimings, timePassed],
  );

  const reportSystemsById = useMemo(
    () => new Map(log.systems.map((system) => [system.system, system])),
    [log.systems],
  );

  const splitText = useMemo<SplitTextChunk[]>(
    () => splitIntelReportTextBySystems(log.text, log.systems),
    [log.text, log.systems],
  );

  const regionIds = useMemo(
    () => new Set(mapRegions.map((r) => String(r))),
    [mapRegions],
  );

  const timestamp = useMemo(() => {
    const date = new Date(log.time * 1000);
    return `${date.getHours()}:${date.getMinutes().toString().padStart(2, "0")}`;
  }, [log.time]);

  const localTimestamp = useMemo(() => {
    const date = new Date(log.time * 1000);
    return date.toLocaleString();
  }, [log.time]);

  const reportChannel = useMemo(() => {
    if (!log.channel_id) return "Unknown";
    return channelNames[log.channel_id] ?? `ID: ${log.channel_id}`;
  }, [channelNames, log.channel_id]);

  const isClearReport = useMemo(() => isClearIntelReport(log), [log]);
  const zkillMeta = useMemo(() => getZKillIntelMeta(log), [log]);
  const zkillRenderHostility = useMemo(() => {
    if (!zkillMeta) return "neutral";
    const victimHostility = normalizeStanding(zkillMeta.zkill.victim_hostility);
    if (
      victimHostility === StandingType.Ours ||
      victimHostility === StandingType.Friendly
    ) {
      return StandingType.Hostile;
    }
    return normalizeStanding(zkillMeta.zkill.killer_hostility);
  }, [zkillMeta]);
  const rowToneClass = useMemo(() => {
    if (!zkillMeta) return "border-slate-800 bg-base-300/40";
    return hostilityRowToneClass(zkillRenderHostility);
  }, [zkillMeta, zkillRenderHostility]);
  const killerNameToneClass = useMemo(() => {
    if (!zkillMeta) return "";
    const standing = normalizeStanding(zkillMeta.zkill.killer_hostility);
    if (standing === StandingType.Neutral) {
      return "text-slate-200";
    }
    return standingOwnerTextToneClass(standing);
  }, [zkillMeta]);
  const victimNameToneClass = useMemo(() => {
    if (!zkillMeta) return "";
    const standing = normalizeStanding(zkillMeta.zkill.victim_hostility);
    if (standing === StandingType.Neutral) {
      return "text-slate-200";
    }
    return standingOwnerTextToneClass(standing);
  }, [zkillMeta, zkillRenderHostility]);
  const killerAllianceLogo = useAllianceLogo(
    zkillMeta?.zkill.killer_alliance_id || undefined,
    32,
  );
  const killerCorpLogo = useCorporationLogo(
    zkillMeta?.zkill.killer_corporation_id || undefined,
    32,
  );
  const victimAllianceLogo = useAllianceLogo(
    zkillMeta?.zkill.victim_alliance_id || undefined,
    32,
  );
  const victimCorpLogo = useCorporationLogo(
    zkillMeta?.zkill.victim_corporation_id || undefined,
    32,
  );

  const openCharacterSearchMenu = (
    event: React.MouseEvent,
    rawText: string,
  ) => {
    event.preventDefault();
    event.stopPropagation();
    const rect = event.currentTarget.getBoundingClientRect();
    const selectedText = window.getSelection()?.toString() ?? "";
    const sourceText = selectedText.trim() ? selectedText : rawText;
    const normalized = sourceText
      .trim()
      .replace(/\s+/g, " ")
      .replace(/^[^\w'-]+|[^\w'-]+$/g, "");
    if (!normalized) {
      return;
    }
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      anchorRect: {
        left: rect.left,
        top: rect.top,
        width: rect.width,
        height: rect.height,
      },
      type: "character-search",
      text: normalized,
    });
  };

  const showMenu = (event: React.MouseEvent) => {
    event.preventDefault();
    const rect = event.currentTarget.getBoundingClientRect();
    const selection = window.getSelection()?.toString();
    if (selection) {
      setContextMenu({
        x: event.clientX,
        y: event.clientY,
        anchorRect: {
          left: rect.left,
          top: rect.top,
          width: rect.width,
          height: rect.height,
        },
        type: "character-search",
        text: selection,
      });
    } else {
      setContextMenu({
        x: event.clientX,
        y: event.clientY,
        anchorRect: {
          left: rect.left,
          top: rect.top,
          width: rect.width,
          height: rect.height,
        },
        type: "text",
        text: "Try selecting a name from intel",
      });
    }
  };

  return (
    <article
      className={`border rounded-lg p-3 ${rowToneClass}`}
      onContextMenu={zkillMeta ? undefined : showMenu}
    >
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span
          className="report-item-relative-time"
          style={{ color: timeColor }}
        >
          {timeSuffix(timePassed)}
        </span>
        <div className="flex items-center gap-2">
          {isClearReport && (
            <span className="report-item-cleared-badge rounded border border-emerald-500/50 bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-emerald-300">
              Cleared
            </span>
          )}
          <HoverCard
            trigger={
              <span className="report-item-time cursor-help" tabIndex={0}>
                {timestamp}
              </span>
            }
            className="hover-card-surface report-time-hover-card"
          >
            <span>Local: {localTimestamp}</span>
            <span>Channel: {reportChannel}</span>
          </HoverCard>
        </div>
      </div>
      <p className="text-xs text-slate-200 mt-1 leading-snug">
        {zkillMeta ? (
          <span className="inline-flex flex-col items-start gap-y-0.5 leading-tight">
            <span>
              {(() => {
                const system = log.systems[0];
                const fallbackName = zkillMeta.zkill.system_name;
                if (!system) {
                  return <span className="text-sm">{fallbackName}</span>;
                }
                if (system.region >= 11000000) {
                  return (
                    <span className="text-sm">
                      {system.name || fallbackName}
                    </span>
                  );
                }
                const regionId = String(system.region);
                if (!regionIds.has(regionId)) {
                  const regionName = getRegionName(
                    regionId,
                    `Region ${regionId}`,
                  );
                  return (
                    <button
                      className="report-item-unloaded-region text-sm"
                      title={`Click to load ${regionName}`}
                      onClick={() =>
                        updateMapConfig({
                          mapRegions: [...mapRegions, regionId],
                        })
                      }
                    >
                      {system.name || fallbackName}
                    </button>
                  );
                }
                return (
                  <button
                    className="report-item-system-link text-sm"
                    onClick={() => setSystemSearch(system.system)}
                  >
                    {system.name || fallbackName}
                  </button>
                );
              })()}
            </span>
            <a
              href={zkillMeta.zkill.url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex flex-wrap items-center gap-x-1.5 gap-y-0.5 rounded px-1 py-0.5 font-semibold text-slate-200 transition-colors hover:bg-slate-500/15"
            >
              <span
                className={`inline-flex items-center gap-1 ${killerNameToneClass}`}
              >
                {(killerAllianceLogo || killerCorpLogo) && (
                  <img
                    src={killerAllianceLogo || killerCorpLogo}
                    alt="Killer organization logo"
                    className="h-3.5 w-3.5 rounded-[2px]"
                    loading="lazy"
                  />
                )}
                <span>{zkillMeta.zkill.killer_name || "Unknown Killer"}</span>
              </span>
              <span className="whitespace-nowrap">
                {zkillMeta.zkill.involved_attackers > 0
                  ? `(+${zkillMeta.zkill.involved_attackers})`
                  : "(Solo)"}
              </span>
              <span className="inline-flex items-center text-slate-300">
                <Swords className="h-3.5 w-3.5" />
              </span>
              <span
                className={`inline-flex items-center gap-1 ${victimNameToneClass}`}
              >
                {(victimAllianceLogo || victimCorpLogo) && (
                  <img
                    src={victimAllianceLogo || victimCorpLogo}
                    alt="Victim organization logo"
                    className="h-3.5 w-3.5 rounded-[2px]"
                    loading="lazy"
                  />
                )}
                <span>{zkillMeta.zkill.victim_name || "Unknown Victim"}</span>
              </span>
              <span className="whitespace-nowrap">
                ({zkillMeta.zkill.victim_ship_name || "Unknown Ship"})
              </span>
            </a>
          </span>
        ) : (
          splitText.map((chunk, idx) => {
            if (typeof chunk === "string") {
              return (
                <span
                  key={idx}
                  onContextMenu={(event) =>
                    openCharacterSearchMenu(event, chunk)
                  }
                >
                  {chunk}
                </span>
              );
            }
            const system = reportSystemsById.get(chunk.systemId);
            if (!system) {
              return <span key={idx}>{chunk.text}</span>;
            }
            if (system.region >= 11000000) {
              return (
                <span key={idx} className="text-sm">
                  {chunk.text}
                </span>
              );
            }
            const regionId = String(system.region);
            if (!regionIds.has(regionId)) {
              const regionName = getRegionName(regionId, `Region ${regionId}`);
              return (
                <span key={idx}>
                  <button
                    className="report-item-unloaded-region text-sm"
                    title={`Click to load ${regionName}`}
                    onClick={() =>
                      updateMapConfig({ mapRegions: [...mapRegions, regionId] })
                    }
                    onContextMenu={(event) =>
                      openSystemContextMenu(
                        event,
                        system.system,
                        `Load ${regionName} to open system menu`,
                      )
                    }
                  >
                    {chunk.text}
                  </button>
                </span>
              );
            }
            return (
              <span key={idx}>
                <button
                  className="report-item-system-link text-sm"
                  onClick={() => setSystemSearch(system.system)}
                  onContextMenu={(event) =>
                    openSystemContextMenu(
                      event,
                      system.system,
                      "System menu unavailable",
                    )
                  }
                >
                  {chunk.text}
                </button>
              </span>
            );
          })
        )}
      </p>
      <div
        className="text-xs text-slate-500 mt-2"
        onContextMenu={(event) => openCharacterSearchMenu(event, log.author)}
      >
        {log.author}
      </div>
    </article>
  );
}
