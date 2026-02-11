import { useIntelStore } from "@/features/intel";

const SERVER_STATUS = {
  disconnected: { status: "Disconnected", color: "text-red-400" },
  connecting: { status: "Connecting", color: "text-amber-300" },
  connected: { status: "Connected", color: "text-emerald-300" },
} as const;

export default function IntelServerStatus() {
  const intelStatus = useIntelStore((state) => state.intelStatus);
  const status = SERVER_STATUS[intelStatus];
  const statusDot =
    intelStatus === "connected"
      ? "bg-emerald-300"
      : intelStatus === "connecting"
        ? "bg-amber-300"
        : "bg-red-400";

  return (
    <span
      className={`inline-block h-2.5 w-2.5 rounded-full ${statusDot}`}
      title={`Server: ${status.status}`}
    />
  );
}
