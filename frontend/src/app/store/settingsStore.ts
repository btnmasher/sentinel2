import { create } from "zustand";
import { persist } from "zustand/middleware";
import { ensurePersistReset } from "./persistReset";

export const SETTINGS_STORE_VERSION = 1;

export type Settings = {
  version: number;
  introduction: boolean;
  theme: "sentinel" | "sentinel-light";
  intel: {
    panelOpen: boolean;
    filtersOpen: boolean;
    charactersOpen: boolean;
    feedOpen: boolean;
    flashEnabled: boolean;
    flashSeconds: number;
    fadeEnabled: boolean;
    fadeSeconds: number;
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
  apply: (group: keyof Settings, name: string, value: any) => void;
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
    flashEnabled: true,
    flashSeconds: 15,
    fadeEnabled: true,
    fadeSeconds: 300,
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
          const current = state.settings[group] as any;
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
            [group]: { ...(state.settings[group] as any), [name]: value },
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
      migrate: (persisted: any) => {
        if (!persisted) return { settings: defaultSettings };
        const persistedSettings = persisted.settings ?? persisted;
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
        return { settings: mergedSettings };
      },
    },
  ),
);
