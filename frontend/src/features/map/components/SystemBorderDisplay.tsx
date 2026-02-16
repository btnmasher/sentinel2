import type { SystemBorderState } from "../hooks/useSystemBorderState";

export default function SystemBorderDisplay({
  border,
}: {
  border: SystemBorderState;
}) {
  if (!border.visible || !border.color) {
    return null;
  }

  return (
    <rect
      className={[
        "map-system-border",
        border.pulse ? "map-system-border-pulse" : "",
      ]
        .filter(Boolean)
        .join(" ")}
      x={-9}
      y={-9}
      width={18}
      height={18}
      rx={4}
      ry={4}
      fill="none"
      stroke={border.color}
      strokeWidth={2.25}
      style={{ filter: `drop-shadow(0 0 4px ${border.color})` }}
    />
  );
}
