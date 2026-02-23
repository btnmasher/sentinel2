import { useIntelStore } from "@/features/intel";
import HoverCard from "@/components/HoverCard";

const SERVER_STATUS = {
  disconnected: { status: "Disconnected", color: "text-red-400" },
  connecting: { status: "Connecting", color: "text-amber-300" },
  connected: { status: "Connected", color: "text-emerald-300" },
} as const;

export default function IntelServerStatus() {
  const intelStatus = useIntelStore((state) => state.intelStatus);
  const version = useIntelStore((state) => state.version);
  const status = SERVER_STATUS[intelStatus];
  const statusDot =
    status === SERVER_STATUS.connected
      ? "bg-emerald-300"
      : status === SERVER_STATUS.connecting
        ? "bg-amber-300"
        : "bg-red-400";
  const versionLabel = version || "-";

  return (
    <HoverCard
      trigger={
        <button
          type="button"
          className={`inline-block h-2.5 w-2.5 rounded-full ${statusDot}`}
          aria-label={`Server status ${status.status}`}
        />
      }
      className="hover-card-surface server-status-hover-card"
    >
      <div className="server-status-hover-title">Server Status</div>
      <div className="server-status-hover-row">
        <span className={`server-status-hover-dot ${statusDot}`} />
        <span>{status.status}</span>
      </div>
      <div className="server-status-hover-meta">Version: {versionLabel}</div>
    </HoverCard>
  );
}
