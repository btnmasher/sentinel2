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
import { UI_DIALOG } from "@/app/store/uiStore";
import { useAppModal } from "@/components/dialogs/AppModals";
import { useSettingsStore } from "@/app/store/settingsStore";
import IntelServerStatus from "./IntelServerStatus";
import UploaderCountBadge from "./UploaderCountBadge";
import ReportHealthBadge from "./ReportHealthBadge";
import { Activity, ChevronDown, ChevronUp } from "lucide-react";
import AlarmMuteToggleButton from "@/components/AlarmMuteToggleButton";
import useIntelDebugTools from "../hooks/useIntelDebugTools";

export default function Intel() {
  const { open: openHelpModal } = useAppModal(UI_DIALOG.Help);
  const { open: openShareLinkModal } = useAppModal(UI_DIALOG.ShareLink);
  const mapViewMode = useSettingsStore((s) => s.settings.map.viewMode);
  const intelPanelOpen = useSettingsStore((s) => s.settings.intel.panelOpen);
  const alarmEnabled = useSettingsStore((s) => s.settings.alarm.enabled);
  const alarmVolume = useSettingsStore((s) => s.settings.alarm.volume);
  const applySetting = useSettingsStore((s) => s.apply);
  const alarmMuted = !alarmEnabled || alarmVolume <= 0;
  useIntelDebugTools();

  const leftControls = (
    <>
      <NavbarSearch />
      <RegionSelect multi />
      <MapLayoutSelect inlineLabel="Layout" />
      <JumpbridgesToggle />
      <MapZoomControls />
      <button
        className="btn btn-xs btn-info btn-outline"
        onClick={openShareLinkModal}
      >
        Share
      </button>
      <button
        className="btn btn-xs btn-info btn-outline"
        onClick={openHelpModal}
      >
        Help
      </button>
    </>
  );

  const rightControls = (
    <>
      <IntelServerStatus />
      <UploaderCountBadge />
      <ReportHealthBadge />
      <AlarmMuteToggleButton
        muted={alarmMuted}
        onToggle={() => applySetting("alarm", "enabled", !alarmEnabled)}
      />
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
      panelOpen={mapViewMode !== "full" || intelPanelOpen}
      panelClassName="w-96"
      onAutoHidePanel={() => applySetting("intel", "panelOpen", true)}
    >
      <MapCanvas />
      <ContextMenu />
    </MapPageShell>
  );
}
