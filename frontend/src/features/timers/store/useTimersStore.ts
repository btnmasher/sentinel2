import { create } from "zustand";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { useUIStore } from "@/app/store/uiStore";
import { getHttpStatus } from "@/utils/httpError";
import type { TimerRecord } from "../types";

type TimersState = {
  timers: TimerRecord[];
  loading: boolean;
  loadedAt?: number;
  realtimeConnected: boolean;
  realtimeUnsubscribe?: () => Promise<void>;
  loadTimers: (options?: { silent?: boolean }) => Promise<void>;
  startRealtime: () => Promise<void>;
  stopRealtime: () => Promise<void>;
};

export const useTimersStore = create<TimersState>((set, get) => ({
  timers: [],
  loading: false,
  loadedAt: undefined,
  realtimeConnected: false,
  realtimeUnsubscribe: undefined,
  loadTimers: async (options) => {
    const silent = Boolean(options?.silent);
    if (!silent) {
      set({ loading: true });
    }
    try {
      const response = await api.get<{ timers?: TimerRecord[] }>(
        "/timers?status=active,canceled&limit=500",
      );
      set({
        timers: response.data.timers ?? [],
        loadedAt: Date.now(),
      });
    } catch {
      if (!silent) {
        useUIStore
          .getState()
          .setToast({ text: "Failed to load timers", color: "error" });
      }
    } finally {
      if (!silent) {
        set({ loading: false });
      }
    }
  },
  startRealtime: async () => {
    if (get().realtimeUnsubscribe) {
      return;
    }

    let refreshTimeout: number | undefined;
    const queueRefresh = () => {
      if (refreshTimeout) {
        window.clearTimeout(refreshTimeout);
      }
      refreshTimeout = window.setTimeout(() => {
        void get().loadTimers({ silent: true });
      }, 150);
    };

    try {
      const unsubscribeRaw = await pb
        .collection("timers")
        .subscribe("*", (event) => {
          if (shouldRefreshForRealtimeEvent(event)) {
            queueRefresh();
          }
        });

      const stop = async () => {
        if (refreshTimeout) {
          window.clearTimeout(refreshTimeout);
          refreshTimeout = undefined;
        }
        await unsubscribeRaw();
      };

      set({
        realtimeConnected: true,
        realtimeUnsubscribe: stop,
      });
    } catch (error: unknown) {
      const status = getHttpStatus(error);
      if (status === 401 || status === 403) {
        return;
      }
    }
  },
  stopRealtime: async () => {
    const unsubscribe = get().realtimeUnsubscribe;
    set({
      realtimeConnected: false,
      realtimeUnsubscribe: undefined,
    });
    if (unsubscribe) {
      await unsubscribe().catch(() => undefined);
    }
  },
}));

function shouldRefreshForRealtimeEvent(event: {
  action?: string;
  record?: {
    expires_at?: string;
    status?: string;
  };
}): boolean {
  if (event.action === "delete") {
    return true;
  }
  if (event.action !== "create" && event.action !== "update") {
    return false;
  }

  const status = String(event.record?.status ?? "").toLowerCase();
  if (status && status !== "active" && status !== "canceled") {
    return false;
  }

  const expiresAt = String(event.record?.expires_at ?? "");
  const expiresMs = Date.parse(expiresAt);
  if (!Number.isFinite(expiresMs)) {
    return true;
  }
  return expiresMs >= Date.now() - 24 * 60 * 60 * 1000;
}
