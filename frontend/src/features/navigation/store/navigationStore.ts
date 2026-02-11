import { create } from "zustand";
import { api } from "@/config/api";
import { ESI_PERMISSION_REQUIRED } from "@/config/esi";
import { useUIStore } from "@/app/store/uiStore";
import type { SystemSearch } from "../types";

type NavigationState = {
  waypoints: SystemSearch[];
  avoid: SystemSearch[];
  waypointInput: string;
  avoidInput: string;
  waypointSuggestions: SystemSearch[];
  avoidSuggestions: SystemSearch[];
  waypointLoading: boolean;
  avoidLoading: boolean;
  waypointCursor: number;
  avoidCursor: number;
  character: number | "";
  route: number[];
  topRoutes: SystemSearch[];
  topRoutesFetchedAt?: number;
  setWaypointInput: (value: string) => void;
  setAvoidInput: (value: string) => void;
  setWaypointSuggestions: (items: SystemSearch[]) => void;
  setAvoidSuggestions: (items: SystemSearch[]) => void;
  setWaypointLoading: (value: boolean) => void;
  setAvoidLoading: (value: boolean) => void;
  setWaypointCursor: (value: number) => void;
  setAvoidCursor: (value: number) => void;
  setCharacter: (value: number | "") => void;
  addWaypoint: (system: SystemSearch) => void;
  addAvoid: (system: SystemSearch) => void;
  removeWaypoint: (id: number) => void;
  removeAvoid: (id: number) => void;
  resolveSystems: (value: string) => Promise<SystemSearch[]>;
  searchSystems: (value: string) => Promise<SystemSearch[]>;
  submitWaypointInput: () => Promise<void>;
  submitAvoidInput: () => Promise<void>;
  pasteWaypoints: (text: string) => Promise<void>;
  pasteAvoids: (text: string) => Promise<void>;
  requestRoute: () => Promise<void>;
  clearRoute: () => Promise<void>;
  loadTopRoutes: (opts?: { force?: boolean }) => Promise<void>;
};

export const useNavigationStore = create<NavigationState>((set, get) => ({
  waypoints: [],
  avoid: [],
  waypointInput: "",
  avoidInput: "",
  waypointSuggestions: [],
  avoidSuggestions: [],
  waypointLoading: false,
  avoidLoading: false,
  waypointCursor: -1,
  avoidCursor: -1,
  character: "",
  route: [],
  topRoutes: [],
  topRoutesFetchedAt: undefined,
  setWaypointInput: (value) => set({ waypointInput: value }),
  setAvoidInput: (value) => set({ avoidInput: value }),
  setWaypointSuggestions: (items) => set({ waypointSuggestions: items }),
  setAvoidSuggestions: (items) => set({ avoidSuggestions: items }),
  setWaypointLoading: (value) => set({ waypointLoading: value }),
  setAvoidLoading: (value) => set({ avoidLoading: value }),
  setWaypointCursor: (value) => set({ waypointCursor: value }),
  setAvoidCursor: (value) => set({ avoidCursor: value }),
  setCharacter: (value) => set({ character: value }),
  addWaypoint: (system) =>
    set((state) => {
      if (state.waypoints.some((item) => item.id === system.id)) {
        return state;
      }
      return { waypoints: [...state.waypoints, system] };
    }),
  addAvoid: (system) =>
    set((state) => {
      if (state.avoid.some((item) => item.id === system.id)) {
        return state;
      }
      return { avoid: [...state.avoid, system] };
    }),
  removeWaypoint: (id) =>
    set((state) => ({
      waypoints: state.waypoints.filter((item) => item.id !== id),
    })),
  removeAvoid: (id) =>
    set((state) => ({
      avoid: state.avoid.filter((item) => item.id !== id),
    })),
  resolveSystems: async (value) => {
    const tokens = value
      .split(",")
      .map((v) => v.trim())
      .filter(Boolean);
    if (tokens.length === 0) return [];
    try {
      const res = await api.get(
        `/map/search?systems=${encodeURIComponent(tokens.join(","))}`,
      );
      return (res.data.systems || []) as SystemSearch[];
    } catch {
      return [];
    }
  },
  searchSystems: async (value) => {
    if (value.trim().length < 2) return [];
    try {
      const res = await api.get(`/map/search?q=${encodeURIComponent(value)}`);
      return (res.data.systems || []) as SystemSearch[];
    } catch {
      return [];
    }
  },
  submitWaypointInput: async () => {
    const { waypointInput, resolveSystems, addWaypoint } = get();
    const systems = await resolveSystems(waypointInput);
    systems.forEach(addWaypoint);
    set({ waypointInput: "", waypointSuggestions: [] });
  },
  submitAvoidInput: async () => {
    const { avoidInput, resolveSystems, addAvoid } = get();
    const systems = await resolveSystems(avoidInput);
    systems.forEach(addAvoid);
    set({ avoidInput: "", avoidSuggestions: [] });
  },
  pasteWaypoints: async (text) => {
    const { resolveSystems, addWaypoint } = get();
    const systems = await resolveSystems(text);
    systems.forEach(addWaypoint);
    set({ waypointInput: "", waypointSuggestions: [] });
  },
  pasteAvoids: async (text) => {
    const { resolveSystems, addAvoid } = get();
    const systems = await resolveSystems(text);
    systems.forEach(addAvoid);
    set({ avoidInput: "", avoidSuggestions: [] });
  },
  requestRoute: async () => {
    const { character, waypoints, avoid } = get();
    if (!character) return;
    try {
      const res = await api.post(`/map/route/${character}`, {
        waypoints: waypoints.map((item) => item.id),
        avoid: avoid.map((item) => item.id),
      });
      set({ route: res.data.route || [] });
    } catch (error: any) {
      const response = error?.response;
      if (
        response?.status === 403 &&
        response?.data === ESI_PERMISSION_REQUIRED
      ) {
        useUIStore
          .getState()
          .setToast({ text: "Error setting route", color: "error" });
        useUIStore.getState().setDialog("permissionRequired", true);
        return;
      }
      useUIStore.getState().setToast({
        text: response?.data || "Error setting route",
        color: "error",
      });
    }
  },
  clearRoute: async () => {
    const { character } = get();
    if (!character) return;
    try {
      await api.delete(`/map/route/${character}`);
      set({ route: [] });
    } catch {
      useUIStore
        .getState()
        .setToast({ text: "Error clearing route", color: "error" });
    }
  },
  loadTopRoutes: async (opts) => {
    const force = opts?.force ?? false;
    const lastFetched = get().topRoutesFetchedAt ?? 0;
    if (!force && Date.now() - lastFetched < 2 * 60 * 1000) {
      return;
    }
    try {
      const res = await api.get("/map/top-routes");
      set({ topRoutes: res.data.routes || [], topRoutesFetchedAt: Date.now() });
    } catch {
      set({ topRoutes: [], topRoutesFetchedAt: Date.now() });
    }
  },
}));
