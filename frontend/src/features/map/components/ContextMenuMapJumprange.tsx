import { useState } from "react";
import { useMapStore } from "../store/mapStore";
import { JUMPRANGES } from "../types/jumpranges";
import { useUIStore } from "@/app/store/uiStore";
import {
  ContextMenuChevron,
  ContextMenuItem,
  ContextMenuList,
  ContextMenuTitle,
} from "./ContextMenuUI";

export default function MapJumprangeMenu() {
  const [selection, setSelection] = useState<
    "primary" | "secondary" | undefined
  >(undefined);
  const jumpranges = useMapStore((s) => s.jumpranges);
  const setJumpranges = useMapStore((s) => s.setJumpranges);
  const setMenu = useUIStore((s) => s.setContextMenu);
  const menu = useUIStore((s) => s.contextMenu);

  type JumprangeOption = { name: string; value: number | undefined };
  const ranges: JumprangeOption[] = Object.values(JUMPRANGES);
  const noneRange: JumprangeOption = { name: "None", value: undefined };
  const secondaryRanges = ranges.filter(
    (range) =>
      range.value !== undefined && (jumpranges.primary ?? 0) < range.value,
  );

  return (
    <ContextMenuList>
      <ContextMenuTitle>Jumprange</ContextMenuTitle>
      <ContextMenuItem
        onClick={() =>
          setMenu(
            menu
              ? {
                  ...menu,
                  type: "map",
                }
              : null,
          )
        }
      >
        Back
      </ContextMenuItem>
      {!selection && (
        <>
          <ContextMenuItem sub onClick={() => setSelection("primary")}>
            <span>Primary range</span>
            <ContextMenuChevron />
          </ContextMenuItem>
          <ContextMenuItem sub onClick={() => setSelection("secondary")}>
            <span>Secondary range</span>
            <ContextMenuChevron />
          </ContextMenuItem>
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
            Clear jumpranges
          </ContextMenuItem>
        </>
      )}
      {selection === "primary" &&
        ranges.map((range) => (
          <ContextMenuItem
            key={range.value}
            onClick={() => {
              setJumpranges({
                enabled: true,
                primary: range.value,
                secondary: undefined,
              });
              setMenu(null);
            }}
          >
            {range.name}
          </ContextMenuItem>
        ))}
      {selection === "secondary" &&
        secondaryRanges.concat([noneRange]).map((range) => (
          <ContextMenuItem
            key={range.value ?? "none"}
            onClick={() => {
              setJumpranges({ enabled: true, secondary: range.value });
              setMenu(null);
            }}
          >
            {range.name}
          </ContextMenuItem>
        ))}
    </ContextMenuList>
  );
}
