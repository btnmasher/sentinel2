import { useEffect } from "react";
import { useNavigationStore } from "../store/navigationStore";

export default function NavigationTopRoutesCard() {
  const topRoutes = useNavigationStore((s) => s.topRoutes);
  const addWaypoint = useNavigationStore((s) => s.addWaypoint);
  const loadTopRoutes = useNavigationStore((s) => s.loadTopRoutes);

  useEffect(() => {
    void loadTopRoutes();
  }, [loadTopRoutes]);

  return (
    <div className="card bg-base-200/70 border border-slate-800">
      <div className="card-body">
        <h3 className="font-display text-lg">Top Routes</h3>
        <div className="space-y-2 text-xs text-slate-300">
          {topRoutes.length === 0 && (
            <div className="text-slate-500">No top routes yet</div>
          )}
          {topRoutes.map((route) => (
            <div key={route.id} className="flex items-center justify-between">
              <span>{route.name}</span>
              <button
                className="btn btn-xs btn-outline"
                onClick={() => addWaypoint(route)}
              >
                Use
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
