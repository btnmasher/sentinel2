import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useUIStore } from "@/app/store/uiStore";
import MapContextMenu from "./ContextMenuMap";
import SystemContextMenu from "./ContextMenuSystem";
import MapJumprangeMenu from "./ContextMenuMapJumprange";
import SystemJumprangeMenu from "./ContextMenuSystemJumprange";
import RouteCharacterMenu from "./ContextMenuRouteCharacter";
import ContextMenuCharacterSearch from "./ContextMenuCharacterSearch";
import ContextMenuCharacter from "./ContextMenuCharacter";
import {
  ContextMenuItem,
  ContextMenuList,
  ContextMenuTitle,
} from "./ContextMenuUI";

export default function ContextMenu() {
  const menu = useUIStore((s) => s.contextMenu);
  const setMenu = useUIStore((s) => s.setContextMenu);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = useState({ x: 0, y: 0 });

  useEffect(() => {
    if (!menu) return;
    const handler = (event: MouseEvent | PointerEvent | TouchEvent) => {
      if (!menuRef.current) {
        setMenu(null);
        return;
      }
      if (!menuRef.current.contains(event.target as Node)) {
        setMenu(null);
      }
    };
    const escHandler = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMenu(null);
      }
    };
    window.addEventListener("pointerdown", handler);
    window.addEventListener("keydown", escHandler);
    return () => {
      window.removeEventListener("pointerdown", handler);
      window.removeEventListener("keydown", escHandler);
    };
  }, [setMenu, menu]);

  useLayoutEffect(() => {
    if (!menu) return;
    setPosition({ x: menu.x, y: menu.y });
  }, [menu]);

  useLayoutEffect(() => {
    if (!menu || !menuRef.current) return;
    const baseX = menu.x;
    const baseY = menu.y;
    const padding = 8;
    const rect = menuRef.current.getBoundingClientRect();
    const maxX = window.innerWidth - rect.width - padding;
    const maxY = window.innerHeight - rect.height - padding;
    const nextX = Math.min(Math.max(baseX, padding), Math.max(padding, maxX));
    const nextY = Math.min(Math.max(baseY, padding), Math.max(padding, maxY));
    setPosition((prev) =>
      prev.x === nextX && prev.y === nextY ? prev : { x: nextX, y: nextY },
    );
  }, [menu?.x, menu?.y, menu?.type]);

  if (!menu) return null;

  const content = (
    <div
      ref={menuRef}
      style={{
        position: "fixed",
        left: position.x,
        top: position.y,
        zIndex: 50,
      }}
      className="context-menu bg-base-200 border border-slate-800 rounded-lg shadow-lg min-w-[200px] max-w-[min(320px,90vw)] overflow-hidden"
      onClick={(event) => event.stopPropagation()}
    >
      {menu.type === "map" && <MapContextMenu />}
      {menu.type === "system" && menu.systemId && (
        <SystemContextMenu systemId={menu.systemId} />
      )}
      {menu.type === "map-jumprange" && <MapJumprangeMenu />}
      {menu.type === "system-jumprange" && menu.systemId && (
        <SystemJumprangeMenu systemId={menu.systemId} />
      )}
      {menu.type === "route-character" && menu.systemId && (
        <RouteCharacterMenu systemId={menu.systemId} />
      )}
      {menu.type === "character-search" && menu.text && (
        <ContextMenuCharacterSearch text={menu.text} />
      )}
      {menu.type === "character" && menu.character && menu.characterId && (
        <ContextMenuCharacter
          character={menu.character}
          characterId={menu.characterId}
        />
      )}
      {menu.type === "text" && menu.text && (
        <ContextMenuList>
          <ContextMenuTitle>Intel</ContextMenuTitle>
          <ContextMenuItem muted>{menu.text}</ContextMenuItem>
        </ContextMenuList>
      )}
    </div>
  );

  if (typeof document === "undefined") return content;
  return createPortal(content, document.body);
}
