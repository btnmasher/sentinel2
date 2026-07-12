import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { ESI_BASE, ESI_PERMISSION_REQUIRED } from "@/config/esi";
import { useAuthStore } from "@/app/store/authStore";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { UI_DIALOG, useUIStore } from "@/app/store/uiStore";
import { getHttpData, getHttpStatus } from "@/utils/httpError";
import type {
  Character,
  Gate,
  Jumpbridge,
  MapLayout,
  Region,
  System,
  TimerSignal,
} from "../types";

const characterRosterCacheTTL = 5 * 60 * 1000;
const characterRosterCacheInvalidatedEvent =
  "sentinel2:character-roster-cache-invalidated";

type Jumpranges = {
  enabled: boolean;
  selectedSystem?: number;
  primary?: number;
  secondary?: number;
};

type LocationEntry = {
  character_id: number | string;
  location?: number;
  system_name?: string;
  region_id?: number;
  in_space?: boolean;
};

type MapState = {
  mapScale: number;
  systemSearch?: number;
  displayJumpbridges: boolean;
  displayTimers: boolean;
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
  timerSignals: Record<number, TimerSignal>;
  route: number[];
  routeWaypointsByCharacter: Record<number, number[]>;
  lastJumpgateUpdate: number;
  characters: Character[];
  charactersCacheKey: string;
  charactersLoadedAt: number;
  visibleCharacterIds: number[];
  characterLocations: Record<number, number>;
  characterLocationSystemNames: Record<number, string>;
  characterLocationRegions: Record<number, number>;
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
  toggleTimers: () => void;
  setJumpranges: (data: Partial<Jumpranges>) => void;
  updateMapConfig: (
    data: Partial<Pick<MapState, "mapRegions" | "mapLayout">>,
  ) => void;
  setMapData: (data: {
    regions: Record<number, Region>;
    systems: Record<number, System>;
    gates: Gate[];
    jumpbridges: Jumpbridge[];
    timerSignals?: Record<number, TimerSignal>;
  }) => void;
  fetchMapTopology: () => Promise<void>;
  fetchMapOverlays: () => Promise<void>;
  fetchMapData: () => Promise<void>;
  loadCharacters: (force?: boolean) => Promise<void>;
  invalidateCharactersCache: () => void;
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
const locationFetchInFlight = new Set<number>();

const claimLocationFetchTargets = (targets: number[]) => {
  const claimed: number[] = [];
  targets.forEach((charId) => {
    if (locationFetchInFlight.has(charId)) {
      return;
    }
    locationFetchInFlight.add(charId);
    claimed.push(charId);
  });
  return claimed;
};

const releaseLocationFetchTargets = (targets: number[]) => {
  targets.forEach((charId) => {
    locationFetchInFlight.delete(charId);
  });
};

const getCharacterRosterCacheKey = () => {
  const { authBackend } = useAppConfigStore.getState();
  const { userId } = useAuthStore.getState();
  return `${authBackend}:${userId ?? ""}`;
};

const isCharacterRosterCacheFresh = (cacheKey: string, loadedAt: number) => {
  if (!cacheKey || loadedAt <= 0) {
    return false;
  }
  if (Date.now() - loadedAt > characterRosterCacheTTL) {
    return false;
  }
  return true;
};

const clearCharacterRosterState = () => ({
  characters: [] as Character[],
  visibleCharacterIds: [] as number[],
  characterLocations: {} as Record<number, number>,
  characterLocationSystemNames: {} as Record<number, string>,
  characterLocationRegions: {} as Record<number, number>,
  characterInSpace: {} as Record<number, boolean>,
  lastLocationFetchAt: {} as Record<number, number>,
});

type RouteErrorContext = {
  operation:
    | "request_route"
    | "request_route_waypoints"
    | "update_route"
    | "clear_route";
  character?: number;
  waypoints?: number[];
};

const buildRouteErrorMeta = (error: unknown, context: RouteErrorContext) => ({
  scope: "map-route",
  ...context,
  status: getHttpStatus(error),
  data: getHttpData(error),
});

const showRouteError = (
  message: unknown,
  context: RouteErrorContext,
  error?: unknown,
) => {
  const text =
    typeof message === "string"
      ? message
      : message && typeof message === "object"
        ? ((message as { message?: string }).message ?? JSON.stringify(message))
        : "Error setting route, try again";
  useUIStore.getState().setToast({
    text,
    color: "error",
    meta: buildRouteErrorMeta(error, context),
  });
};

const handleRouteError = (error: unknown, context: RouteErrorContext) => {
  const status = getHttpStatus(error);
  const data = getHttpData(error);
  if (status === 403 && data === ESI_PERMISSION_REQUIRED) {
    useUIStore.getState().setToast({
      text: "Error setting route",
      color: "error",
      meta: buildRouteErrorMeta(error, context),
    });
    useUIStore.getState().setModal(UI_DIALOG.PermissionRequired, true);
    return;
  }
  showRouteError(data || "Error finding route, try again", context, error);
};

export const useMapStore = create<MapState>()(
  persist(
    (set, get) => ({
      mapScale: 1,
      systemSearch: undefined,
      displayJumpbridges: true,
      displayTimers: true,
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
      timerSignals: {},
      route: [],
      routeWaypointsByCharacter: {},
      lastJumpgateUpdate: Date.now(),
      characters: [],
      charactersCacheKey: "",
      charactersLoadedAt: 0,
      visibleCharacterIds: [],
      characterLocations: {},
      characterLocationSystemNames: {},
      characterLocationRegions: {},
      characterInSpace: {},
      lastLocationFetchAt: {},
      favoriteCharacters: [],
      lastRouteCharacter: undefined,
      setMapScale: (scale) => set({ mapScale: scale }),
      setSystemSearch: (system) => set({ systemSearch: system }),
      setMapControls: (controls) => set({ mapControls: controls }),
      toggleJumpbridges: () =>
        set((state) => ({ displayJumpbridges: !state.displayJumpbridges })),
      toggleTimers: () =>
        set((state) => ({ displayTimers: !state.displayTimers })),
      setJumpranges: (data) =>
        set((state) => ({ jumpranges: { ...state.jumpranges, ...data } })),
      updateMapConfig: (data) => set((state) => ({ ...state, ...data })),
      setMapData: (data) =>
        set({
          regions: data.regions,
          systems: data.systems,
          gates: data.gates,
          jumpbridges: data.jumpbridges,
          timerSignals: data.timerSignals ?? {},
          lastJumpgateUpdate: Date.now(),
        }),
      fetchMapTopology: async () => {
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
          });
        } catch (error: unknown) {
          useUIStore.getState().setToast({
            text: "Error loading map data",
            color: "error",
            meta: {
              scope: "map-data",
              operation: "fetch_map_topology",
              mapRegions,
              mapLayout,
              status: getHttpStatus(error),
              data: getHttpData(error),
            },
          });
        }
      },
      fetchMapOverlays: async () => {
        const { mapRegions } = get();
        if (!mapRegions || mapRegions.length === 0) {
          return;
        }
        try {
          const response = await api.get(
            `/map/regions/${mapRegions.join("+")}/overlays`,
          );
          set({
            jumpbridges: response.data.jumpbridges ?? [],
            timerSignals: response.data.timer_signals ?? {},
            lastJumpgateUpdate: Date.now(),
          });
        } catch (error: unknown) {
          useUIStore.getState().setToast({
            text: "Error loading map overlays",
            color: "error",
            meta: {
              scope: "map-data",
              operation: "fetch_map_overlays",
              mapRegions,
              status: getHttpStatus(error),
              data: getHttpData(error),
            },
          });
        }
      },
      fetchMapData: async () => {
        await get().fetchMapTopology();
        await get().fetchMapOverlays();
      },
      loadCharacters: async (force = false) => {
        const { standaloneAuth } = useAppConfigStore.getState();
        const userId = useAuthStore.getState().userId;
        const cacheKey = getCharacterRosterCacheKey();
        const currentState = get();

        if (!force && currentState.charactersCacheKey === cacheKey) {
          if (isCharacterRosterCacheFresh(cacheKey, currentState.charactersLoadedAt)) {
            return;
          }
        }

        let ids: number[] = [];
        let names: Character[] | null = null;

        if (!standaloneAuth && !userId) {
          set({
            ...clearCharacterRosterState(),
            charactersCacheKey: cacheKey,
            charactersLoadedAt: Date.now(),
          });
          return;
        }

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
            ...clearCharacterRosterState(),
            charactersCacheKey: cacheKey,
            charactersLoadedAt: Date.now(),
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
            charactersCacheKey: cacheKey,
            charactersLoadedAt: Date.now(),
            visibleCharacterIds: Array.from(new Set(nextVisible)),
          };
        });
      },
      invalidateCharactersCache: () => {
        locationFetchInFlight.clear();
        set({
          charactersCacheKey: "",
          charactersLoadedAt: 0,
          ...clearCharacterRosterState(),
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
          set({
            characterLocations: {},
            characterLocationSystemNames: {},
            characterLocationRegions: {},
            characterInSpace: {},
          });
          return;
        }
        const lastFetch = get().lastLocationFetchAt;
        const now = Date.now();
        const nextTargets = targets.filter((charId) => {
          const last = lastFetch[charId] ?? 0;
          return now - last >= 30_000;
        });
        const claimedTargets = claimLocationFetchTargets(nextTargets);
        if (claimedTargets.length === 0) {
          return;
        }
        const updates: Record<number, number> = {};
        const systemNameUpdates: Record<number, string> = {};
        const regionUpdates: Record<number, number> = {};
        const inSpaceUpdates: Record<number, boolean> = {};
        const fetchUpdates: Record<number, number> = {};
        try {
          const response = await api.post("/map/locations", {
            characters: claimedTargets,
          });
          const locations = response.data.locations || [];
          locations.forEach((entry: LocationEntry) => {
            const charId = Number(entry.character_id);
            if (!charId) return;
            const location = entry.location;
            const inSpace = entry.in_space !== false;
            inSpaceUpdates[charId] = inSpace;
            if (typeof location === "number") {
              updates[charId] = location;
            }
            if (typeof entry.system_name === "string" && entry.system_name.trim()) {
              systemNameUpdates[charId] = entry.system_name;
            }
            if (typeof entry.region_id === "number" && entry.region_id > 0) {
              regionUpdates[charId] = entry.region_id;
            }
            fetchUpdates[charId] = now;
          });
        } catch {
          return;
        } finally {
          releaseLocationFetchTargets(claimedTargets);
        }
        set((state) => ({
          characterLocations: { ...state.characterLocations, ...updates },
          characterLocationSystemNames: {
            ...state.characterLocationSystemNames,
            ...systemNameUpdates,
          },
          characterLocationRegions: {
            ...state.characterLocationRegions,
            ...regionUpdates,
          },
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
          await get().refreshCharacterLocations([character]);
          await get().updateRoute(character);
          routePolling = window.setInterval(() => {
            void get().updateRoute(character);
          }, 30000);
        } catch (error) {
          handleRouteError(error, {
            operation: "request_route_waypoints",
            character,
            waypoints,
          });
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
          const cachedLocation = get().characterLocations[character];
          if (!cachedLocation) {
            return;
          }
          const location = cachedLocation;
          let route = get().route;
          const index = route.indexOf(location);

          if (index >= 0) {
            route = route.slice(index);
            if (route.length <= 1) {
              await get().clearRoute(character);
              return;
            }
            set((state) => {
              const currentWaypoints =
                state.routeWaypointsByCharacter[character] ?? [];
              const nextWaypoints = currentWaypoints.filter((waypoint) => {
                const waypointIndex = route.indexOf(waypoint);
                return waypointIndex > 0;
              });
              const waypointsChanged =
                nextWaypoints.length !== currentWaypoints.length;
              return {
                route,
                lastRouteCharacter: character,
                routeWaypointsByCharacter: waypointsChanged
                  ? {
                      ...state.routeWaypointsByCharacter,
                      [character]: nextWaypoints,
                    }
                  : state.routeWaypointsByCharacter,
              };
            });
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
          showRouteError(
            "Error updating route",
            { operation: "update_route", character },
            error,
          );
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
          showRouteError(
            "Error clearing route, try again",
            { operation: "clear_route", character: target },
            error,
          );
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
        displayTimers: state.displayTimers,
        jumpranges: state.jumpranges,
        favoriteCharacters: state.favoriteCharacters,
        characters: state.characters,
        charactersCacheKey: state.charactersCacheKey,
        charactersLoadedAt: state.charactersLoadedAt,
        visibleCharacterIds: state.visibleCharacterIds,
        characterLocations: state.characterLocations,
        characterLocationSystemNames: state.characterLocationSystemNames,
        characterLocationRegions: state.characterLocationRegions,
        characterInSpace: state.characterInSpace,
        lastLocationFetchAt: state.lastLocationFetchAt,
        route: state.route,
        lastRouteCharacter: state.lastRouteCharacter,
        routeWaypointsByCharacter: state.routeWaypointsByCharacter,
      }),
    },
  ),
);

if (typeof window !== "undefined") {
  const globalWindow = window as Window & {
    __sentinel2CharacterRosterCacheListenerInstalled__?: boolean;
  };
  if (!globalWindow.__sentinel2CharacterRosterCacheListenerInstalled__) {
    globalWindow.__sentinel2CharacterRosterCacheListenerInstalled__ = true;
    window.addEventListener(characterRosterCacheInvalidatedEvent, () => {
      useMapStore.getState().invalidateCharactersCache();
    });
  }
}

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
