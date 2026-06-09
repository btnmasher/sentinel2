import { useEffect } from "react";
import { useAuthStore } from "@/app/store/authStore";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useMapStore } from "@/features/map";
import { useIntelStore } from "../store/intelStore";

const ZKILL_REALTIME_RESYNC_INTERVAL_MS = 30_000;

export default function useIntelRealtime() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const intelStatus = useIntelStore((s) => s.intelStatus);
  const setIntelStatus = useIntelStore((s) => s.setIntelStatus);
  const connectRealtime = useIntelStore((s) => s.connectRealtime);
  const disconnectRealtime = useIntelStore((s) => s.disconnectRealtime);
  const syncZKillRealtime = useIntelStore((s) => s.syncZKillRealtime);
  const zkillFeedEnabled = useSettingsStore(
    (s) => s.settings.intel.zkillFeedEnabled,
  );
  const mapRegions = useMapStore((s) => s.mapRegions);
  const selectedRegionIds = mapRegions
    .map((value) => Number(value))
    .filter((value) => Number.isFinite(value) && value > 0);
  const selectedRegionKey = Array.from(new Set(selectedRegionIds))
    .slice()
    .sort((a, b) => a - b)
    .join(",");

  useEffect(() => {
    if (!isAuthenticated) {
      void disconnectRealtime();
      return;
    }

    setIntelStatus("connecting");
    void connectRealtime().then((result) => {
      if (result === "auth_error") {
        useAuthStore
          .getState()
          .forceLogout("Authentication expired, returning to home.");
      }
    });

    return () => {
      void disconnectRealtime();
    };
  }, [connectRealtime, disconnectRealtime, isAuthenticated, setIntelStatus]);

  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }
    const regionIds =
      selectedRegionKey === ""
        ? []
        : selectedRegionKey
            .split(",")
            .map((value) => Number(value))
            .filter((value) => Number.isFinite(value) && value > 0);
    void syncZKillRealtime(regionIds, zkillFeedEnabled);
  }, [
    isAuthenticated,
    intelStatus,
    selectedRegionKey,
    syncZKillRealtime,
    zkillFeedEnabled,
  ]);

  useEffect(() => {
    if (!isAuthenticated || intelStatus !== "connected") {
      return;
    }
    const regionIds =
      selectedRegionKey === ""
        ? []
        : selectedRegionKey
            .split(",")
            .map((value) => Number(value))
            .filter((value) => Number.isFinite(value) && value > 0);
    const sync = () => {
      void syncZKillRealtime(regionIds, zkillFeedEnabled);
    };
    const interval = window.setInterval(sync, ZKILL_REALTIME_RESYNC_INTERVAL_MS);
    sync();
    return () => {
      window.clearInterval(interval);
    };
  }, [
    isAuthenticated,
    intelStatus,
    selectedRegionKey,
    syncZKillRealtime,
    zkillFeedEnabled,
  ]);
}
