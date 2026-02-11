import { useEffect, useMemo, useState } from "react";
import { api } from "@/config/api";
import { REGIONS, useMapStore } from "@/features/map";

type SystemSearch = {
  id: number;
  name: string;
  region: string;
  region_id: number;
};
type RegionSearch = { id: string; name: string };

export default function NavbarSearch() {
  const [search, setSearch] = useState("");
  const [results, setResults] = useState<SystemSearch[]>([]);
  const [loading, setLoading] = useState(false);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);

  const loadedRegions = useMemo(
    () => new Set(mapRegions.map((r) => String(r))),
    [mapRegions],
  );
  const regionResults = useMemo<RegionSearch[]>(() => {
    if (search.trim().length < 2) {
      return [];
    }
    const lower = search.trim().toLowerCase();
    return REGIONS.filter((region) =>
      region.name.toLowerCase().includes(lower),
    ).map((region) => ({ id: region.region, name: region.name }));
  }, [search]);

  useEffect(() => {
    if (search.trim().length < 2) {
      setResults([]);
      return;
    }
    const handler = setTimeout(() => {
      setLoading(true);
      api
        .get(`/map/search?q=${encodeURIComponent(search)}`)
        .then((res) => setResults(res.data.systems || []))
        .finally(() => setLoading(false));
    }, 250);
    return () => clearTimeout(handler);
  }, [search]);

  const selectSystem = (system: SystemSearch) => {
    if (!loadedRegions.has(String(system.region_id))) {
      updateMapConfig({
        mapRegions: [...mapRegions, String(system.region_id)],
      });
    }
    setSystemSearch(system.id);
    setSearch("");
    setResults([]);
  };

  const selectRegion = (region: RegionSearch) => {
    if (!loadedRegions.has(region.id)) {
      updateMapConfig({ mapRegions: [...mapRegions, region.id] });
    }
    setSystemSearch(undefined);
    setSearch("");
    setResults([]);
  };

  return (
    <div className="relative">
      <input
        className="input input-xs input-bordered bg-base-200 w-64 px-2"
        placeholder="Search"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
      />
      {loading && (
        <span className="absolute right-3 top-2 text-xs text-slate-400">
          ...
        </span>
      )}
      {(regionResults.length > 0 || results.length > 0) && (
        <div className="absolute mt-2 w-72 bg-base-200 border border-slate-800 rounded-lg shadow-lg z-50">
          {regionResults.length > 0 && (
            <div className="px-3 pt-2 text-[10px] uppercase tracking-[0.2em] text-slate-500">
              Regions
            </div>
          )}
          {regionResults.map((region) => {
            const loaded = loadedRegions.has(region.id);
            return (
              <button
                key={`region-${region.id}`}
                className="w-full text-left px-3 py-2 hover:bg-base-300 text-sm"
                onClick={() => selectRegion(region)}
              >
                <div className="font-semibold text-slate-100">
                  {region.name}
                </div>
                <div
                  className={`text-xs ${loaded ? "text-slate-400" : "text-fuchsia-300"}`}
                >
                  {loaded ? "Loaded" : "Load region"}
                </div>
              </button>
            );
          })}
          {regionResults.length > 0 && results.length > 0 && (
            <div className="mx-3 border-t border-slate-800" />
          )}
          {results.map((system) => {
            const loaded = loadedRegions.has(String(system.region_id));
            return (
              <button
                key={system.id}
                className="w-full text-left px-3 py-2 hover:bg-base-300 text-sm"
                onClick={() => selectSystem(system)}
              >
                <div className="font-semibold text-slate-100">
                  {system.name}
                </div>
                <div
                  className={`text-xs ${loaded ? "text-slate-400" : "text-fuchsia-300"}`}
                >
                  {system.region}
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
