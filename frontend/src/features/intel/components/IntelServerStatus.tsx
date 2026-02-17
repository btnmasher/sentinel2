import { useIntelStore } from "@/features/intel";

const SERVER_STATUS = {
  disconnected: { status: "Disconnected", color: "text-red-400" },
  connecting: { status: "Connecting", color: "text-amber-300" },
  connected: { status: "Connected", color: "text-emerald-300" },
} as const;

export default function IntelServerStatus() {
  const intelStatus = useIntelStore((state) => state.intelStatus);
  const version = useIntelStore((state) => state.version);
  const status = SERVER_STATUS[intelStatus];
  const title =
    status === SERVER_STATUS.connected
      ? `Server: ${status.status}\nVersion: ${version || "-"}`
      : `Server: ${status.status}`;
  const statusDot =
    status === SERVER_STATUS.connected
      ? "bg-emerald-300"
      : status === SERVER_STATUS.connecting
        ? "bg-amber-300"
        : "bg-red-400";

  return (
    <span
      className={`inline-block h-2.5 w-2.5 rounded-full ${statusDot}`}
      title={title}
    />
  );
}
