import { useMemo } from "react";

type MapResponse = {
  regions: Record<
    number,
    { region: number; name: string; position: { x: number; y: number } }
  >;
  systems: Record<
    number,
    { system: number; name: string; position: { x: number; y: number } }
  >;
  gates: Array<{ from: number; to: number }>;
};

export default function StaticMap({ data }: { data: MapResponse }) {
  const { systems, gates } = data;

  const bounds = useMemo(() => {
    const points = Object.values(systems).map((s) => s.position);
    const xs = points.map((p) => p.x);
    const ys = points.map((p) => p.y);
    const minX = Math.min(...xs);
    const maxX = Math.max(...xs);
    const minY = Math.min(...ys);
    const maxY = Math.max(...ys);
    return { minX, maxX, minY, maxY };
  }, [systems]);

  const width = 900;
  const height = 420;
  const padding = 40;

  const scaleX = (value: number) =>
    padding +
    ((value - bounds.minX) / (bounds.maxX - bounds.minX || 1)) *
      (width - padding * 2);
  const scaleY = (value: number) =>
    padding +
    ((value - bounds.minY) / (bounds.maxY - bounds.minY || 1)) *
      (height - padding * 2);

  return (
    <svg
      width="100%"
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className="bg-base-300/40 rounded-lg"
    >
      <g stroke="rgba(148,163,184,0.3)" strokeWidth="1">
        {gates.map((gate, idx) => {
          const from = systems[gate.from];
          const to = systems[gate.to];
          if (!from || !to) return null;
          return (
            <line
              key={idx}
              x1={scaleX(from.position.x)}
              y1={scaleY(from.position.y)}
              x2={scaleX(to.position.x)}
              y2={scaleY(to.position.y)}
            />
          );
        })}
      </g>
      <g>
        {Object.values(systems).map((sys) => (
          <circle
            key={sys.system}
            cx={scaleX(sys.position.x)}
            cy={scaleY(sys.position.y)}
            r={3}
            fill="rgba(34,197,94,0.8)"
          >
            <title>{sys.name}</title>
          </circle>
        ))}
      </g>
    </svg>
  );
}
