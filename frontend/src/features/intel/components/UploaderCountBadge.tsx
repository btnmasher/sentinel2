import { Link } from "react-router-dom";
import { Download, Upload } from "lucide-react";
import { useIntelStore } from "@/features/intel";
import HoverCard from "@/components/HoverCard";

export default function UploaderCountBadge() {
  const uploaders = useIntelStore((state) => state.uploaders);
  const hasActiveUploaders = uploaders > 0;

  return (
    <HoverCard
      trigger={
        <button
          type="button"
          className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content transition-colors hover:bg-base-300/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          aria-label="Active intel uploader count"
        >
          <span
            className={`intel-badge-icon-bg inline-flex h-6 w-6 items-center justify-center rounded-full ${
              hasActiveUploaders ? "" : "intel-status-icon--alert"
            }`}
          >
            <Upload
              className={`h-3.5 w-3.5 ${
                hasActiveUploaders
                  ? "intel-status-text-active"
                  : "intel-status-text-stale"
              }`}
            />
          </span>
          <span>{uploaders}</span>
        </button>
      }
      className="hover-card-surface intel-badge-hover-card w-72 p-3 text-xs"
    >
      {hasActiveUploaders ? (
        <>
          <p className="text-sm font-semibold text-emerald-400">
            Uploaders online
          </p>
          <p className="mt-1 text-base-content/80">
            {uploaders} uploader{uploaders === 1 ? "" : "s"}{" "}
            {uploaders === 1 ? "is" : "are"} currently online and contributing
            intel.
          </p>
          <p className="mt-1 text-base-content/80">
            You can join them to help keep intel fresh.
          </p>
          <Link
            to="/uploader"
            className="mt-3 inline-flex items-center gap-1.5 rounded-md border border-primary/40 bg-primary/10 px-2 py-1 text-xs font-medium text-primary transition hover:bg-primary/20"
          >
            <Download className="h-3 w-3" />
            Go to download page
          </Link>
        </>
      ) : (
        <>
          <p className="intel-status-text-stale text-sm font-semibold">
            No uploaders online yet
          </p>
          <p className="mt-1 text-base-content/80">
            You can help keep intel fresh by running the uploader companion app.
          </p>
          <Link
            to="/uploader"
            className="mt-3 inline-flex items-center gap-1.5 rounded-md border border-primary/40 bg-primary/10 px-2 py-1 text-xs font-medium text-primary transition hover:bg-primary/20"
          >
            <Download className="h-3 w-3" />
            Go to download page
          </Link>
        </>
      )}
    </HoverCard>
  );
}
