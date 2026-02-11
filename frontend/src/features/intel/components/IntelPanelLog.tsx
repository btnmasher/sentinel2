import { useEffect, useMemo, useState } from "react";
import type { IntelReport } from "../types";
import { REGION_MAP, useMapStore } from "@/features/map";
import { useUIStore } from "@/features/ui";
import { LOG_COLORS } from "@/utils/logColors";

function timeSuffix(minutes: number) {
  if (minutes === 0) return "new";
  if (minutes > 60) return `${Math.floor(minutes / 60)}h`;
  return `${minutes}m`;
}

export default function IntelPanelLog({ log }: { log: IntelReport }) {
  const systems = useMapStore((s) => s.systems);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const setContextMenu = useUIStore((s) => s.setContextMenu);

  const [timePassed, setTimePassed] = useState(0);

  useEffect(() => {
    const intelAge = Math.floor((Date.now() - log.time * 1000) / 60000);
    setTimePassed(intelAge);
    const timer = setInterval(() => {
      setTimePassed((prev) => prev + 1);
    }, 60000);
    return () => clearInterval(timer);
  }, [log.time]);

  const timeColor = useMemo(() => {
    return (
      LOG_COLORS.find((color) => timePassed >= color.minutes) ?? LOG_COLORS[0]
    );
  }, [timePassed]);

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
    >
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span style={{ color: timeColor.color }}>{timeSuffix(timePassed)}</span>
        <span>{timestamp}</span>
      </div>
      <p className="text-sm text-slate-200 mt-1">
        {splitText.map((chunk, idx) => {
          if (typeof chunk === "string") {
            return <span key={idx}>{chunk} </span>;
          }
          if ("tooltip" in chunk) {
            return (
              <span
                key={idx}
                className="tooltip tooltip-bottom text-amber-300"
                data-tip={chunk.tooltip}
              >
                {chunk.text}
              </span>
            );
          }
          const system = chunk.system as any;
          if (!system) return <span key={idx}>{chunk.text} </span>;
          if (system.region >= 11000000) {
            return <span key={idx}>{chunk.text} </span>;
          }
          const regionId = String(system.region);
          if (!regionIds.has(regionId)) {
            const regionName = REGION_MAP[regionId] || `Region ${regionId}`;
            return (
              <button
                key={idx}
                className="tooltip tooltip-bottom text-fuchsia-300"
                data-tip={`Click to load ${regionName}`}
                onClick={() =>
                  updateMapConfig({ mapRegions: [...mapRegions, regionId] })
                }
              >
                {chunk.text}
              </button>
            );
          }
          return (
            <button
              key={idx}
              className="text-sky-300"
              onClick={() => setSystemSearch(system.system)}
            >
              {chunk.text}
            </button>
          );
        })}
      </p>
      <div className="text-xs text-slate-500 mt-2">{log.author}</div>
    </article>
  );
}
