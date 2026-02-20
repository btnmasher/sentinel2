import { useMemo } from "react";
import { useMapStore } from "../store/mapStore";
import { REGIONS } from "../types/regions";
import { placeUnloadedMarker } from "../utils/mapUtils";
import UnloadedLink from "./UnloadedLink";
import type { Region, System } from "../types";

export function buildJumpbridgePath(
  from: System,
  to: System,
  regions: Record<number, Region>,
) {
  const aX = from.position.x + regions[from.region].position.x;
  const aY = from.position.y + regions[from.region].position.y;
  const bX = to.position.x + regions[to.region].position.x;
  const bY = to.position.y + regions[to.region].position.y;

  const dX = Math.abs(aX - bX);
  const dY = Math.abs(aY - bY);
  const midX = dX < 20 ? aX - 30 : (aX + bX) / 2;
  const midY = dY < 20 ? aY - 30 : (aY + bY) / 2;
  const dxyA = dX > dY ? `${aX},${midY}` : `${midX},${aY}`;
  const dxyB = dX > dY ? `${bX},${midY}` : `${midX},${bY}`;

  return `M ${aX},${aY} C ${aX},${aY} ${dxyA} ${midX},${midY} ${dxyB} ${bX},${bY} ${bX},${bY}`;
}

export default function MapJumpbridge({
  jumpbridge,
}: {
  jumpbridge: {
    from: number;
    to: number;
    from_region?: number;
    to_region?: number;
    friendly: boolean;
    disabled?: boolean;
  };
}) {
  const systems = useMapStore((s) => s.systems);
  const regions = useMapStore((s) => s.regions);
  const mapScale = useMapStore((s) => s.mapScale);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);

  const bridgedSystems = useMemo(() => {
    if (!systems[jumpbridge.to]) {
      const missingRegion =
        jumpbridge.to_region &&
        REGIONS.find(
          (region) => region.region === String(jumpbridge.to_region),
        );
      return {
        from: systems[jumpbridge.from],
        missing: missingRegion,
      };
    }
    if (!systems[jumpbridge.from]) {
      const missingRegion =
        jumpbridge.from_region &&
        REGIONS.find(
          (region) => region.region === String(jumpbridge.from_region),
        );
      return {
        from: systems[jumpbridge.to],
        missing: missingRegion,
      };
    }
    return {
      from: systems[jumpbridge.from],
      to: systems[jumpbridge.to],
    };
  }, [jumpbridge, systems]);

  if (!bridgedSystems.from) return null;

  if (bridgedSystems.to) {
    const path = buildJumpbridgePath(
      bridgedSystems.from,
      bridgedSystems.to,
      regions,
    );

    return (
      <path
        className={`map-gate jumpbridge${jumpbridge.disabled ? " jumpbridge-disabled" : ""}`}
        opacity={0.5}
        d={path}
      />
    );
  }

  if (bridgedSystems.missing) {
    const base = {
      x:
        bridgedSystems.from.position.x +
        regions[bridgedSystems.from.region].position.x,
      y:
        bridgedSystems.from.position.y +
        regions[bridgedSystems.from.region].position.y,
    };
    const systemsInRegion = Object.values(systems)
      .filter((system) => system.region === bridgedSystems.from.region)
      .map((system) => ({
        x: regions[system.region].position.x + system.position.x,
        y: regions[system.region].position.y + system.position.y,
      }));

    const pos = placeUnloadedMarker({
      origin: base,
      hashSeed: `${bridgedSystems.from.system},${bridgedSystems.missing.region}`,
      systems: systemsInRegion,
      minDistance: 90,
      maxDistance: 180,
    });
    const missingRegion = REGIONS.find(
      (region) => region.region === String(bridgedSystems.missing?.region),
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
        origin={base}
        position={pos}
        label={bridgedSystems.missing.name}
        mapScale={mapScale}
        className={`map-gate jumpbridge${jumpbridge.disabled ? " jumpbridge-disabled" : ""}`}
        onClick={loadMissing}
      />
    );
  }

  return null;
}
