import { useMapStore } from "../store/mapStore";
import { useUIStore } from "@/app/store/uiStore";
import {
  ContextMenuItem,
  ContextMenuList,
  ContextMenuSeparator,
  ContextMenuTitle,
} from "./ContextMenuUI";

export default function RouteCharacterMenu({ systemId }: { systemId: number }) {
  const characters = useMapStore((s) => s.characters);
  const favoriteCharacters = useMapStore((s) => s.favoriteCharacters);
  const requestRoute = useMapStore((s) => s.requestRoute);
  const addRouteWaypoint = useMapStore((s) => s.addRouteWaypoint);
  const systems = useMapStore((s) => s.systems);
  const setMenu = useUIStore((s) => s.setContextMenu);
  const menu = useUIStore((s) => s.contextMenu);
  const routeMode = menu?.routeMode ?? "set";

  const favorites = characters.filter((c) => favoriteCharacters.includes(c.id));
  const nonFavorites = characters.filter(
    (c) => !favoriteCharacters.includes(c.id),
  );

  return (
    <ContextMenuList>
      <ContextMenuTitle>
        {routeMode === "add" ? "Add Waypoint" : "Set Route"} ·{" "}
        {systems[systemId]?.name ?? systemId}
      </ContextMenuTitle>
      <ContextMenuItem
        onClick={() =>
          setMenu(
            menu
              ? {
                  ...menu,
                  type: "system",
                  systemId,
                }
              : null,
          )
        }
      >
        Back
      </ContextMenuItem>
      {characters.length === 0 && (
        <ContextMenuItem muted>Loading...</ContextMenuItem>
      )}
      {favorites.map((character) => (
        <ContextMenuItem
          key={character.id}
          onClick={() => {
            if (routeMode === "add") {
              addRouteWaypoint(character.id, systemId);
            } else {
              requestRoute(character.id, systemId);
            }
            setMenu(null);
          }}
        >
          {character.name}
        </ContextMenuItem>
      ))}
      {favorites.length > 0 && nonFavorites.length > 0 && (
        <ContextMenuSeparator />
      )}
      {nonFavorites.map((character) => (
        <ContextMenuItem
          key={character.id}
          onClick={() => {
            if (routeMode === "add") {
              addRouteWaypoint(character.id, systemId);
            } else {
              requestRoute(character.id, systemId);
            }
            setMenu(null);
          }}
        >
          {character.name}
        </ContextMenuItem>
      ))}
    </ContextMenuList>
  );
}
