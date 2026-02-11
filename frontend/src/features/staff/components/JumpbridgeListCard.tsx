import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { useUIStore } from "@/app/store/uiStore";

type JumpbridgePair = {
  fromId: number;
  toId: number;
  fromName: string;
  toName: string;
};

const parseJumpbridgeInput = (text: string) =>
  text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const match = line.match(
        /^(.+?)\s*(?:»|->|-->|—>|=>|→)\s*(.+?)(?:\s+-\s+.*)?$/,
      );
      if (!match) return null;
      const from = match[1].trim();
      const to = match[2].trim();
      if (!from || !to) return null;
      return `${from} --> ${to}`;
    })
    .filter((line): line is string => Boolean(line));

const chunkIds = (ids: number[], size = 40) => {
  const chunks: number[][] = [];
  for (let i = 0; i < ids.length; i += size) {
    chunks.push(ids.slice(i, i + size));
  }
  return chunks;
};

export default function JumpbridgeListCard() {
  const setToast = useUIStore((s) => s.setToast);
  const requestConfirm = useUIStore((s) => s.requestConfirm);
  const [pairs, setPairs] = useState<JumpbridgePair[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [jumpbridgeText, setJumpbridgeText] = useState("");
  const [importing, setImporting] = useState(false);

  const loadPairs = useCallback(async () => {
    setLoading(true);
    try {
      const records = await pb.collection("jumpbridges").getFullList();
      const uniquePairs = new Map<string, { from: number; to: number }>();
      const ids = new Set<number>();

      records.forEach((record) => {
        const fromId = Number(record.from_solarsystem);
        const toId = Number(record.to_solarsystem);
        if (!fromId || !toId) return;
        const [a, b] = fromId < toId ? [fromId, toId] : [toId, fromId];
        const key = `${a}-${b}`;
        if (!uniquePairs.has(key)) {
          uniquePairs.set(key, { from: a, to: b });
        }
        ids.add(a);
        ids.add(b);
      });

      const idList = Array.from(ids);
      const nameMap = new Map<number, string>();
      for (const chunk of chunkIds(idList)) {
        const filter = chunk.map((id) => `eve_id = ${id}`).join(" || ");
        const systems = await pb
          .collection("solar_systems")
          .getFullList({ filter });
        systems.forEach((system) => {
          const eveId = Number(system.eve_id);
          if (eveId) {
            nameMap.set(eveId, String(system.name ?? ""));
          }
        });
      }

      const nextPairs: JumpbridgePair[] = Array.from(uniquePairs.values()).map(
        (pair) => ({
          fromId: pair.from,
          toId: pair.to,
          fromName: nameMap.get(pair.from) || String(pair.from),
          toName: nameMap.get(pair.to) || String(pair.to),
        }),
      );
      nextPairs.sort((a, b) => a.fromName.localeCompare(b.fromName));
      setPairs(nextPairs);
    } catch {
      setPairs([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadPairs();
  }, [loadPairs]);

  const importJumpbridges = async () => {
    const parsed = parseJumpbridgeInput(jumpbridgeText);
    if (parsed.length === 0) {
      setToast({
        text: "Jumpbridge import failed: empty input.",
        color: "error",
      });
      return;
    }
    setImporting(true);
    try {
      const res = await api.post("/staff/jumpbridges/import", {
        jumpbridges: parsed.join("\n"),
      });
      setToast({
        text: "Jumpbridge import succeeded.",
        color: "success",
      });
      setJumpbridgeText("");
      setModalOpen(false);
      await loadPairs();
    } catch (error: any) {
      const detail =
        error?.response?.data?.message ||
        error?.response?.data ||
        error?.message;
      setToast({
        text: `Jumpbridge import failed: ${detail || "Unknown error"}`,
        color: "error",
      });
    } finally {
      setImporting(false);
    }
  };

  const clearJumpbridges = async () => {
    requestConfirm(
      "Clear jumpbridges?",
      "This will remove all jumpbridge pairings and rebuild system graphs.",
      async () => {
        try {
          await api.post("/staff/jumpbridges/clear");
          setToast({
            text: "Jumpbridges cleared.",
            color: "success",
          });
          await loadPairs();
        } catch (error: any) {
          const detail =
            error?.response?.data?.message ||
            error?.response?.data ||
            error?.message;
          setToast({
            text: `Jumpbridge clear failed: ${detail || "Unknown error"}`,
            color: "error",
          });
        }
      },
    );
  };

  const emptyState = useMemo(
    () => !loading && pairs.length === 0,
    [loading, pairs.length],
  );

  return (
    <section className="card bg-base-200/70 border border-slate-800">
      <div className="card-body space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="font-display text-2xl">Jumpbridges</h2>
            <p className="text-sm text-slate-400">
              Unique system pairings used for routing.
            </p>
          </div>
          <div className="flex items-center gap-2">
            {pairs.length > 0 && (
              <button
                className="btn btn-sm btn-error btn-outline"
                onClick={clearJumpbridges}
              >
                <Trash2 className="h-4 w-4" />
                Clear
              </button>
            )}
            <button
              className="btn btn-sm btn-info btn-outline"
              onClick={() => setModalOpen(true)}
            >
              <Plus className="h-4 w-4" />
              Import
            </button>
          </div>
        </div>

        <div className="space-y-2 text-sm">
          {loading && (
            <div className="text-slate-500">Loading jumpbridges…</div>
          )}
          {emptyState && (
            <div className="text-slate-500">No jumpbridges imported yet.</div>
          )}
          {pairs.map((pair) => (
            <div
              key={`${pair.fromId}-${pair.toId}`}
              className="grid grid-cols-[1fr_2fr_1fr] items-center rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2"
            >
              <span className="justify-self-end pr-4 font-semibold text-slate-100">
                {pair.fromName}
              </span>
              <div className="flex items-center justify-center">
                <svg
                  className="h-3 w-full text-emerald-400"
                  viewBox="0 0 120 12"
                >
                  <defs>
                    <marker
                      id="jb-arrow"
                      viewBox="0 0 24 24"
                      refX="18"
                      refY="12"
                      markerWidth="8"
                      markerHeight="8"
                      orient="auto-start-reverse"
                    >
                      <path
                        d="M5 12h14M13 5l7 7-7 7"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      />
                    </marker>
                  </defs>
                  <line
                    x1="0"
                    y1="6"
                    x2="120"
                    y2="6"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeDasharray="2 6"
                    strokeOpacity="0.7"
                    markerEnd="url(#jb-arrow)"
                    markerStart="url(#jb-arrow)"
                  />
                </svg>
              </div>
              <span className="justify-self-start pl-4 font-semibold text-slate-100">
                {pair.toName}
              </span>
            </div>
          ))}
        </div>
      </div>

      {modalOpen && (
        <div className="modal modal-open">
          <div className="modal-box bg-base-200 border border-slate-700 max-w-lg flex flex-col gap-3 h-[70vh]">
            <div>
              <h3 className="font-display text-lg">Import Jumpbridges</h3>
              <p className="text-sm text-slate-400">
                Paste the jumpbridge list (structure_id FROM --&gt; TO). This
                will replace existing jumpbridges.
              </p>
            </div>
            <textarea
              className="textarea textarea-bordered bg-base-300 flex-1 w-full"
              value={jumpbridgeText}
              onChange={(e) => setJumpbridgeText(e.target.value)}
            />
            <div className="modal-action">
              <button
                className="btn btn-sm btn-outline"
                onClick={() => setModalOpen(false)}
                disabled={importing}
              >
                Cancel
              </button>
              <button
                className="btn btn-sm btn-primary btn-outline"
                onClick={importJumpbridges}
                disabled={importing}
              >
                {importing ? "Importing..." : "Import"}
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
