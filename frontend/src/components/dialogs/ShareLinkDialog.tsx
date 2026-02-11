import { useMemo, useState } from "react";
import Modal from "./Modal";
import { useUIStore } from "@/app/store/uiStore";
import { useMapStore } from "@/features/map";
import { useIntelStore } from "@/features/intel";

export default function ShareLinkDialog() {
  const dialogs = useUIStore((s) => s.dialogs);
  const setDialog = useUIStore((s) => s.setDialog);
  const setToast = useUIStore((s) => s.setToast);
  const mapLayout = useMapStore((s) => s.mapLayout);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const logFilters = useIntelStore((s) => s.logFilters);
  const [shareOptions, setShareOptions] = useState({
    mapLayout: true,
    regions: true,
    logFilters: true,
  });

  const shareUrl = useMemo(() => {
    const url = new URL(window.location.href);
    if (shareOptions.mapLayout) {
      url.searchParams.set("layout", mapLayout);
    }
    if (shareOptions.regions) {
      url.searchParams.set("regions", mapRegions.join(","));
    }
    if (shareOptions.logFilters) {
      url.searchParams.set("filters", logFilters.system.join(","));
    }
    return url.toString();
  }, [logFilters.system, mapLayout, mapRegions, shareOptions]);

  const copyShareUrl = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setToast({ text: "Link copied to clipboard" });
    } catch (err) {
      setToast({ text: "Unable to copy link", color: "error" });
    }
  };

  return (
    <Modal
      open={dialogs.shareLink}
      title="Share map"
      onClose={() => setDialog("shareLink", false)}
      actions={
        <>
          <button className="btn btn-sm btn-outline" onClick={copyShareUrl}>
            Copy
          </button>
          <button
            className="btn btn-sm btn-outline"
            onClick={() => setDialog("shareLink", false)}
          >
            Close
          </button>
        </>
      }
    >
      <div>
        <label className="label text-xs">Sharable link</label>
        <input
          className="input input-bordered input-sm w-full"
          value={shareUrl}
          readOnly
        />
      </div>
      <div>
        <h4 className="font-semibold">Choose what to share</h4>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={shareOptions.mapLayout}
            onChange={(e) =>
              setShareOptions((prev) => ({
                ...prev,
                mapLayout: e.target.checked,
              }))
            }
          />
          Map layout
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={shareOptions.regions}
            onChange={(e) =>
              setShareOptions((prev) => ({
                ...prev,
                regions: e.target.checked,
              }))
            }
          />
          Regions
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={shareOptions.logFilters}
            onChange={(e) =>
              setShareOptions((prev) => ({
                ...prev,
                logFilters: e.target.checked,
              }))
            }
          />
          Log filters
        </label>
      </div>
    </Modal>
  );
}
