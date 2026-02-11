import { useMapStore } from "../store/mapStore";
import { useUIStore } from "@/app/store/uiStore";
import {
  ContextMenuChevron,
  ContextMenuItem,
  ContextMenuList,
  ContextMenuSeparator,
  ContextMenuTitle,
} from "./ContextMenuUI";

export default function SystemContextMenu({ systemId }: { systemId: number }) {
  const systems = useMapStore((s) => s.systems);
  const jumpranges = useMapStore((s) => s.jumpranges);
  const setJumpranges = useMapStore((s) => s.setJumpranges);
  const lastRouteCharacter = useMapStore((s) => s.lastRouteCharacter);
  const routeWaypointsByCharacter = useMapStore(
    (s) => s.routeWaypointsByCharacter,
  );
  const removeRouteWaypoint = useMapStore((s) => s.removeRouteWaypoint);
  const setMenu = useUIStore((s) => s.setContextMenu);
  const menu = useUIStore((s) => s.contextMenu);
  const system = systems[systemId];

  if (!system) return null;

  const dotlan = `https://evemaps.dotlan.net/system/${system.name.replace(/ /g, "_")}`;
  const zkill = `https://zkillboard.com/system/${system.system}/`;

  const hasJumpranges =
    jumpranges.primary !== undefined || jumpranges.secondary !== undefined;
  const activeRouteCharacter = lastRouteCharacter;
  const activeWaypoints =
    activeRouteCharacter !== undefined
      ? (routeWaypointsByCharacter[activeRouteCharacter] ?? [])
      : [];
  const isWaypoint = activeWaypoints.includes(systemId);

  return (
    <ContextMenuList>
      <ContextMenuTitle>{system.name}</ContextMenuTitle>
      <ContextMenuItem href={dotlan} target="_blank" rel="noreferrer">
        Dotlan
      </ContextMenuItem>
      <ContextMenuItem href={zkill} target="_blank" rel="noreferrer">
        zKillboard
      </ContextMenuItem>
      {isWaypoint && activeRouteCharacter !== undefined ? (
        <ContextMenuItem
          onClick={() => {
            void removeRouteWaypoint(activeRouteCharacter, systemId);
            setMenu(null);
          }}
        >
          Clear Waypoint
        </ContextMenuItem>
      ) : (
        <ContextMenuItem
          sub
          onClick={() =>
            setMenu(
              menu
                ? {
                    ...menu,
                    type: "route-character",
                    routeMode: "add",
                    systemId,
                  }
                : null,
            )
          }
        >
          <span>Add Waypoint</span>
          <ContextMenuChevron />
        </ContextMenuItem>
      )}
      <ContextMenuItem
        sub
        onClick={() =>
          setMenu(
            menu
              ? {
                  ...menu,
                  type: "route-character",
                  routeMode: "set",
                  systemId,
                }
              : null,
          )
        }
      >
        <span>Set Route</span>
        <ContextMenuChevron />
      </ContextMenuItem>
      <ContextMenuItem
        sub
        onClick={() =>
          setMenu(
            menu
              ? {
                  ...menu,
                  type: "system-jumprange",
                  systemId,
                }
              : null,
          )
        }
      >
        <span>Show Jumprange</span>
        <ContextMenuChevron />
      </ContextMenuItem>
      {hasJumpranges && (
        <ContextMenuItem
          onClick={() => {
            setJumpranges({
              enabled: false,
              selectedSystem: undefined,
              primary: undefined,
              secondary: undefined,
            });
            setMenu(null);
          }}
        >
          Clear Jumpranges
        </ContextMenuItem>
      )}
      <ContextMenuSeparator />
      <ContextMenuItem onClick={() => setMenu(null)}>Close</ContextMenuItem>
    </ContextMenuList>
  );
}
