import { useMapStore } from "../store/mapStore";
import { JUMPRANGES } from "../types/jumpranges";
import { useUIStore } from "@/app/store/uiStore";
import {
  ContextMenuItem,
  ContextMenuList,
  ContextMenuTitle,
} from "./ContextMenuUI";

export default function SystemJumprangeMenu({
  systemId,
}: {
  systemId: number;
}) {
  const systems = useMapStore((s) => s.systems);
  const setJumpranges = useMapStore((s) => s.setJumpranges);
  const setMenu = useUIStore((s) => s.setContextMenu);
  const menu = useUIStore((s) => s.contextMenu);

  return (
    <ContextMenuList>
      <ContextMenuTitle>
        {systems[systemId]?.name ?? "Jumprange"}
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
      {Object.values(JUMPRANGES).map((range) => (
        <ContextMenuItem
          key={range.value}
          onClick={() => {
            setJumpranges({
              enabled: true,
              selectedSystem: systemId,
              primary: range.value,
              secondary: undefined,
            });
            setMenu(null);
          }}
        >
          {range.name}
        </ContextMenuItem>
      ))}
    </ContextMenuList>
  );
}
