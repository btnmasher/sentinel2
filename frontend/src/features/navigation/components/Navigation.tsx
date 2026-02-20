import { useState } from "react";
import {
  ContextMenu,
  JumpbridgesToggle,
  TimersToggle,
  MapLayoutSelect,
  MapPageShell,
  MapZoomControls,
  MapCanvas,
  RegionSelect,
} from "@/features/map";
import NavigationPanel from "./NavigationPanel";
import { useNavigationStore } from "../store/navigationStore";
import { UI_DIALOG } from "@/app/store/uiStore";
import { useAppModal } from "@/components/dialogs/AppModals";
import { useSettingsStore } from "@/app/store/settingsStore";
import AlarmMuteToggleButton from "@/components/AlarmMuteToggleButton";
import { ChevronDown, ChevronUp, Route } from "lucide-react";

export default function Navigation() {
  const [panelOpen, setPanelOpen] = useState(true);

  const route = useNavigationStore((s) => s.route);

  const { open: openHelpModal } = useAppModal(UI_DIALOG.Help);
  const mapViewMode = useSettingsStore((s) => s.settings.map.viewMode);
  const alarmEnabled = useSettingsStore((s) => s.settings.alarm.enabled);
  const alarmVolume = useSettingsStore((s) => s.settings.alarm.volume);
  const applySetting = useSettingsStore((s) => s.apply);
  const alarmMuted = !alarmEnabled || alarmVolume <= 0;

  const leftControls = (
    <>
      <RegionSelect multi />
      <MapLayoutSelect inlineLabel="Layout" />
      <JumpbridgesToggle />
      <TimersToggle />
      <MapZoomControls />
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
      {route.length > 0 && (
        <span className="px-2 py-1 rounded border border-slate-700 bg-base-300/70">
          Route: {route.length}
        </span>
      )}
      <AlarmMuteToggleButton
        muted={alarmMuted}
        onToggle={() => applySetting("alarm", "enabled", !alarmEnabled)}
      />
      {mapViewMode === "full" && (
        <button
          className="btn btn-xs btn-ghost"
          onClick={() => setPanelOpen((prev) => !prev)}
          aria-label="Toggle navigation panel"
        >
          {panelOpen ? (
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
      pageBadge={{ icon: <Route className="h-3 w-3" />, label: "Navigation" }}
      leftControls={leftControls}
      rightControls={rightControls}
      panel={<NavigationPanel />}
      panelOpen={panelOpen}
      panelClassName="w-96"
      onAutoHidePanel={() => setPanelOpen(true)}
    >
      <MapCanvas />
      <ContextMenu />
    </MapPageShell>
  );
}
