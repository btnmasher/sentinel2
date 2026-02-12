import useConfirm from "@/app/hooks/useConfirm";
import { useAdminMapDataStore } from "../store/adminMapDataStore";

export default function MapDataSection() {
  const requestConfirm = useConfirm();
  const loadingLabel = useAdminMapDataStore((s) => s.loadingLabel);
  const runAction = useAdminMapDataStore((s) => s.runAction);

  return (
    <section className="card bg-base-200/70 border border-slate-800">
      <div className="card-body space-y-4">
        <h2 className="font-display text-2xl">Map Data</h2>
        <p className="text-xs text-slate-400">
          Run map data workflows in the background. Use the full update to
          refresh everything.
        </p>
        <div className="flex flex-wrap gap-2">
          <button
            className="btn btn-xs btn-success btn-outline"
            onClick={() =>
              requestConfirm({
                title: "Run Map Data Update",
                body: "Start the full map data update?",
                onConfirm: () =>
                  runAction("Full update", "/admin/map-data/run"),
                confirmLabel: "Run update",
                cancelLabel: "Cancel",
                tone: "default",
              })
            }
            disabled={loadingLabel !== null}
          >
            Run full update
          </button>
          <button
            className="btn btn-xs btn-info btn-outline"
            onClick={() => runAction("SDE import", "/admin/map-data/sde")}
            disabled={loadingLabel !== null}
          >
            SDE import
          </button>
          <button
            className="btn btn-xs btn-info btn-outline"
            onClick={() =>
              runAction(
                "Build real positions",
                "/admin/map-data/real-positions",
              )
            }
            disabled={loadingLabel !== null}
          >
            Build real positions
          </button>
          <button
            className="btn btn-xs btn-info btn-outline"
            onClick={() =>
              runAction(
                "Build eve2d positions",
                "/admin/map-data/eve2d-positions",
              )
            }
            disabled={loadingLabel !== null}
          >
            Build eve2d positions
          </button>
          <button
            className="btn btn-xs btn-info btn-outline"
            onClick={() => runAction("Dotlan import", "/admin/map-data/dotlan")}
            disabled={loadingLabel !== null}
          >
            Dotlan import
          </button>
          <button
            className="btn btn-xs btn-info btn-outline"
            onClick={() =>
              runAction(
                "Build metro positions",
                "/admin/map-data/metro-positions",
              )
            }
            disabled={loadingLabel !== null}
          >
            Build metro positions
          </button>
        </div>
        {loadingLabel && (
          <p className="text-xs text-slate-400">Running: {loadingLabel}…</p>
        )}
      </div>
    </section>
  );
}
