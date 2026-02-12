import { create } from "zustand";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { useUIStore } from "@/app/store/uiStore";
import { getHttpStatus } from "@/utils/httpError";

type AuthState = {
  loaded: boolean;
  isAuthenticated: boolean;
  isStaff: boolean;
  isAdmin: boolean;
  userId: string | null;
  provider: string | null;
  load: () => Promise<void>;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
  syncFromPB: () => void;
  invalidate: () => void;
  forceLogout: (message?: string) => void;
};

let refreshTimer: number | null = null;
let refreshInFlight: Promise<void> | null = null;
let refreshFailures = 0;

const DEFAULT_REFRESH_MS = 12 * 60 * 1000;
const REFRESH_SKEW_MS = 2 * 60 * 1000;
const MIN_REFRESH_MS = 60 * 1000;
const MAX_BACKOFF_MS = 5 * 60 * 1000;

const clearRefreshTimer = () => {
  if (refreshTimer !== null) {
    window.clearTimeout(refreshTimer);
    refreshTimer = null;
  }
};

const decodeJwtPayload = (token: string): Record<string, unknown> | null => {
  const parts = token.split(".");
  if (parts.length < 2) return null;
  try {
    const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64.padEnd(
      base64.length + ((4 - (base64.length % 4)) % 4),
      "=",
    );
    const json = atob(padded);
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
};

const tokenExpiryMs = () => {
  const payload = decodeJwtPayload(pb.authStore.token);
  const exp = payload?.exp;
  if (typeof exp !== "number") return null;
  return exp * 1000;
};

const scheduleRefresh = () => {
  clearRefreshTimer();
  const expiry = tokenExpiryMs();
  const now = Date.now();
  let delay = DEFAULT_REFRESH_MS;
  if (expiry) {
    delay = Math.max(MIN_REFRESH_MS, expiry - now - REFRESH_SKEW_MS);
  }
  if (refreshFailures > 0) {
    const backoff = Math.min(
      MAX_BACKOFF_MS,
      30_000 * Math.pow(2, refreshFailures - 1),
    );
    delay = Math.max(delay, backoff);
  }
  refreshTimer = window.setTimeout(() => {
    void useAuthStore.getState().refresh();
  }, delay);
};

const redirectToLanding = (message: string) => {
  useUIStore.getState().setToast({
    timeout: 5000,
    color: "error",
    text: message,
  });
  if (window.location.pathname !== "/") {
    window.location.href = "/";
  }
};

const resolveAuthState = () => {
  const record = pb.authStore.model as Record<string, unknown> | null;
  const accessLevel =
    typeof record?.access_level === "string" ? record.access_level : "";
  return {
    loaded: true,
    isAuthenticated: pb.authStore.isValid,
    isStaff: accessLevel === "staff" || accessLevel === "admin",
    isAdmin: accessLevel === "admin",
    userId: typeof record?.id === "string" ? record.id : null,
    provider:
      typeof record?.auth_provider === "string" ? record.auth_provider : null,
  };
};

export const useAuthStore = create<AuthState>((set) => ({
  loaded: false,
  isAuthenticated: false,
  isStaff: false,
  isAdmin: false,
  userId: null,
  provider: null,
  load: async () => {
    set(resolveAuthState());
    if (!pb.authStore.isValid) {
      clearRefreshTimer();
      return;
    }
    await useAuthStore.getState().refresh();
  },
  refresh: async () => {
    if (!pb.authStore.isValid) {
      pb.authStore.clear();
      set(resolveAuthState());
      clearRefreshTimer();
      return;
    }
    if (refreshInFlight) {
      return refreshInFlight;
    }
    refreshInFlight = (async () => {
      try {
        const res = await api.post("/auth/refresh", null, {
          headers: { "X-Auth-Check": "1" },
        });
        if (res?.data?.token) {
          pb.authStore.save(res.data.token, res.data.record);
        }
        refreshFailures = 0;
        set(resolveAuthState());
        scheduleRefresh();
      } catch (error: unknown) {
        const status = getHttpStatus(error);
        if (status === 401 || status === 403) {
          useAuthStore.getState().forceLogout();
        } else {
          refreshFailures = Math.min(refreshFailures + 1, 6);
          set(resolveAuthState());
          scheduleRefresh();
        }
      } finally {
        refreshInFlight = null;
      }
    })();
    return refreshInFlight;
  },
  logout: async () => {
    try {
      await api.post("/auth/logout");
    } catch {
      // ignore logout errors
    }
    pb.authStore.clear();
    set(resolveAuthState());
    refreshFailures = 0;
    clearRefreshTimer();
    if (window.location.pathname !== "/") {
      window.location.href = "/";
    }
  },
  syncFromPB: () => {
    set(resolveAuthState());
    if (pb.authStore.isValid) {
      scheduleRefresh();
    } else {
      refreshFailures = 0;
      clearRefreshTimer();
    }
  },
  invalidate: () => {
    pb.authStore.clear();
    set(resolveAuthState());
    refreshFailures = 0;
    clearRefreshTimer();
  },
  forceLogout: (message = "Session expired. Please log in again.") => {
    pb.authStore.clear();
    set(resolveAuthState());
    refreshFailures = 0;
    clearRefreshTimer();
    redirectToLanding(message);
  },
}));

pb.authStore.onChange(() => {
  useAuthStore.getState().syncFromPB();
});
