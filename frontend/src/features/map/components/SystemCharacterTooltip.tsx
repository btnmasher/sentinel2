import type { SystemCharacterBadge } from "../hooks/useSystemCharacters";

const STATE_COLOR = {
  undocked: "#38bdf8",
  docked: "#94a3b8",
} as const;

export default function SystemCharacterTooltip({
  characters,
}: {
  characters: SystemCharacterBadge[];
}) {
  if (characters.length === 0) {
    return null;
  }

  const labels = characters.map(
    (char) => `${char.name} (${char.inSpace ? "Undocked" : "Docked"})`,
  );
  const longestLabel = labels.reduce(
    (max, label) => Math.max(max, label.length),
    0,
  );
  const width = Math.min(280, Math.max(138, Math.ceil(longestLabel * 5.6 + 26)));
  const rowHeight = 12;
  const padding = 6;
  const height = padding * 2 + rowHeight * characters.length;

  return (
    <g
      className="map-system-character-tooltip"
      transform="translate(12 -10)"
      pointerEvents="none"
    >
      <rect
        className="map-system-character-tooltip-bg"
        x={0}
        y={0}
        width={width}
        height={height}
        rx={4}
        ry={4}
      />
      {characters.map((char, idx) => {
        const y = padding + idx * rowHeight + 6;
        return (
          <g key={char.id}>
            <circle
              cx={8}
              cy={y - 2}
              r={2.4}
              fill={char.inSpace ? STATE_COLOR.undocked : STATE_COLOR.docked}
            />
            <text x={14} y={y}>
              {char.name} ({char.inSpace ? "Undocked" : "Docked"})
            </text>
          </g>
        );
      })}
    </g>
  );
}
