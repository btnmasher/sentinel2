import { useIntelStore } from "@/features/intel";
import { IntelPanel } from "@/features/intel";
import {
  ContextMenu,
  MapLayoutSelect,
  MapPageShell,
  JumpbridgesToggle,
  MapZoomControls,
  MapCanvas,
  RegionSelect,
} from "@/features/map";
import NavbarSearch from "@/components/NavbarSearch";
import { useUIStore } from "@/app/store/uiStore";
import { useSettingsStore } from "@/app/store/settingsStore";
import IntelServerStatus from "./IntelServerStatus";
import {
  Activity,
  ChevronDown,
  ChevronUp,
  ScrollText,
  Upload,
} from "lucide-react";

export default function Intel() {
  const reports = useIntelStore((state) => state.reports);
  const uploaders = useIntelStore((state) => state.uploaders);
  const version = useIntelStore((state) => state.version);
  const setDialog = useUIStore((s) => s.setDialog);
  const mapViewMode = useSettingsStore((s) => s.settings.map.viewMode);
  const intelPanelOpen = useSettingsStore((s) => s.settings.intel.panelOpen);
  const applySetting = useSettingsStore((s) => s.apply);

  const leftControls = (
    <>
      <NavbarSearch />
      <RegionSelect multi />
      <MapLayoutSelect inlineLabel="Layout" />
      <JumpbridgesToggle />
      <MapZoomControls />
      <button
        className="btn btn-xs btn-info btn-outline"
        onClick={() => setDialog("shareLink", true)}
      >
        Share
      </button>
      <button
        className="btn btn-xs btn-info btn-outline"
        onClick={() => setDialog("help", true)}
      >
        Help
      </button>
    </>
  );

  const rightControls = (
    <>
      <IntelServerStatus />
      <span className="px-2 py-1 rounded border border-slate-700 bg-base-300/70">
        Version: {version || "-"}
      </span>
      <span className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content">
        <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-base-100/80 text-base-content">
          <Upload className="h-3.5 w-3.5" />
        </span>
        <span>{uploaders}</span>
      </span>
      <span className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content">
        <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-base-100/80 text-base-content">
          <ScrollText className="h-3.5 w-3.5" />
        </span>
        <span>{reports.length}</span>
      </span>
      {mapViewMode === "full" && (
        <button
          className="btn btn-xs btn-ghost"
          onClick={() => applySetting("intel", "panelOpen", !intelPanelOpen)}
          aria-label="Toggle intel panel"
        >
          {intelPanelOpen ? (
            <ChevronUp className="h-5 w-5" />
          ) : (
            <ChevronDown className="h-5 w-5" />
          )}
        </button>
      )}
    </>
  );

  return (
    <MapPageShell
      pageBadge={{ icon: <Activity className="h-3 w-3" />, label: "Intel" }}
      leftControls={leftControls}
      rightControls={rightControls}
      panel={<IntelPanel />}
      panelOpen={intelPanelOpen}
      panelClassName="w-96"
    >
      <MapCanvas />
      <ContextMenu />
    </MapPageShell>
  );
}
