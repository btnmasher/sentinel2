import ReportItem from "./ReportItem";
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
    >
      {logs.length === 0 ? (
        <div className="flex min-h-32 items-center justify-center text-xs text-slate-500">
          No reports yet.
        </div>
      ) : (
        <div className="min-h-0 max-h-[50vh] space-y-3 overflow-y-auto pr-2">
          {logs.map((report) => (
            <ReportItem
              key={report.recordId ?? String(report.id)}
              log={report}
              channelNames={channelNames}
            />
          ))}
        </div>
      )}
    </AccordionCard>
  );
}
