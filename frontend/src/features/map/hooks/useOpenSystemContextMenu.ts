import { useCallback } from "react";
import type { MouseEvent as ReactMouseEvent } from "react";
import { useUIStore } from "@/app/store/uiStore";
import { useMapStore } from "../store/mapStore";

export function useOpenSystemContextMenu() {
  const setContextMenu = useUIStore((s) => s.setContextMenu);
  const systems = useMapStore((s) => s.systems);

  return useCallback(
    (event: ReactMouseEvent<Element>, systemId: number, fallbackText?: string) => {
      event.preventDefault();
      event.stopPropagation();
      const rect = event.currentTarget.getBoundingClientRect();
      const anchorRect = {
        left: rect.left,
        top: rect.top,
        width: rect.width,
        height: rect.height,
      };

      if (systems[systemId]) {
        setContextMenu({
          x: event.clientX,
          y: event.clientY,
          anchorRect,
          type: "system",
          systemId,
        });
        return;
      }

      if (fallbackText) {
        setContextMenu({
          x: event.clientX,
          y: event.clientY,
          anchorRect,
          type: "text",
          text: fallbackText,
        });
      }
    },
    [setContextMenu, systems],
  );
}
