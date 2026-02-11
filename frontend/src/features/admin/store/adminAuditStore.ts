import { create } from "zustand";
import { api } from "@/config/api";
import type { AuditEntry } from "../types";

type AuditFilters = {
  action: string;
  actor: string;
  summary: string;
};

type AdminAuditState = {
  entries: AuditEntry[];
  loading: boolean;
  page: number;
  hasMore: boolean;
  action: string;
  actor: string;
  summary: string;
  setAction: (value: string) => void;
  setActor: (value: string) => void;
  setSummary: (value: string) => void;
  setPage: (value: number) => void;
  clear: () => void;
  fetchAudit: (userId: string, page?: number, overrides?: Partial<AuditFilters>) => Promise<void>;
};

const initialState = {
  entries: [],
  loading: false,
  page: 1,
  hasMore: false,
  action: "all",
  actor: "",
  summary: "",
};

export const useAdminAuditStore = create<AdminAuditState>((set, get) => ({
  ...initialState,
  setAction: (value) => set({ action: value }),
  setActor: (value) => set({ actor: value }),
  setSummary: (value) => set({ summary: value }),
  setPage: (value) => set({ page: value }),
  clear: () => set({ ...initialState }),
  fetchAudit: async (userId, page = 1, overrides) => {
    const current = get();
    const actionValue = overrides?.action ?? current.action;
    const actorValue = overrides?.actor ?? current.actor;
    const summaryValue = overrides?.summary ?? current.summary;

    set({ loading: true });
    try {
      const res = await api.get("/admin/audit", {
        params: {
          user_id: userId,
          limit: 20,
          page,
          action: actionValue === "all" ? "" : actionValue,
          actor: actorValue.trim(),
          summary: summaryValue.trim(),
        },
      });
      set({
        entries: res.data.logs || [],
        hasMore: Boolean(res.data.hasMore),
        page: res.data.page || page,
        loading: false,
      });
    } catch {
      set({ entries: [], hasMore: false, loading: false });
    }
  },
}));
