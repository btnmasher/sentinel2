import IntelPanelLog from "./IntelPanelLog";
import type { IntelReport } from "../types";
import AccordionCard from "@/components/AccordionCard";

type IntelFeedCardProps = {
  logs: IntelReport[];
  channelNames: Record<string, string>;
  open: boolean;
  onToggle: () => void;
};

export default function IntelFeedCard({
  logs,
  channelNames,
  open,
  onToggle,
}: IntelFeedCardProps) {
  return (
    <AccordionCard
      title="Intel Feed"
      subtitle="Latest reports"
      open={open}
      onToggle={onToggle}
      className="intel-feed-card min-h-0 flex flex-1 flex-col"
      contentClassName="min-h-0 flex flex-1 flex-col overflow-hidden"
    >
      {logs.length === 0 ? (
        <div className="flex min-h-32 items-center justify-center text-xs text-slate-500">
          No reports yet.
        </div>
      ) : (
        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-2">
          {logs.map((report) => (
            <IntelPanelLog
              key={report.id}
              log={report}
              channelNames={channelNames}
            />
          ))}
        </div>
      )}
    </AccordionCard>
  );
}
