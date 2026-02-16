import { useMemo } from "react";
import { useMapStore } from "../store/mapStore";
import { useSettingsStore } from "@/app/store/settingsStore";

export function useSystemVisibility() {
  const mapScale = useMapStore((s) => s.mapScale);
  const alwaysShowSystems = useSettingsStore(
    (s) => s.settings.map.alwaysShowSystems,
  );

  return useMemo(
    () => ({
      showSystem: alwaysShowSystems || mapScale > 0.4,
      showSystemText: alwaysShowSystems || mapScale > 0.75,
    }),
    [alwaysShowSystems, mapScale],
  );
}
