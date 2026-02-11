import { useEffect } from "react";
import { useMapStore } from "@/features/map";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useNavigationStore } from "../store/navigationStore";
import SystemInput from "./SystemInput";
import SelectionDropdown from "@/components/SelectionDropdown";
import type { SystemSearch } from "../types";

export default function NavigationRouteControlCard() {
  const characters = useMapStore((s) => s.characters);
  const navigationCharacterId = useSettingsStore(
    (s) => s.settings.map.navigationCharacterId,
  );
  const applySetting = useSettingsStore((s) => s.apply);

  const {
    waypoints,
    avoid,
    waypointInput,
    avoidInput,
    waypointSuggestions,
    avoidSuggestions,
    waypointLoading,
    avoidLoading,
    waypointCursor,
    avoidCursor,
    character,
    route,
    setWaypointInput,
    setAvoidInput,
    setWaypointSuggestions,
    setAvoidSuggestions,
    setWaypointLoading,
    setAvoidLoading,
    setWaypointCursor,
    setAvoidCursor,
    setCharacter,
    addWaypoint,
    addAvoid,
    removeWaypoint,
    removeAvoid,
    submitWaypointInput,
    submitAvoidInput,
    pasteWaypoints,
    pasteAvoids,
    searchSystems,
    requestRoute,
    clearRoute,
  } = useNavigationStore();

  const characterOptions = characters.map((char) => ({
    id: String(char.id),
    label: char.name,
  }));

  useEffect(() => {
    if (character || characters.length === 0) return;
    const stored = navigationCharacterId;
    if (stored && characters.some((char) => char.id === stored)) {
      setCharacter(stored);
      return;
    }
    const main = characters.find((char) => char.is_main);
    const fallback = main?.id ?? characters[0]?.id;
    if (fallback) {
      setCharacter(fallback);
      applySetting("map", "navigationCharacterId", fallback);
    }
  }, [
    applySetting,
    character,
    characters,
    navigationCharacterId,
    setCharacter,
  ]);

  useEffect(() => {
    if (waypointInput.trim().length < 2) {
      setWaypointSuggestions([]);
      setWaypointLoading(false);
      return;
    }
    const handler = window.setTimeout(() => {
      setWaypointLoading(true);
      void searchSystems(waypointInput)
        .then((systems) => setWaypointSuggestions(systems))
        .finally(() => setWaypointLoading(false));
    }, 200);
    return () => window.clearTimeout(handler);
  }, [
    searchSystems,
    setWaypointLoading,
    setWaypointSuggestions,
    waypointInput,
  ]);

  useEffect(() => {
    if (avoidInput.trim().length < 2) {
      setAvoidSuggestions([]);
      setAvoidLoading(false);
      return;
    }
    const handler = window.setTimeout(() => {
      setAvoidLoading(true);
      void searchSystems(avoidInput)
        .then((systems) => setAvoidSuggestions(systems))
        .finally(() => setAvoidLoading(false));
    }, 200);
    return () => window.clearTimeout(handler);
  }, [searchSystems, setAvoidLoading, setAvoidSuggestions, avoidInput]);

  useEffect(() => {
    if (waypointSuggestions.length === 0) {
      setWaypointCursor(-1);
    } else if (waypointCursor === -1) {
      setWaypointCursor(0);
    } else if (waypointCursor >= waypointSuggestions.length) {
      setWaypointCursor(waypointSuggestions.length - 1);
    }
  }, [waypointCursor, waypointSuggestions, setWaypointCursor]);

  useEffect(() => {
    if (avoidSuggestions.length === 0) {
      setAvoidCursor(-1);
    } else if (avoidCursor === -1) {
      setAvoidCursor(0);
    } else if (avoidCursor >= avoidSuggestions.length) {
      setAvoidCursor(avoidSuggestions.length - 1);
    }
  }, [avoidCursor, avoidSuggestions, setAvoidCursor]);

  const selectWaypointSuggestion = (index: number) => {
    const system = waypointSuggestions[index];
    if (!system) return;
    addWaypoint(system);
    setWaypointInput("");
    setWaypointSuggestions([]);
  };

  const selectAvoidSuggestion = (index: number) => {
    const system = avoidSuggestions[index];
    if (!system) return;
    addAvoid(system);
    setAvoidInput("");
    setAvoidSuggestions([]);
  };

  const renderBadge = (
    system: SystemSearch,
    onRemove: (id: number) => void,
    className: string,
  ) => (
    <button
      key={system.id}
      className={`badge badge-sm ${className}`}
      onClick={() => onRemove(system.id)}
      title="Remove"
    >
      {system.name}
    </button>
  );

  return (
    <div className="card bg-base-200/70 border border-slate-800">
      <div className="card-body">
        <h3 className="font-display text-lg">Route Control</h3>
        <SelectionDropdown
          items={characterOptions}
          selected={character ? [String(character)] : []}
          onChange={(next) => {
            const nextId = next[0] ? Number(next[0]) : "";
            setCharacter(nextId);
            if (nextId) {
              applySetting("map", "navigationCharacterId", nextId);
            }
          }}
          searchable
          label="Character"
          placeholder="Select Character"
          buttonClassName="w-full"
        />

        <div className="space-y-2 text-xs text-slate-300">
          <div className="flex flex-wrap gap-2">
            {waypoints.length === 0 && (
              <span className="text-slate-500">No waypoints</span>
            )}
            {waypoints.map((item) =>
              renderBadge(item, removeWaypoint, "badge-outline"),
            )}
          </div>
          <SystemInput
            placeholder="Add waypoint (name or ID)"
            value={waypointInput}
            loading={waypointLoading}
            cursor={waypointCursor}
            suggestions={waypointSuggestions}
            onChange={setWaypointInput}
            onCursorChange={setWaypointCursor}
            onSelectSuggestion={selectWaypointSuggestion}
            onSubmit={() => void submitWaypointInput()}
            onClearSuggestions={() => setWaypointSuggestions([])}
            onPasteSystems={(text) => void pasteWaypoints(text)}
          />
        </div>

        <div className="space-y-2 text-xs text-slate-300">
          <div className="flex flex-wrap gap-2">
            {avoid.length === 0 && (
              <span className="text-slate-500">No avoids</span>
            )}
            {avoid.map((item) => renderBadge(item, removeAvoid, "badge-ghost"))}
          </div>
          <SystemInput
            placeholder="Add avoid (name or ID)"
            value={avoidInput}
            loading={avoidLoading}
            cursor={avoidCursor}
            suggestions={avoidSuggestions}
            onChange={setAvoidInput}
            onCursorChange={setAvoidCursor}
            onSelectSuggestion={selectAvoidSuggestion}
            onSubmit={() => void submitAvoidInput()}
            onClearSuggestions={() => setAvoidSuggestions([])}
            onPasteSystems={(text) => void pasteAvoids(text)}
          />
        </div>

        <div className="flex gap-2">
          <button
            className="btn btn-sm btn-info btn-outline"
            onClick={requestRoute}
          >
            Set Route
          </button>
          <button className="btn btn-outline btn-sm" onClick={clearRoute}>
            Clear
          </button>
        </div>
        {route.length > 0 && (
          <div className="text-xs text-slate-400">
            Route length: {route.length}
          </div>
        )}
      </div>
    </div>
  );
}
