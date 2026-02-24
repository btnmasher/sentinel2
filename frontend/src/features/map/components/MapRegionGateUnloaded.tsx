import { useMapStore } from "../store/mapStore";
import { REGIONS } from "../types/regions";
import { placeUnloadedMarker } from "../utils/mapUtils";
import type { Gate } from "../types";
import UnloadedLink from "./UnloadedLink";

export default function MapRegionGateUnloaded({ gate }: { gate: Gate }) {
  const systems = useMapStore((s) => s.systems);
  const regions = useMapStore((s) => s.regions);
  const mapScale = useMapStore((s) => s.mapScale);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);

  const loadedSystem = regions[gate.to_region]
    ? systems[gate.to]
    : systems[gate.from];
  if (!loadedSystem) return null;

  const loadedRegion = regions[loadedSystem.region];
  const loadedPosition = {
    x: loadedRegion.position.x + loadedSystem.position.x,
    y: loadedRegion.position.y + loadedSystem.position.y,
  };

  const missingSide = !regions[gate.to_region] ? "to" : "from";
  const missingRegionId = gate[`${missingSide}_region`];
  const loadedSide = missingSide === "to" ? "from" : "to";
  const gateX = gate[`${loadedSide}_dotlan_x`] ?? gate[`${loadedSide}_metro_x`];
  const gateY = gate[`${loadedSide}_dotlan_y`] ?? gate[`${loadedSide}_metro_y`];
  let direction;
  if (gateX != null && gateY != null) {
    const base = {
      x: loadedRegion.position.x + gateX,
      y: loadedRegion.position.y + gateY,
    };
    direction = {
      x: base.x - loadedPosition.x,
      y: base.y - loadedPosition.y,
    };
  }

  const systemsInRegion = Object.values(systems)
    .filter((system) => system.region === loadedSystem.region)
    .map((system) => ({
      x: loadedRegion.position.x + system.position.x,
      y: loadedRegion.position.y + system.position.y,
    }));

  const pos = placeUnloadedMarker({
    origin: loadedPosition,
    hashSeed: `${missingRegionId},${loadedSystem.system},${gate.from},${gate.to}`,
    direction,
    systems: systemsInRegion,
    minDistance: 90,
    maxDistance: 200,
  });

  const missingRegion = REGIONS.find(
    (region) => region.region === String(missingRegionId),
  );

  const loadMissing = () => {
    if (!missingRegion) return;
    updateMapConfig({
      mapRegions: Array.from(
        new Set([...useMapStore.getState().mapRegions, missingRegion.region]),
      ),
    });
  };

  return (
    <UnloadedLink
      origin={loadedPosition}
      position={pos}
      label={missingRegion?.name}
      mapScale={mapScale}
      className="map-gate region"
      onClick={loadMissing}
    />
  );
}
