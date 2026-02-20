import useConfirm from "@/app/hooks/useConfirm";
import Panel from "@/components/Panel";
import { useAdminMapDataStore } from "../store/adminMapDataStore";

export default function MapDataSection() {
  const requestConfirm = useConfirm();
  const loadingLabel = useAdminMapDataStore((s) => s.loadingLabel);
  const runAction = useAdminMapDataStore((s) => s.runAction);

  return (
    <Panel
      title="Map Data"
      hint="Run map data workflows in the background. Use the full update to refresh everything."
      bodyClassName="space-y-4"
    >
      <div className="flex flex-wrap gap-2">
        <button
          className="btn btn-xs btn-success btn-outline"
          onClick={() =>
            requestConfirm({
              title: "Run Map Data Update",
              body: "Start the full map data update?",
              onConfirm: () => runAction("Full update", "/admin/map-data/run"),
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
            runAction("Build real positions", "/admin/map-data/real-positions")
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
          onClick={() => runAction("Import planets", "/admin/map-data/planets")}
          disabled={loadingLabel !== null}
        >
          Import planets
        </button>
        <button
          className="btn btn-xs btn-info btn-outline"
          onClick={() => runAction("Import moons", "/admin/map-data/moons")}
          disabled={loadingLabel !== null}
        >
          Import moons
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
    </Panel>
  );
}
