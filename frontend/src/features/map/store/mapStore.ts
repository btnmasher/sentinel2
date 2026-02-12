import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { ESI_BASE, ESI_PERMISSION_REQUIRED } from "@/config/esi";
import { useAuthStore } from "@/app/store/authStore";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { UI_DIALOG, useUIStore } from "@/app/store/uiStore";
import { ensurePersistReset } from "@/app/store/persistReset";
import type {
  Character,
  Gate,
  Jumpbridge,
  MapLayout,
  Region,
  System,
} from "../types";

type Jumpranges = {
  enabled: boolean;
  selectedSystem?: number;
  primary?: number;
  secondary?: number;
};

type MapState = {
  mapScale: number;
  systemSearch?: number;
  displayJumpbridges: boolean;
  jumpranges: Jumpranges;
  mapControls: {
    fit?: () => void;
    zoomIn?: () => void;
    zoomOut?: () => void;
  };
  mapRegions: string[];
  mapLayout: MapLayout;
  regions: Record<number, Region>;
  systems: Record<number, System>;
  gates: Gate[];
  jumpbridges: Jumpbridge[];
  route: number[];
  routeWaypointsByCharacter: Record<number, number[]>;
  lastJumpgateUpdate: number;
  characters: Character[];
  visibleCharacterIds: number[];
  characterLocations: Record<number, number>;
  characterInSpace: Record<number, boolean>;
  lastLocationFetchAt: Record<number, number>;
  favoriteCharacters: number[];
  lastRouteCharacter?: number;
  setMapScale: (scale: number) => void;
  setSystemSearch: (system?: number) => void;
  setMapControls: (controls: {
    fit?: () => void;
    zoomIn?: () => void;
    zoomOut?: () => void;
  }) => void;
  toggleJumpbridges: () => void;
  setJumpranges: (data: Partial<Jumpranges>) => void;
  updateMapConfig: (
    data: Partial<Pick<MapState, "mapRegions" | "mapLayout">>,
  ) => void;
  setMapData: (data: {
    regions: Record<number, Region>;
    systems: Record<number, System>;
    gates: Gate[];
    jumpbridges: Jumpbridge[];
  }) => void;
  fetchMapData: () => Promise<void>;
  loadCharacters: () => Promise<void>;
  setVisibleCharacters: (ids: number[]) => void;
  selectAllCharacters: () => void;
  selectNoCharacters: () => void;
  refreshCharacterLocations: (ids?: number[]) => Promise<void>;
  requestRoute: (character: number, destination: number) => Promise<void>;
  requestRouteWithWaypoints: (
    character: number,
    waypoints: number[],
  ) => Promise<void>;
  addRouteWaypoint: (character: number, destination: number) => Promise<void>;
  removeRouteWaypoint: (
    character: number,
    destination: number,
  ) => Promise<void>;
  updateRoute: (character: number) => Promise<void>;
  clearRoute: (character?: number) => Promise<void>;
  setFavoriteCharacters: (ids: number[]) => void;
};

let routePolling: number | undefined;

const showRouteError = (message: unknown) => {
  const text =
    typeof message === "string"
      ? message
      : message && typeof message === "object"
        ? ((message as { message?: string }).message ?? JSON.stringify(message))
        : "Error setting route, try again";
  useUIStore.getState().setToast({ text, color: "error" });
};

const handleRouteError = (error: any) => {
  const response = error?.response;
  if (response?.status === 403 && response?.data === ESI_PERMISSION_REQUIRED) {
    useUIStore
      .getState()
      .setToast({ text: "Error setting route", color: "error" });
    useUIStore.getState().setModal(UI_DIALOG.PermissionRequired, true);
    return;
  }
  showRouteError(response?.data || "Error finding route, try again");
};

ensurePersistReset();

export const useMapStore = create<MapState>()(
  persist(
    (set, get) => ({
      mapScale: 1,
      systemSearch: undefined,
      displayJumpbridges: true,
      jumpranges: {
        enabled: false,
        selectedSystem: undefined,
        primary: undefined,
        secondary: undefined,
      },
      mapControls: {},
      mapRegions: [],
      mapLayout: "dotlan",
      regions: {},
      systems: {},
      gates: [],
      jumpbridges: [],
      route: [],
      routeWaypointsByCharacter: {},
      lastJumpgateUpdate: Date.now(),
      characters: [],
      visibleCharacterIds: [],
      characterLocations: {},
      characterInSpace: {},
      lastLocationFetchAt: {},
      favoriteCharacters: [],
      lastRouteCharacter: undefined,
      setMapScale: (scale) => set({ mapScale: scale }),
      setSystemSearch: (system) => set({ systemSearch: system }),
      setMapControls: (controls) => set({ mapControls: controls }),
      toggleJumpbridges: () =>
        set((state) => ({ displayJumpbridges: !state.displayJumpbridges })),
      setJumpranges: (data) =>
        set((state) => ({ jumpranges: { ...state.jumpranges, ...data } })),
      updateMapConfig: (data) => set((state) => ({ ...state, ...data })),
      setMapData: (data) =>
        set({
          regions: data.regions,
          systems: data.systems,
          gates: data.gates,
          jumpbridges: data.jumpbridges,
          lastJumpgateUpdate: Date.now(),
        }),
      fetchMapData: async () => {
        const { mapRegions, mapLayout } = get();
        if (!mapRegions || mapRegions.length === 0) {
          return;
        }
        try {
          const response = await api.get(
            `/map/regions/${mapRegions.join("+")}/${mapLayout}`,
          );
          set({
            regions: response.data.regions,
            systems: response.data.systems,
            gates: response.data.gates,
            jumpbridges: response.data.jumpbridges,
            lastJumpgateUpdate: Date.now(),
          });
        } catch (error) {
          useUIStore
            .getState()
            .setToast({ text: "Error loading map data", color: "error" });
        }
      },
      loadCharacters: async () => {
        const { standaloneAuth } = useAppConfigStore.getState();
        const userId = useAuthStore.getState().userId;
        let ids: number[] = [];
        let names: Character[] | null = null;

        if (standaloneAuth && userId) {
          const records = await pb.collection("characters").getFullList({
            filter: `user = "${userId}"`,
            sort: "-is_main",
          });
          names = records
            .map((record) => ({
              id: Number(record.eve_character_id),
              name: String(record.eve_character_name ?? ""),
              is_main: Boolean(record.is_main),
            }))
            .filter((record) => record.id && record.name);
          ids = names.map((record) => record.id);
        } else {
          const response = await api.get("/map/characters");
          ids = response.data.characters || [];
        }
        if (ids.length === 0) {
          set({
            characters: [],
            visibleCharacterIds: [],
            characterLocations: {},
            characterInSpace: {},
          });
          return;
        }
        let sorted: Character[];
        if (names) {
          sorted = names.sort((a, b) => a.id - b.id);
        } else {
          const esi = await fetch(`${ESI_BASE}/v3/universe/names/`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(ids),
          });
          const data = (await esi.json()) as Character[];
          sorted = data.sort((a, b) => a.id - b.id);
        }
        set((state) => {
          const existing = new Set(state.visibleCharacterIds);
          const nextVisible =
            state.visibleCharacterIds.length === 0
              ? sorted.map((char) => char.id)
              : sorted
                  .map((char) => char.id)
                  .filter((id) => existing.has(id))
                  .concat(
                    sorted
                      .map((char) => char.id)
                      .filter((id) => !existing.has(id)),
                  );
          return {
            characters: sorted,
            visibleCharacterIds: Array.from(new Set(nextVisible)),
          };
        });
      },
      setVisibleCharacters: (ids) => set({ visibleCharacterIds: ids }),
      selectAllCharacters: () =>
        set((state) => ({
          visibleCharacterIds: state.characters.map((char) => char.id),
        })),
      selectNoCharacters: () => set({ visibleCharacterIds: [] }),
      refreshCharacterLocations: async (ids) => {
        const targets = ids && ids.length > 0 ? ids : get().visibleCharacterIds;
        if (targets.length === 0) {
          set({ characterLocations: {}, characterInSpace: {} });
          return;
        }
        const lastFetch = get().lastLocationFetchAt;
        const now = Date.now();
        const nextTargets = targets.filter((charId) => {
          const last = lastFetch[charId] ?? 0;
          return now - last >= 30_000;
        });
        if (nextTargets.length === 0) {
          return;
        }
        const updates: Record<number, number> = {};
        const inSpaceUpdates: Record<number, boolean> = {};
        const fetchUpdates: Record<number, number> = {};
        try {
          const response = await api.post("/map/locations", {
            characters: nextTargets,
          });
          const locations = response.data.locations || [];
          locations.forEach((entry: any) => {
            const charId = Number(entry.character_id);
            if (!charId) return;
            const location = entry.location;
            const inSpace = entry.in_space !== false;
            inSpaceUpdates[charId] = inSpace;
            if (typeof location === "number") {
              updates[charId] = location;
            }
            fetchUpdates[charId] = now;
          });
        } catch {
          return;
        }
        set((state) => ({
          characterLocations: { ...state.characterLocations, ...updates },
          characterInSpace: { ...state.characterInSpace, ...inSpaceUpdates },
          lastLocationFetchAt: {
            ...state.lastLocationFetchAt,
            ...fetchUpdates,
          },
        }));
      },
      requestRouteWithWaypoints: async (character, waypoints) => {
        if (routePolling) {
          window.clearInterval(routePolling);
          routePolling = undefined;
        }
        try {
          const response = await api.post(`/map/route/${character}`, {
            waypoints,
          });
          set((state) => ({
            route: response.data.route || [],
            lastRouteCharacter: character,
            routeWaypointsByCharacter: {
              ...state.routeWaypointsByCharacter,
              [character]: waypoints,
            },
          }));
          routePolling = window.setInterval(() => {
            void get().updateRoute(character);
          }, 30000);
        } catch (error) {
          handleRouteError(error);
        }
      },
      requestRoute: async (character, destination) => {
        await get().requestRouteWithWaypoints(character, [destination]);
      },
      addRouteWaypoint: async (character, destination) => {
        const current = get().routeWaypointsByCharacter[character] ?? [];
        const next = [...current, destination];
        await get().requestRouteWithWaypoints(character, next);
      },
      removeRouteWaypoint: async (character, destination) => {
        const current = get().routeWaypointsByCharacter[character] ?? [];
        const next = current.filter((id) => id !== destination);
        if (next.length === 0) {
          await get().clearRoute(character);
          return;
        }
        await get().requestRouteWithWaypoints(character, next);
      },
      updateRoute: async (character) => {
        try {
          const lastFetch = get().lastLocationFetchAt[character] ?? 0;
          const cachedLocation = get().characterLocations[character];
          const now = Date.now();
          let location = cachedLocation;

          if (!location || now - lastFetch >= 30_000) {
            const response = await api.post("/map/locations", {
              characters: [character],
            });
            const entry = response.data.locations?.[0];
            location = entry?.location;
            if (typeof entry?.character_id === "number") {
              const charId = Number(entry.character_id);
              set((state) => ({
                characterLocations: {
                  ...state.characterLocations,
                  ...(typeof entry.location === "number"
                    ? { [charId]: entry.location }
                    : {}),
                },
                characterInSpace: {
                  ...state.characterInSpace,
                  [charId]: entry.in_space !== false,
                },
                lastLocationFetchAt: {
                  ...state.lastLocationFetchAt,
                  [charId]: now,
                },
              }));
            }
          }
          let route = get().route;
          const index = route.indexOf(location);

          if (index >= 0 && index < route.length - 1) {
            route = route.slice(index);
            set({ route, lastRouteCharacter: character });
            return;
          }

          if (index < 0) {
            useUIStore.getState().setToast({
              text: "You left the route, no longer tracking your location.",
              color: "secondary",
              timeout: 5000,
            });
          }

          if (routePolling) {
            window.clearInterval(routePolling);
            routePolling = undefined;
          }
          set({ route: [], lastRouteCharacter: character });
        } catch (error) {
          showRouteError("Error updating route");
        }
      },
      clearRoute: async (character) => {
        const target = character ?? get().lastRouteCharacter;
        if (!target) return;
        try {
          await api.delete(`/map/route/${target}`);
          if (routePolling) {
            window.clearInterval(routePolling);
            routePolling = undefined;
          }
          set((state) => {
            const nextWaypoints = { ...state.routeWaypointsByCharacter };
            delete nextWaypoints[target];
            return { route: [], routeWaypointsByCharacter: nextWaypoints };
          });
        } catch (error) {
          showRouteError("Error clearing route, try again");
        }
      },
      setFavoriteCharacters: (ids) => set({ favoriteCharacters: ids }),
    }),
    {
      name: "intel-map-config/data",
      partialize: (state) => ({
        mapRegions: state.mapRegions,
        mapLayout: state.mapLayout,
        displayJumpbridges: state.displayJumpbridges,
        jumpranges: state.jumpranges,
        favoriteCharacters: state.favoriteCharacters,
        visibleCharacterIds: state.visibleCharacterIds,
        characterLocations: state.characterLocations,
        characterInSpace: state.characterInSpace,
        lastLocationFetchAt: state.lastLocationFetchAt,
        route: state.route,
        lastRouteCharacter: state.lastRouteCharacter,
        routeWaypointsByCharacter: state.routeWaypointsByCharacter,
      }),
    },
  ),
);

export const systemScale = (mapScale: number) =>
  mapScale > 1.95 ? 1.95 / mapScale : 1.25;

export const regionList = (regions: Record<number, Region>) =>
  Object.values(regions);

export const systemList = (
  systems: Record<number, System>,
  regions: Record<number, Region>,
) => Object.values(systems).filter((system) => regions[system.region]);

export const regionMap = (
  regions: Record<number, Region>,
  systems: Record<number, System>,
) => {
  const map: Record<number, { region: Region; systems: System[] }> = {};
  regionList(regions).forEach((region) => {
    map[region.region] = { region, systems: [] };
  });
  systemList(systems, regions).forEach((system) => {
    map[system.region].systems.push(system);
  });
  return map;
};
