import { Link } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { Download, Upload } from "lucide-react";
import { useIntelStore } from "@/features/intel";
import HoverCardPortal from "@/components/HoverCardPortal";

export default function UploaderCountBadge() {
  const uploaders = useIntelStore((state) => state.uploaders);
  const hasActiveUploaders = uploaders > 0;
  const [open, setOpen] = useState(false);
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const closeTimerRef = useRef<number | null>(null);

  const showCard = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
    setOpen(true);
  };

  const hideCard = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current);
    }
    closeTimerRef.current = window.setTimeout(() => {
      setOpen(false);
      closeTimerRef.current = null;
    }, 90);
  };

  useEffect(
    () => () => {
      if (closeTimerRef.current) {
        window.clearTimeout(closeTimerRef.current);
      }
    },
    [],
  );

  return (
    <div>
      <button
        ref={buttonRef}
        type="button"
        className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content transition-colors hover:bg-base-300/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
        aria-label="Active intel uploader count"
        onMouseEnter={showCard}
        onMouseLeave={hideCard}
        onFocus={showCard}
        onBlur={hideCard}
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
      <HoverCardPortal
        anchorRef={buttonRef}
        open={open}
        onMouseEnter={showCard}
        onMouseLeave={hideCard}
        className="w-72 rounded-xl border border-base-content/15 bg-base-100/95 p-3 text-xs text-base-content shadow-xl backdrop-blur-sm"
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
              You can help keep intel fresh by running the uploader companion
              app.
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
      </HoverCardPortal>
    </div>
  );
}
