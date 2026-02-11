import { transformComponent } from "../utils/mapUtils";

type UnloadedLinkProps = {
  origin: { x: number; y: number };
  position: { x: number; y: number };
  label?: string;
  mapScale: number;
  className: string;
  onClick?: () => void;
};

export default function UnloadedLink({
  origin,
  position,
  label,
  mapScale,
  className,
  onClick,
}: UnloadedLinkProps) {
  const scale = mapScale > 1.95 ? 1.95 / mapScale : 1.25;

  return (
    <g className="map-gate-unloaded" onClick={onClick}>
      <line
        className={className}
        opacity={0.5}
        x1={origin.x}
        y1={origin.y}
        x2={position.x}
        y2={position.y}
      />
      <g transform={transformComponent(position, scale)}>
        <rect className={className} x={-8} y={-8} height={16} width={16} rx={3} ry={3} />
        {label && mapScale > 0.85 && (
          <>
            <text
              textAnchor="middle"
              className="map-gate-unloaded-label"
              fontSize={6}
              x={0}
              y={2}
            >
              load
            </text>
            <text
              textAnchor="middle"
              className="map-gate-unloaded-label"
              fontSize={6}
              x={0}
              y={16}
            >
              {label}
            </text>
          </>
        )}
      </g>
    </g>
  );
}
