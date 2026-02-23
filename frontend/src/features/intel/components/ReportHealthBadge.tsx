import { useEffect, useMemo, useState } from "react";
import { ScrollText } from "lucide-react";
import { useIntelStore } from "@/features/intel";
import { pb } from "@/config/pb";
import HoverCard from "@/components/HoverCard";

const CHANNEL_WARN_AFTER_SECONDS = 10 * 60;
const CHANNEL_STALE_AFTER_SECONDS = 60 * 60;
const HEALTH_RECOMPUTE_INTERVAL_MS = 30_000;

const CHANNEL_STATUS_STYLE = {
  active: {
    label: "Active",
    textColor: "intel-status-text-active",
    dotColor: "intel-status-dot-active",
    badgeClass: "badge-success",
  },
  stale: {
    label: "Stale",
    textColor: "intel-status-text-warn",
    dotColor: "intel-status-dot-warn",
    badgeClass: "badge-warning",
  },
  veryStale: {
    label: "Very stale",
    textColor: "intel-status-text-stale",
    dotColor: "intel-status-dot-stale",
    badgeClass: "badge-error",
  },
} as const;

type ChannelStatus = keyof typeof CHANNEL_STATUS_STYLE;
const CHANNEL_STATUS_SCORE: Record<ChannelStatus, number> = {
  active: 2,
  stale: 1,
  veryStale: 0,
};

export default function ReportHealthBadge() {
  const reports = useIntelStore((state) => state.reports);
  const reportCount = useIntelStore((state) => state.reportCount);
  const [nowMs, setNowMs] = useState(() => Date.now());
  const [configuredChannels, setConfiguredChannels] = useState<
    Array<{ id: string; name: string }>
  >([]);

  useEffect(() => {
    let active = true;
    pb.collection("intel_channels")
      .getFullList({ sort: "channel_name" })
      .then((records) => {
        if (!active) return;
        const channels = records
          .map((record) => ({
            id: String(record.id || "").trim(),
            name: String(record.channel_name || "").trim(),
          }))
          .filter((channel) => channel.id && channel.name);
        setConfiguredChannels(channels);
      })
      .catch(() => {
        if (active) {
          setConfiguredChannels([]);
        }
      });

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const intervalID = window.setInterval(() => {
      setNowMs(Date.now());
    }, HEALTH_RECOMPUTE_INTERVAL_MS);
    return () => {
      window.clearInterval(intervalID);
    };
  }, []);

  const summary = useMemo(() => {
    const now = Math.floor(nowMs / 1000);
    const latestByChannelID = new Map<string, number>();

    for (const report of reports) {
      const channel = (report.channel_id || "").trim();
      if (!channel) continue;
      const current = latestByChannelID.get(channel) ?? 0;
      if (report.time > current) {
        latestByChannelID.set(channel, report.time);
      }
    }

    const channelRows: Array<{
      channelId: string;
      channelName: string;
      ageSeconds: number;
      status: ChannelStatus;
      hasUpdates: boolean;
    }> = [];
    const counts: Record<ChannelStatus, number> = {
      active: 0,
      stale: 0,
      veryStale: 0,
    };

    for (const channel of configuredChannels) {
      const latestTimestamp = latestByChannelID.get(channel.id) ?? 0;
      const hasUpdates = latestTimestamp > 0;
      const ageSeconds = hasUpdates ? Math.max(0, now - latestTimestamp) : 0;
      const status: ChannelStatus =
        hasUpdates && ageSeconds <= CHANNEL_WARN_AFTER_SECONDS
          ? "active"
          : hasUpdates && ageSeconds <= CHANNEL_STALE_AFTER_SECONDS
            ? "stale"
            : "veryStale";
      counts[status]++;
      channelRows.push({
        channelId: channel.id,
        channelName: channel.name,
        ageSeconds,
        status,
        hasUpdates,
      });
    }

    channelRows.sort((a, b) => {
      if (a.status !== b.status) {
        return CHANNEL_STATUS_SCORE[a.status] - CHANNEL_STATUS_SCORE[b.status];
      }
      if (a.hasUpdates !== b.hasUpdates) {
        return a.hasUpdates ? -1 : 1;
      }
      if (a.ageSeconds !== b.ageSeconds) {
        return a.ageSeconds - b.ageSeconds;
      }
      return a.channelName.localeCompare(b.channelName);
    });
    return {
      channels: channelRows,
      counts,
    };
  }, [configuredChannels, nowMs, reports]);

  const averageScore =
    summary.channels.length === 0
      ? 0
      : summary.channels.reduce(
          (total, channel) => total + CHANNEL_STATUS_SCORE[channel.status],
          0,
        ) / summary.channels.length;

  const overallStatus: ChannelStatus =
    averageScore >= 1.5
      ? "active"
      : averageScore >= 0.75
        ? "stale"
        : "veryStale";

  const outlierStatus: ChannelStatus =
    summary.channels.length === 0
      ? "veryStale"
      : summary.counts.veryStale > 0
        ? "veryStale"
        : summary.counts.stale > 0
          ? "stale"
          : "active";

  const statusStyle = CHANNEL_STATUS_STYLE[overallStatus];
  const outlierStyle = CHANNEL_STATUS_STYLE[outlierStatus];
  const showOutlierDot = outlierStatus !== overallStatus;

  const formatAge = (ageSeconds: number) => {
    if (ageSeconds < 60) return `${ageSeconds}s`;
    const minutes = Math.floor(ageSeconds / 60);
    if (minutes < 60) return `${minutes}m`;
    const hours = Math.floor(minutes / 60);
    return `${hours}h`;
  };

  return (
    <HoverCard
      trigger={
        <button
          type="button"
          className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content transition-colors hover:bg-base-300/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          aria-label="Intel report health badge"
        >
          <span
            className={`intel-badge-icon-bg relative inline-flex h-6 w-6 items-center justify-center rounded-full text-base-content ${
              overallStatus === "veryStale" ? "intel-status-icon--alert" : ""
            }`}
          >
            <ScrollText className={`h-3.5 w-3.5 ${statusStyle.textColor}`} />
            {showOutlierDot ? (
              <span
                className={`absolute -right-0.5 -top-0.5 h-2.5 w-2.5 rounded-full border border-base-100 ${outlierStyle.dotColor}`}
              />
            ) : null}
          </span>
          <span>{reportCount}</span>
        </button>
      }
      className="hover-card-surface intel-badge-hover-card w-96 max-w-[calc(100vw-2rem)] p-3 text-xs"
    >
      <div className="flex items-center justify-between">
        <p className="text-sm font-semibold">Intel report health</p>
        <span
          className={`inline-flex items-center gap-1.5 ${statusStyle.textColor}`}
        >
          <span className={`h-2 w-2 rounded-full ${statusStyle.dotColor}`} />
          {statusStyle.label}
        </span>
      </div>
      <p className="mt-1 text-base-content/80">
        {reportCount} report{reportCount === 1 ? "" : "s"} seen
        {summary.channels.length > 0
          ? ` across ${summary.channels.length} channel${summary.channels.length === 1 ? "" : "s"}.`
          : "."}
      </p>
      <div className="mt-3 flex flex-wrap gap-1.5">
        <span
          className={`badge badge-xs ${CHANNEL_STATUS_STYLE.active.badgeClass}`}
        >
          Active {summary.counts.active}
        </span>
        <span
          className={`badge badge-xs ${CHANNEL_STATUS_STYLE.stale.badgeClass}`}
        >
          Stale {summary.counts.stale}
        </span>
        <span
          className={`badge badge-xs ${CHANNEL_STATUS_STYLE.veryStale.badgeClass}`}
        >
          Very stale {summary.counts.veryStale}
        </span>
      </div>
      {summary.channels.length > 0 ? (
        <div className="mt-3 max-h-36 space-y-1 overflow-y-auto rounded-md border border-base-content/10 bg-base-200/45 p-2">
          {summary.channels.map((channel) => {
            const rowStyle = CHANNEL_STATUS_STYLE[channel.status];
            return (
              <div
                key={channel.channelId}
                className="flex items-center justify-between gap-2"
              >
                <span className="truncate text-base-content/85">
                  {channel.channelName}
                </span>
                <span
                  className={`inline-flex items-center gap-1.5 ${rowStyle.textColor}`}
                >
                  <span
                    className={`h-2 w-2 rounded-full ${rowStyle.dotColor}`}
                  />
                  {!channel.hasUpdates
                    ? "has had no report yet"
                    : channel.status === "active"
                      ? `last report ${formatAge(channel.ageSeconds)} ago`
                      : `has had no report for ${formatAge(channel.ageSeconds)}`}
                </span>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="mt-3 text-base-content/70">No channels configured yet.</p>
      )}
    </HoverCard>
  );
}
