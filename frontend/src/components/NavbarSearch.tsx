import { useMemo, useState } from "react";
import {
  SystemSearchField,
  type MapRegionSearchResult,
  type MapSystemSearchResult,
  useMapStore,
} from "@/features/map";

export default function NavbarSearch() {
  const [search, setSearch] = useState("");
  const mapRegions = useMapStore((s) => s.mapRegions);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);

  const loadedRegions = useMemo(
    () => new Set(mapRegions.map((r) => String(r))),
    [mapRegions],
  );

  const selectSystem = (system: MapSystemSearchResult) => {
    if (!loadedRegions.has(String(system.region_id))) {
      updateMapConfig({
        mapRegions: [...mapRegions, String(system.region_id)],
      });
    }
    setSystemSearch(system.id);
    setSearch("");
  };

  const selectRegion = (regionId: string) => {
    if (!loadedRegions.has(regionId)) {
      updateMapConfig({ mapRegions: [...mapRegions, regionId] });
    }
    setSystemSearch(undefined);
    setSearch("");
  };

  return (
    <div className="relative">
      <SystemSearchField
        query={search}
        onQueryChange={setSearch}
        onSelect={selectSystem}
        includeRegions
        onSelectRegion={(region: MapRegionSearchResult) =>
          selectRegion(region.id)
        }
        placeholder="Search systems or regions"
        containerClassName="w-56"
        inputClassName="input input-xs input-bordered bg-base-200 w-56 px-2"
        panelClassName="max-h-44 overflow-auto rounded-lg border border-slate-800 bg-base-200 shadow-lg"
        minQueryLength={2}
        selectionInputMode="clear"
      />
    </div>
  );
}
