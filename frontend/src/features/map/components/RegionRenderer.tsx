import { useMemo } from "react";
import MapRegionJumpranges from "./MapRegionJumpranges";
import MapSystem from "./MapSystem";
import type { Gate, Region, System } from "../types";
import { systemScale, useMapStore } from "../store/mapStore";
import { transformComponent } from "../utils/mapUtils";

export default function RegionRenderer({
  region,
  regionSystems,
  showBase = true,
  showSystems = true,
}: {
  region: Region;
  regionSystems: System[];
  showBase?: boolean;
  showSystems?: boolean;
}) {
  const gates = useMapStore((s) => s.gates);
  const mapScale = useMapStore((s) => s.mapScale);
  const jumpranges = useMapStore((s) => s.jumpranges);
  const systems = useMapStore((s) => s.systems);

  const regionSystemIds = useMemo(
    () => regionSystems.map((system) => system.system),
    [regionSystems],
  );

  const gateLines = useMemo(() => {
    return (gates as Gate[])
      .filter(
        (gate) => gate.type !== "region" && regionSystemIds.includes(gate.to),
      )
      .map((gate) => ({
        a: systems[gate.from]?.position,
        b: systems[gate.to]?.position,
        type: gate.type,
      }))
      .filter((gate) => gate.a && gate.b);
  }, [gates, regionSystemIds, systems]);

  const scale = systemScale(mapScale);

  return (
    <g className="map-region" transform={transformComponent(region.position)}>
      {showBase && jumpranges.enabled && (
        <MapRegionJumpranges
          regionSystems={regionSystems}
          mapScale={mapScale}
        />
      )}
      {showBase && (
        <text className="map-region-label" fontSize="40">
          {region.name}
        </text>
      )}

      <g>
        {showBase &&
          gateLines.map((gate, idx) => (
            <line
              key={`${gate.type}-${idx}`}
              x1={gate.a.x}
              y1={gate.a.y}
              x2={gate.b.x}
              y2={gate.b.y}
              className={`map-gate ${gate.type}`}
            />
          ))}

        {showSystems &&
          regionSystemIds.map((systemId) => (
            <MapSystem key={systemId} systemId={systemId} scale={scale} />
          ))}
      </g>
    </g>
  );
}
