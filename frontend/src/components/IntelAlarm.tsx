import { useEffect, useRef } from "react";
import { useIntelStore } from "@/features/intel";
import { useMapStore } from "@/features/map";
import { useSettingsStore } from "@/app/store/settingsStore";
import { UI_DIALOG } from "@/app/store/uiStore";
import { useAppModal } from "@/components/dialogs/AppModals";

export default function IntelAlarm() {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const hasConnectedRef = useRef(false);
  const logFilters = useIntelStore((s) => s.logFilters);
  const lastReports = useIntelStore((s) => s.lastReports);
  const intelStatus = useIntelStore((s) => s.intelStatus);
  const regions = useMapStore((s) => s.regions);
  const settings = useSettingsStore((s) => s.settings);
  const { open: openAlarmStartModal } = useAppModal(UI_DIALOG.AlarmStart);

  const play = (overrideVolume?: number) => {
    if (!settings.alarm.enabled) return;
    const audio = audioRef.current;
    if (!audio) return;
    const volume =
      overrideVolume !== undefined
        ? overrideVolume
        : settings.alarm.volume / 100;
    audio.volume = volume;
    const result = audio.play();
    if (result && result.catch) {
      result.catch(() => {
        if (!settings.introduction) {
          openAlarmStartModal();
        }
      });
    }
  };

  useEffect(() => {
    if (!settings.alarm.enabled) return;
    const storageKey = "sentinel_autoplay_checked";
    if (sessionStorage.getItem(storageKey)) {
      return;
    }
    sessionStorage.setItem(storageKey, "1");
    play(0);
  }, [settings.alarm.enabled]);

  useEffect(() => {
    if (intelStatus === "connected") {
      hasConnectedRef.current = true;
      return;
    }
    if (intelStatus === "disconnected") {
      if (hasConnectedRef.current) {
        play();
      }
    }
  }, [intelStatus]);

  useEffect(() => {
    if (!lastReports.length) return;
    const newLog = lastReports[0];
    const regionIds = new Set(
      Object.keys(regions).map((id) => parseInt(id, 10)),
    );
    const unknownLocation = newLog.systems.length === 0;
    const loadedRegion = newLog.regions.some((region) => regionIds.has(region));
    const hasSystemFiltered = newLog.systems.some((system) =>
      logFilters.system.includes(system.system),
    );

    if (logFilters.system.length > 0 && logFilters.includeSystemAlarm) {
      if (hasSystemFiltered) play();
      return;
    }

    if (!logFilters.includeUnknownAlarm && unknownLocation) {
      return;
    }
    if (
      !logFilters.includeUnloadedRegionsAlarm &&
      !loadedRegion &&
      !unknownLocation
    ) {
      return;
    }

    play();
  }, [lastReports, logFilters, regions]);

  return <audio ref={audioRef} src={`/audio/${settings.alarm.sound}.mp3`} />;
}
