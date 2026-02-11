import IntelPanelLog from "./IntelPanelLog";
import type { IntelReport } from "../types";
import AccordionCard from "@/components/AccordionCard";

type IntelFeedCardProps = {
  logs: IntelReport[];
  open: boolean;
  onToggle: () => void;
};

export default function IntelFeedCard({
  logs,
  open,
  onToggle,
}: IntelFeedCardProps) {
  return (
    <AccordionCard
      title="Intel Feed"
      subtitle="Latest reports"
      open={open}
      onToggle={onToggle}
    >
      {logs.length === 0 ? (
        <div className="flex items-center justify-center text-xs text-slate-500 min-h-32">
          No reports yet.
        </div>
      ) : (
        <div className="space-y-3 overflow-y-auto max-h-[55vh] pr-2">
          {logs.map((report) => (
            <IntelPanelLog key={report.id} log={report} />
          ))}
        </div>
      )}
    </AccordionCard>
  );
}
