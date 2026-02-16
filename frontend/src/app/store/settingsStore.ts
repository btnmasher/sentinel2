import { create } from "zustand";
import { persist } from "zustand/middleware";
import { ensurePersistReset } from "./persistReset";

export const SETTINGS_STORE_VERSION = 2;

export type IntelThreatTimings = {
  flash: number;
  red: number;
  orange: number;
  yellow: number;
  green: number;
};

export type Settings = {
  version: number;
  introduction: boolean;
  theme: "sentinel" | "sentinel-light";
  intel: {
    panelOpen: boolean;
    filtersOpen: boolean;
    charactersOpen: boolean;
    feedOpen: boolean;
    threatTimings: IntelThreatTimings;
  };
  map: {
    invertZoom: boolean;
    alwaysShowSystems: boolean;
    viewMode: "full" | "panel";
    navigationCharacterId?: number;
  };
  alarm: {
    enabled: boolean;
    volume: number;
    sound: string;
  };
};

type SettingsState = {
  settings: Settings;
  toggle: (group: keyof Settings, name: string) => void;
  apply: (group: keyof Settings, name: string, value: unknown) => void;
  setTheme: (value: Settings["theme"]) => void;
  setIntroduction: (value: boolean) => void;
  reset: () => void;
};

const defaultSettings: Settings = {
  version: SETTINGS_STORE_VERSION,
  introduction: true,
  theme: "sentinel",
  intel: {
    panelOpen: true,
    filtersOpen: true,
    charactersOpen: true,
    feedOpen: true,
    threatTimings: {
      flash: 15,
      red: 30,
      orange: 60,
      yellow: 300,
      green: 300,
    },
  },
  map: {
    invertZoom: false,
    alwaysShowSystems: false,
    viewMode: "full",
    navigationCharacterId: undefined,
  },
  alarm: {
    enabled: true,
    volume: 10,
    sound: "woop",
  },
};

ensurePersistReset();

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      settings: defaultSettings,
      toggle: (group, name) =>
        set((state) => {
          const current = state.settings[group] as Record<string, unknown>;
          if (current && typeof current[name] === "boolean") {
            return {
              settings: {
                ...state.settings,
                [group]: { ...current, [name]: !current[name] },
              },
            };
          }
          return state;
        }),
      apply: (group, name, value) =>
        set((state) => ({
          settings: {
            ...state.settings,
            [group]: {
              ...(state.settings[group] as Record<string, unknown>),
              [name]: value,
            },
          },
        })),
      setTheme: (value) =>
        set((state) => ({
          settings: { ...state.settings, theme: value },
        })),
      setIntroduction: (value) =>
        set((state) => ({
          settings: { ...state.settings, introduction: value },
        })),
      reset: () => set({ settings: defaultSettings }),
    }),
    {
      name: "intel-map-config/settings",
      version: SETTINGS_STORE_VERSION,
      migrate: (persisted: unknown) => {
        if (!persisted) return { settings: defaultSettings };
        const persistedRecord =
          typeof persisted === "object" && persisted !== null
            ? (persisted as Record<string, unknown>)
            : {};
        const persistedSettingsRaw =
          (persistedRecord.settings as Record<string, unknown> | undefined) ??
          persistedRecord;
        const persistedSettings = persistedSettingsRaw as Partial<Settings> & {
          intel?: Partial<Settings["intel"]>;
          map?: Partial<Settings["map"]>;
          alarm?: Partial<Settings["alarm"]>;
        };
        const mergedSettings: Settings = {
          ...defaultSettings,
          ...persistedSettings,
          intel: {
            ...defaultSettings.intel,
            ...(persistedSettings?.intel ?? {}),
          },
          map: {
            ...defaultSettings.map,
            ...(persistedSettings?.map ?? {}),
          },
          alarm: {
            ...defaultSettings.alarm,
            ...(persistedSettings?.alarm ?? {}),
          },
          version: SETTINGS_STORE_VERSION,
        };
        const legacyIntel = (persistedSettings?.intel ?? {}) as Partial<
          Settings["intel"]
        > & {
          flashEnabled?: boolean;
          flashSeconds?: number;
          fadeEnabled?: boolean;
          fadeSeconds?: number;
        };
        const legacyFlash = legacyIntel.flashEnabled
          ? Number(
              legacyIntel.flashSeconds ??
                defaultSettings.intel.threatTimings.flash,
            )
          : 0;
        const legacyFade = legacyIntel.fadeEnabled
          ? Number(legacyIntel.fadeSeconds ?? 0)
          : 0;
        const quarterFade = Math.max(0, Math.floor(legacyFade / 4));
        mergedSettings.intel.threatTimings = {
          flash: Math.max(
            0,
            Number(
              mergedSettings.intel.threatTimings?.flash ??
                (legacyFlash || defaultSettings.intel.threatTimings.flash),
            ),
          ),
          red: Math.max(
            0,
            Number(
              mergedSettings.intel.threatTimings?.red ??
                (quarterFade || defaultSettings.intel.threatTimings.red),
            ),
          ),
          orange: Math.max(
            0,
            Number(
              mergedSettings.intel.threatTimings?.orange ??
                (quarterFade || defaultSettings.intel.threatTimings.orange),
            ),
          ),
          yellow: Math.max(
            0,
            Number(
              mergedSettings.intel.threatTimings?.yellow ??
                (quarterFade || defaultSettings.intel.threatTimings.yellow),
            ),
          ),
          green: Math.max(
            0,
            Number(
              mergedSettings.intel.threatTimings?.green ??
                (quarterFade || defaultSettings.intel.threatTimings.green),
            ),
          ),
        };
        return { settings: mergedSettings };
      },
    },
  ),
);
