import { useCallback } from "react";
import { useIntelStore } from "@/features/intel";
import { useMapStore } from "../store/mapStore";
import { useOpenSystemContextMenu } from "./useOpenSystemContextMenu";

export function useSystemInteractions(systemId: number) {
  const setJumpranges = useMapStore((s) => s.setJumpranges);
  const openSystemContextMenu = useOpenSystemContextMenu();

  const handleMouseUp = useCallback(
    (event: React.MouseEvent<SVGGElement>) => {
      if (event.button !== 0) return;
      if (event.shiftKey) {
        useIntelStore.getState().toggleSystemFilter(systemId);
        return;
      }
      setJumpranges({ selectedSystem: systemId });
    },
    [setJumpranges, systemId],
  );

  const handleContextMenu = useCallback(
    (event: React.MouseEvent<SVGGElement>) => {
      openSystemContextMenu(event, systemId);
    },
    [openSystemContextMenu, systemId],
  );

  return { handleMouseUp, handleContextMenu };
}
