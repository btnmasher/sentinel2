import { create } from "zustand";
import { api } from "@/config/api";

type AppConfigState = {
  authBackend: string;
  standaloneAuth: boolean;
  defaultRegions: string[];
  oidcPortalUrl: string;
  loaded: boolean;
  load: () => Promise<void>;
};

export const useAppConfigStore = create<AppConfigState>((set) => ({
  authBackend: "unknown",
  standaloneAuth: false,
  defaultRegions: [],
  oidcPortalUrl: "https://auth.pleaseignore.com",
  loaded: false,
  load: async () => {
    try {
      const res = await api.get("/app-config");
      set({
        authBackend: res.data.auth_backend || "unknown",
        standaloneAuth: Boolean(res.data.standalone_auth),
        defaultRegions: Array.isArray(res.data.default_regions)
          ? res.data.default_regions
          : [],
        oidcPortalUrl:
          typeof res.data.oidc_portal_url === "string"
            ? res.data.oidc_portal_url
            : "https://auth.pleaseignore.com",
        loaded: true,
      });
    } catch {
      set({ loaded: true });
    }
  },
}));
