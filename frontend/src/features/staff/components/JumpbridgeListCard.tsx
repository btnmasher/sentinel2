import { useCallback, useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import useConfirm from "@/app/hooks/useConfirm";
import useModal from "@/app/hooks/useModal";
import { useUIStore } from "@/app/store/uiStore";
import Panel from "@/components/Panel";
import JumpbridgeImportModal from "./JumpbridgeImportModal";
import JumpbridgePairModal, {
  type JumpbridgePair,
} from "./JumpbridgePairModal";

const JUMPBRIDGE_MODAL = {
  Import: "import",
  Pair: "pair",
} as const;
type JumpbridgeModalKey =
  (typeof JUMPBRIDGE_MODAL)[keyof typeof JUMPBRIDGE_MODAL];

const chunkIds = (ids: number[], size = 40) => {
  const chunks: number[][] = [];
  for (let i = 0; i < ids.length; i += size) {
    chunks.push(ids.slice(i, i + size));
  }
  return chunks;
};

const getErrorDetail = (error: unknown): string => {
  if (!error || typeof error !== "object") return "Unknown error";
  const maybeResponse = (error as { response?: unknown }).response;
  if (maybeResponse && typeof maybeResponse === "object") {
    const response = maybeResponse as { data?: unknown };
    if (response.data && typeof response.data === "object") {
      const message = (response.data as { message?: unknown }).message;
      if (typeof message === "string" && message.trim() !== "") {
        return message;
      }
    }
    if (typeof response.data === "string" && response.data.trim() !== "") {
      return response.data;
    }
  }
  const message = (error as { message?: unknown }).message;
  if (typeof message === "string" && message.trim() !== "") {
    return message;
  }
  return "Unknown error";
};

export default function JumpbridgeListCard() {
  const requestConfirm = useConfirm();
  const setToast = useUIStore((s) => s.setToast);
  const [pairs, setPairs] = useState<JumpbridgePair[]>([]);
  const [loading, setLoading] = useState(true);
  const [showImportModal, setShowImportModal] = useState(false);
  const [showPairModal, setShowPairModal] = useState(false);
  const [editingPair, setEditingPair] = useState<JumpbridgePair | null>(null);
  const setJumpbridgeModal = (modal: JumpbridgeModalKey, open: boolean) => {
    if (modal === JUMPBRIDGE_MODAL.Import) {
      setShowImportModal(open);
      if (open) {
        setShowPairModal(false);
        setEditingPair(null);
      }
      return;
    }
    setShowPairModal(open);
    if (!open) {
      setEditingPair(null);
      return;
    }
    setShowImportModal(false);
  };

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

  useModal({
    open: showImportModal,
    modalKey: JUMPBRIDGE_MODAL.Import,
    setOpenByKey: setJumpbridgeModal,
    build: () => ({
      title: "Import Jumpbridges",
      sizeClass: "max-w-lg h-[70vh] flex flex-col gap-3",
      body: <JumpbridgeImportModal onImported={loadPairs} />,
    }),
  });

  useModal({
    open: showPairModal,
    modalKey: JUMPBRIDGE_MODAL.Pair,
    setOpenByKey: setJumpbridgeModal,
    build: () => ({
      title: editingPair
        ? "Edit Jumpbridge Connection"
        : "Add Jumpbridge Connection",
      sizeClass: "max-w-lg",
      body: (
        <JumpbridgePairModal
          editingPair={editingPair}
          pairs={pairs}
          onSaved={loadPairs}
        />
      ),
    }),
  });

  const openImportModal = () => {
    setJumpbridgeModal(JUMPBRIDGE_MODAL.Import, true);
  };

  const openPairModal = (pair?: JumpbridgePair) => {
    setEditingPair(pair ?? null);
    setJumpbridgeModal(JUMPBRIDGE_MODAL.Pair, true);
  };

  const clearJumpbridges = async () => {
    requestConfirm({
      title: "Clear jumpbridges?",
      body: "This will remove all jumpbridge pairings and rebuild system graphs.",
      onConfirm: async () => {
        try {
          await api.post("/staff/jumpbridges/clear");
          setToast({
            text: "Jumpbridges cleared.",
            color: "success",
          });
          await loadPairs();
        } catch (error: unknown) {
          const detail = getErrorDetail(error);
          setToast({
            text: `Jumpbridge clear failed: ${detail || "Unknown error"}`,
            color: "error",
          });
        }
      },
      confirmLabel: "Clear",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  };

  const removePair = async (pair: JumpbridgePair) => {
    requestConfirm({
      title: "Remove jumpbridge pair?",
      body: `${pair.fromName} <-> ${pair.toName}`,
      onConfirm: async () => {
        try {
          await api.post("/staff/jumpbridges/remove", {
            from_id: pair.fromId,
            to_id: pair.toId,
          });
          setToast({
            text: "Jumpbridge pair removed.",
            color: "success",
          });
          await loadPairs();
        } catch (error: unknown) {
          const detail = getErrorDetail(error);
          setToast({
            text: `Remove jumpbridge failed: ${detail || "Unknown error"}`,
            color: "error",
          });
        }
      },
      confirmLabel: "Remove",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  };

  const emptyState = useMemo(
    () => !loading && pairs.length === 0,
    [loading, pairs.length],
  );
  const actions = (
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
        onClick={openImportModal}
      >
        <Plus className="h-4 w-4" />
        Import
      </button>
    </div>
  );

  return (
    <Panel
      title="Jumpbridges"
      hint="Unique system pairings used for routing."
      actions={actions}
      bodyClassName="space-y-4"
    >
      <div className="space-y-2 text-sm">
        {loading && <div className="text-slate-500">Loading jumpbridges…</div>}
        {emptyState && (
          <div className="text-slate-500">No jumpbridges imported yet.</div>
        )}
        {pairs.map((pair) => (
          <div
            key={`${pair.fromId}-${pair.toId}`}
            className="grid grid-cols-[1fr_2fr_1fr_auto] items-center rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2"
          >
            <span className="justify-self-end pr-4 font-semibold text-slate-100">
              {pair.fromName}
            </span>
            <div className="flex items-center justify-center">
              <svg className="h-3 w-full text-emerald-400" viewBox="0 0 120 12">
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
            <div className="ml-3 flex items-center gap-1">
              <button
                className="btn btn-xs btn-outline btn-square"
                onClick={() => openPairModal(pair)}
                aria-label={`Edit jumpbridge ${pair.fromName} to ${pair.toName}`}
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                className="btn btn-xs btn-outline btn-square btn-error"
                onClick={() => removePair(pair)}
                aria-label={`Remove jumpbridge ${pair.fromName} to ${pair.toName}`}
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          </div>
        ))}
        <button
          className="w-full rounded-lg border border-dashed border-slate-700/80 bg-base-300/20 px-3 py-2 text-slate-300 transition hover:bg-base-300/40 hover:text-slate-100"
          onClick={() => openPairModal()}
          aria-label="Add jumpbridge connection"
        >
          <span className="flex items-center justify-center gap-2 text-sm">
            <Plus className="h-4 w-4" />
            Add jumpbridge connection
          </span>
        </button>
      </div>
    </Panel>
  );
}
