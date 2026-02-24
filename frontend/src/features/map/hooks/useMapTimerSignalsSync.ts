import { useEffect } from "react";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useTimersStore } from "@/features/timers";
import { useMapStore } from "../store/mapStore";

export default function useMapTimerSignalsSync() {
  const timersEnabled = useAppConfigStore((s) => s.timersEnabled);
  const timersLoadedAt = useTimersStore((s) => s.loadedAt);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const displayTimers = useMapStore((s) => s.displayTimers !== false);
  const fetchMapOverlays = useMapStore((s) => s.fetchMapOverlays);

  useEffect(() => {
    if (!timersEnabled) return;
    if (!displayTimers) return;
    if (!timersLoadedAt) return;
    if (!mapRegions || mapRegions.length === 0) return;

    const timer = window.setTimeout(() => {
      void fetchMapOverlays();
    }, 250);

    return () => window.clearTimeout(timer);
  }, [
    displayTimers,
    fetchMapOverlays,
    mapRegions,
    timersEnabled,
    timersLoadedAt,
  ]);
}
