import { LocateFixed, ZoomIn, ZoomOut } from "lucide-react";

import { useMapStore } from "../store/mapStore";

export default function MapZoomControls() {
  const mapControls = useMapStore((s) => s.mapControls);

  return (
    <div className="flex items-center gap-1">
      <button
        className="btn btn-xs btn-square btn-ghost"
        onClick={() => mapControls.zoomOut?.()}
        aria-label="Zoom out"
      >
        <ZoomOut className="h-4 w-4" />
      </button>
      <button
        className="btn btn-xs btn-square btn-ghost"
        onClick={() => mapControls.zoomIn?.()}
        aria-label="Zoom in"
      >
        <ZoomIn className="h-4 w-4" />
      </button>
      <button
        className="btn btn-xs btn-square btn-ghost"
        onClick={() => mapControls.fit?.()}
        aria-label="Fit map"
      >
        <LocateFixed className="h-4 w-4" />
      </button>
    </div>
  );
}
