import { useMapStore } from "../store/mapStore";
import { useUIStore } from "@/app/store/uiStore";
import {
  ContextMenuChevron,
  ContextMenuItem,
  ContextMenuList,
  ContextMenuSeparator,
  ContextMenuTitle,
} from "./ContextMenuUI";

export default function MapContextMenu() {
  const clearRoute = useMapStore((s) => s.clearRoute);
  const lastRouteCharacter = useMapStore((s) => s.lastRouteCharacter);
  const route = useMapStore((s) => s.route);
  const jumpranges = useMapStore((s) => s.jumpranges);
  const setJumpranges = useMapStore((s) => s.setJumpranges);
  const setMenu = useUIStore((s) => s.setContextMenu);
  const menu = useUIStore((s) => s.contextMenu);

  const hasJumpranges =
    jumpranges.primary !== undefined || jumpranges.secondary !== undefined;

  return (
    <ContextMenuList>
      <ContextMenuTitle>Map</ContextMenuTitle>
      <ContextMenuItem
        onClick={() => {
          if (lastRouteCharacter) {
            clearRoute(lastRouteCharacter);
          }
          setMenu(null);
        }}
        disabled={!lastRouteCharacter || route.length === 0}
      >
        Clear Route
      </ContextMenuItem>
      <ContextMenuItem
        sub
        onClick={() =>
          setMenu(
            menu
              ? {
                  ...menu,
                  type: "map-jumprange",
                }
              : null,
          )
        }
      >
        <span>Set Jumpranges</span>
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
