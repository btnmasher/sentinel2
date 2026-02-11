import { useMemo } from "react";
import * as d3 from "d3";
import { useMapStore } from "../store/mapStore";
import { METERS_PER_LIGHTYEAR } from "../types/constants";
import type { System } from "../types";

const BOUNDS_OFFSET = 25;

const SELECTED_COLOR = "#66393e";
const PRIMARY_COLOR = "#1A6C8F";
const SECONDARY_COLOR = "#0E3C4F";

function calculateSystemColor(
  system: System,
  selectedSystem: System,
  primaryDistance?: number,
  secondaryDistance?: number,
) {
  const value = Math.sqrt(
    (selectedSystem.absolute.x - system.absolute.x) ** 2 +
      (selectedSystem.absolute.y - system.absolute.y) ** 2 +
      (selectedSystem.absolute.z - system.absolute.z) ** 2,
  );

  if (system.system === selectedSystem.system) {
    return SELECTED_COLOR;
  }
  if (primaryDistance && value < primaryDistance * METERS_PER_LIGHTYEAR) {
    return PRIMARY_COLOR;
  }
  if (secondaryDistance && value < secondaryDistance * METERS_PER_LIGHTYEAR) {
    return SECONDARY_COLOR;
  }
  return "none";
}

export default function MapRegionJumpranges({
  regionSystems,
  mapScale,
}: {
  regionSystems: System[];
  mapScale: number;
}) {
  const systems = useMapStore((s) => s.systems);
  const jumpranges = useMapStore((s) => s.jumpranges);

  const selectedSystem = useMemo(() => {
    const system = jumpranges.selectedSystem
      ? systems[jumpranges.selectedSystem]
      : undefined;
    return (
      system ?? regionSystems[Math.floor(Math.random() * regionSystems.length)]
    );
  }, [jumpranges.selectedSystem, regionSystems, systems]);

  const polygons = useMemo(() => {
    if (!selectedSystem) return [];
    const xs = regionSystems.map((s) => s.position.x);
    const ys = regionSystems.map((s) => s.position.y);
    const minX = Math.min(...xs) - BOUNDS_OFFSET;
    const maxX = Math.max(...xs) + BOUNDS_OFFSET;
    const minY = Math.min(...ys) - BOUNDS_OFFSET;
    const maxY = Math.max(...ys) + BOUNDS_OFFSET;

    const points = regionSystems.map((system) => [
      system.position.x,
      system.position.y,
    ]);
    const colors = regionSystems.map((system) =>
      calculateSystemColor(
        system,
        selectedSystem,
        jumpranges.primary,
        jumpranges.secondary,
      ),
    );

    const delaunay = d3.Delaunay.from(points as [number, number][]);
    const voronoi = delaunay.voronoi([minX, minY, maxX, maxY]);

    return points.map((_, i) => {
      const polygon = voronoi.cellPolygon(i);
      return {
        points: polygon
          ? polygon.map((p: [number, number]) => p.join(",")).join(" ")
          : "",
        color: colors[i],
      };
    });
  }, [jumpranges.primary, jumpranges.secondary, regionSystems, selectedSystem]);

  if (!selectedSystem) return null;

  return (
    <g opacity={mapScale < 0.2 ? 0.3 : 0.7}>
      {polygons.map((polygon, idx) => (
        <polygon
          key={idx}
          points={polygon.points}
          strokeWidth={1}
          strokeOpacity={0.2}
          vectorEffect="non-scaling-stroke"
          stroke="#000"
          opacity={0.7}
          fill={polygon.color}
        />
      ))}
    </g>
  );
}
