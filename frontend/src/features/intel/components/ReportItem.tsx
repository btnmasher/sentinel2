import { useEffect, useMemo, useState } from "react";
import type { IntelReport } from "../types";
import {
  REGION_MAP,
  colorForAge,
  useMapStore,
  useOpenSystemContextMenu,
} from "@/features/map";
import { useUIStore } from "@/features/ui";
import { isClearIntelReport } from "../utils/intelReportUtils";
import { useSettingsStore } from "@/app/store/settingsStore";

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
  const systems = useMapStore((s) => s.systems);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const setContextMenu = useUIStore((s) => s.setContextMenu);
  const openSystemContextMenu = useOpenSystemContextMenu();
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

  const splitText = useMemo(() => {
    const words = log.text.split(" ");
    return words.map((word) => {
      if (word.length < 3) return word;
      const cleaned = word.replace("*", "").toLowerCase();
      const matches = log.systems.filter((system) =>
        system.name.toLowerCase().startsWith(cleaned),
      );
      if (matches.length === 0) return word;
      if (matches.length === 1) return { text: word, system: matches[0] };
      const loaded = matches
        .map((system) => systems[system.system])
        .filter(Boolean);
      if (loaded.length === 1) return { text: word, system: loaded[0] };
      return { text: word, tooltip: "Multiple systems returned" };
    });
  }, [log.systems, log.text, systems]);

  const regionIds = useMemo(
    () => new Set(mapRegions.map((r) => String(r))),
    [mapRegions],
  );

  const timestamp = useMemo(() => {
    const date = new Date(log.time * 1000);
    return `${date.getHours()}:${date.getMinutes().toString().padStart(2, "0")}`;
  }, [log.time]);

  const channelTooltip = useMemo(() => {
    if (!log.channel_id) return "";
    const channelName = channelNames[log.channel_id];
    if (channelName) return `Channel: ${channelName}`;
    return `Channel ID: ${log.channel_id}`;
  }, [channelNames, log.channel_id]);

  const isClearReport = useMemo(() => isClearIntelReport(log), [log]);

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
      className="border border-slate-800 rounded-lg p-3 bg-base-300/40"
      onContextMenu={showMenu}
      title={channelTooltip}
    >
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span style={{ color: timeColor }}>{timeSuffix(timePassed)}</span>
        <div className="flex items-center gap-2">
          {isClearReport && (
            <span className="rounded border border-emerald-500/50 bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-emerald-300">
              Cleared
            </span>
          )}
          <span>{timestamp}</span>
        </div>
      </div>
      <p className="text-sm text-slate-200 mt-1">
        {splitText.map((chunk, idx) => {
          if (typeof chunk === "string") {
            return (
              <span key={idx}>
                {idx > 0 ? " " : ""}
                {chunk}
              </span>
            );
          }
          if ("tooltip" in chunk) {
            return (
              <span key={idx}>
                {idx > 0 ? " " : ""}
                <span className="text-amber-300" title={chunk.tooltip}>
                  {chunk.text}
                </span>
              </span>
            );
          }
          const system = chunk.system;
          if (!system) {
            return (
              <span key={idx}>
                {idx > 0 ? " " : ""}
                {chunk.text}
              </span>
            );
          }
          if (system.region >= 11000000) {
            return (
              <span key={idx}>
                {idx > 0 ? " " : ""}
                {chunk.text}
              </span>
            );
          }
          const regionId = String(system.region);
          if (!regionIds.has(regionId)) {
            const regionName = REGION_MAP[regionId] || `Region ${regionId}`;
            return (
              <span key={idx}>
                {idx > 0 ? " " : ""}
                <button
                  className="report-item-unloaded-region"
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
              {idx > 0 ? " " : ""}
              <button
                className="text-sky-300"
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
        })}
      </p>
      <div className="text-xs text-slate-500 mt-2">{log.author}</div>
    </article>
  );
}
