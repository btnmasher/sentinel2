import { create } from "zustand";
import { api } from "@/config/api";

type AppConfigState = {
  authBackend: string;
  standaloneAuth: boolean;
  timersEnabled: boolean;
  defaultRegions: string[];
  oidcPortalUrl: string;
  version: string;
  loaded: boolean;
  setVersion: (version: string) => void;
  load: () => Promise<void>;
};

export const useAppConfigStore = create<AppConfigState>((set) => ({
  authBackend: "unknown",
  standaloneAuth: false,
  timersEnabled: true,
  defaultRegions: [],
  oidcPortalUrl: "https://auth.pleaseignore.com",
  version: "",
  loaded: false,
  setVersion: (version) => set({ version }),
  load: async () => {
    try {
      const res = await api.get("/app-config");
      set({
        authBackend: res.data.auth_backend || "unknown",
        standaloneAuth: Boolean(res.data.standalone_auth),
        timersEnabled:
          typeof res.data.timers_enabled === "boolean"
            ? res.data.timers_enabled
            : true,
        defaultRegions: Array.isArray(res.data.default_regions)
          ? res.data.default_regions
          : [],
        oidcPortalUrl:
          typeof res.data.oidc_portal_url === "string"
            ? res.data.oidc_portal_url
            : "https://auth.pleaseignore.com",
        version: typeof res.data.version === "string" ? res.data.version : "",
        loaded: true,
      });
    } catch {
      set({ loaded: true });
    }
  },
}));
