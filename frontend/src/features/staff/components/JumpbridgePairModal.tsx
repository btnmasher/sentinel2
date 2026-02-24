import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Pencil, Plus, X } from "lucide-react";
import { api } from "@/config/api";
import { useUIStore } from "@/app/store/uiStore";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";

export type JumpbridgePair = {
  fromId: number;
  toId: number;
  fromName: string;
  toName: string;
};

type SearchSystem = {
  id: number;
  name: string;
  region?: string;
};

type JumpbridgePairModalProps = {
  editingPair?: JumpbridgePair | null;
  pairs: JumpbridgePair[];
  onSaved: () => Promise<void>;
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

export default function JumpbridgePairModal({
  editingPair,
  pairs,
  onSaved,
}: JumpbridgePairModalProps) {
  const { close } = useModalBody();
  const setToast = useUIStore((s) => s.setToast);
  const [fromQuery, setFromQuery] = useState(editingPair?.fromName ?? "");
  const [toQuery, setToQuery] = useState(editingPair?.toName ?? "");
  const [fromOptions, setFromOptions] = useState<SearchSystem[]>([]);
  const [toOptions, setToOptions] = useState<SearchSystem[]>([]);
  const [selectedFrom, setSelectedFrom] = useState<SearchSystem | null>(
    editingPair ? { id: editingPair.fromId, name: editingPair.fromName } : null,
  );
  const [selectedTo, setSelectedTo] = useState<SearchSystem | null>(
    editingPair ? { id: editingPair.toId, name: editingPair.toName } : null,
  );
  const [fromSearchEnabled, setFromSearchEnabled] = useState(false);
  const [toSearchEnabled, setToSearchEnabled] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const fromInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      fromInputRef.current?.focus();
      fromInputRef.current?.select();
    }, 0);
    return () => window.clearTimeout(timer);
  }, []);

  const blockedSystemIds = useMemo(() => {
    const ids = new Set<number>();
    pairs.forEach((pair) => {
      if (
        editingPair &&
        ((pair.fromId === editingPair.fromId &&
          pair.toId === editingPair.toId) ||
          (pair.fromId === editingPair.toId &&
            pair.toId === editingPair.fromId))
      ) {
        return;
      }
      ids.add(pair.fromId);
      ids.add(pair.toId);
    });
    return ids;
  }, [editingPair, pairs]);

  const searchSystems = useCallback(
    async (query: string, otherSelectedId?: number) => {
      if (query.trim().length < 2) return [];
      const res = await api.get(`/map/search?q=${encodeURIComponent(query)}`);
      const systems: SearchSystem[] = Array.isArray(res.data?.systems)
        ? res.data.systems
            .map((item: unknown) => {
              const record = item as {
                id?: number;
                name?: string;
                region?: string;
              };
              return {
                id: Number(record.id),
                name: String(record.name ?? ""),
                region: record.region ? String(record.region) : undefined,
              };
            })
            .filter((item: SearchSystem) => item.id > 0 && item.name !== "")
        : [];
      return systems.filter(
        (item) => !blockedSystemIds.has(item.id) && item.id !== otherSelectedId,
      );
    },
    [blockedSystemIds],
  );

  useEffect(() => {
    if (!fromSearchEnabled) {
      setFromOptions([]);
      return;
    }
    const q = fromQuery.trim();
    if (q.length < 2) {
      setFromOptions([]);
      return;
    }
    let cancelled = false;
    const timer = setTimeout(async () => {
      try {
        const options = await searchSystems(q, selectedTo?.id);
        if (!cancelled) setFromOptions(options);
      } catch {
        if (!cancelled) setFromOptions([]);
      }
    }, 180);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [fromQuery, fromSearchEnabled, searchSystems, selectedTo?.id]);

  useEffect(() => {
    if (!toSearchEnabled) {
      setToOptions([]);
      return;
    }
    const q = toQuery.trim();
    if (q.length < 2) {
      setToOptions([]);
      return;
    }
    let cancelled = false;
    const timer = setTimeout(async () => {
      try {
        const options = await searchSystems(q, selectedFrom?.id);
        if (!cancelled) setToOptions(options);
      } catch {
        if (!cancelled) setToOptions([]);
      }
    }, 180);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [searchSystems, selectedFrom?.id, toQuery, toSearchEnabled]);

  const onFromChange = (value: string) => {
    setFromSearchEnabled(true);
    setFromQuery(value);
    const selected =
      fromOptions.find(
        (opt) => opt.name.toLowerCase() === value.toLowerCase(),
      ) ?? null;
    setSelectedFrom(selected);
  };

  const onToChange = (value: string) => {
    setToSearchEnabled(true);
    setToQuery(value);
    const selected =
      toOptions.find((opt) => opt.name.toLowerCase() === value.toLowerCase()) ??
      null;
    setSelectedTo(selected);
  };

  const resolveSystemFromInput = async (
    value: string,
    selected: SearchSystem | null,
    otherSelectedId?: number,
  ): Promise<SearchSystem | null> => {
    const trimmed = value.trim();
    if (trimmed.length < 2) return null;
    if (selected && selected.name.toLowerCase() === trimmed.toLowerCase()) {
      return selected;
    }
    const systems = await searchSystems(trimmed, otherSelectedId);
    const exact =
      systems.find(
        (system) => system.name.toLowerCase() === trimmed.toLowerCase(),
      ) ?? null;
    return exact;
  };

  const submit = async () => {
    setSubmitting(true);
    try {
      const resolvedFrom = await resolveSystemFromInput(
        fromQuery,
        selectedFrom,
        selectedTo?.id,
      );
      const resolvedTo = await resolveSystemFromInput(
        toQuery,
        selectedTo,
        resolvedFrom?.id,
      );
      if (!resolvedFrom || !resolvedTo) {
        setToast({
          text: "Select valid From/To systems from the search results.",
          color: "error",
        });
        return;
      }
      if (resolvedFrom.id === resolvedTo.id) {
        setToast({
          text: "From and To must be different systems.",
          color: "error",
        });
        return;
      }

      if (editingPair) {
        const res = await api.post("/staff/jumpbridges/update", {
          old_from_id: editingPair.fromId,
          old_to_id: editingPair.toId,
          from_id: resolvedFrom.id,
          to_id: resolvedTo.id,
        });
        const changed = Boolean(res.data?.changed);
        setToast({
          text: changed ? "Jumpbridge pair updated." : "No jumpbridge changes.",
          color: "success",
        });
      } else {
        const res = await api.post("/staff/jumpbridges/add", {
          from_id: resolvedFrom.id,
          to_id: resolvedTo.id,
        });
        const changed = Boolean(res.data?.changed);
        setToast({
          text: changed
            ? "Jumpbridge pair added."
            : "Jumpbridge pair already exists.",
          color: "success",
        });
      }
      await onSaved();
      close();
    } catch (error: unknown) {
      const detail = getErrorDetail(error);
      setToast({
        text: `${editingPair ? "Update" : "Add"} jumpbridge failed: ${detail || "Unknown error"}`,
        color: "error",
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <p className="text-sm text-slate-400">
        Search excludes systems already linked by jumpbridges
        {editingPair ? ", except the pair being edited." : "."}
      </p>
      <div className="grid grid-cols-1 gap-2 md:grid-cols-[minmax(0,8.5rem)_auto_minmax(0,8.5rem)] md:items-end md:justify-center">
        <div className="space-y-1">
          <label className="text-xs text-slate-400">From system</label>
          <input
            ref={fromInputRef}
            autoFocus
            className="input input-xs input-bordered bg-base-300 w-full"
            list="jumpbridge-from-systems"
            value={fromQuery}
            onChange={(e) => onFromChange(e.target.value)}
            placeholder="enter system name"
          />
          <datalist id="jumpbridge-from-systems">
            {fromOptions.map((system) => (
              <option key={`from-${system.id}`} value={system.name}>
                {system.region || ""}
              </option>
            ))}
          </datalist>
        </div>
        <div className="flex items-center justify-center py-1 text-emerald-400 md:pb-2">
          <svg className="h-3 w-20" viewBox="0 0 72 12">
            <defs>
              <marker
                id="jb-modal-arrow"
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
              x2="72"
              y2="6"
              stroke="currentColor"
              strokeWidth="2"
              strokeDasharray="2 6"
              strokeOpacity="0.85"
              markerEnd="url(#jb-modal-arrow)"
              markerStart="url(#jb-modal-arrow)"
            />
          </svg>
        </div>
        <div className="space-y-1">
          <label className="text-xs text-slate-400">To system</label>
          <input
            className="input input-xs input-bordered bg-base-300 w-full"
            list="jumpbridge-to-systems"
            value={toQuery}
            onChange={(e) => onToChange(e.target.value)}
            placeholder="enter system name"
          />
          <datalist id="jumpbridge-to-systems">
            {toOptions.map((system) => (
              <option key={`to-${system.id}`} value={system.name}>
                {system.region || ""}
              </option>
            ))}
          </datalist>
        </div>
      </div>
      <div className="modal-action mt-2">
        <button
          className="btn btn-sm btn-outline"
          onClick={() => close()}
          disabled={submitting}
        >
          <X className="h-4 w-4" />
          Cancel
        </button>
        <button
          className="btn btn-sm btn-primary btn-outline"
          onClick={submit}
          disabled={
            submitting ||
            fromQuery.trim().length === 0 ||
            toQuery.trim().length === 0
          }
        >
          {editingPair ? (
            <>
              <Pencil className="h-4 w-4" />
              Save
            </>
          ) : (
            <>
              <Plus className="h-4 w-4" />
              Add
            </>
          )}
        </button>
      </div>
    </>
  );
}
