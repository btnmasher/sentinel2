import { create } from "zustand";
import { api } from "@/config/api";
import { useUIStore } from "@/app/store/uiStore";
import { getErrorMessage } from "@/utils/httpError";

type AdminMapDataState = {
  loadingLabel: string | null;
  runAction: (label: string, path: string) => Promise<void>;
};

export const useAdminMapDataStore = create<AdminMapDataState>((set) => ({
  loadingLabel: null,
  runAction: async (label, path) => {
    const { setToast } = useUIStore.getState();
    set({ loadingLabel: label });
    try {
      const res = await api.post(path);
      setToast({
        text: `${label} started (job ${res.data?.job_id || "unknown"})`,
        color: "info",
      });
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, `Failed to start ${label}`),
        color: "error",
      });
    } finally {
      set({ loadingLabel: null });
    }
  },
}));
