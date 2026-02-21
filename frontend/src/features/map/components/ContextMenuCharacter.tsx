import {
  ContextMenuItem,
  ContextMenuList,
  ContextMenuTitle,
} from "./ContextMenuUI";

export default function ContextMenuCharacter({
  character,
  characterId,
}: {
  character: string;
  characterId: number;
}) {
  const zkill = `https://zkillboard.com/character/${characterId}`;
  const eveWho = `https://evewho.com/characters/${characterId}`;

  return (
    <ContextMenuList>
      <ContextMenuTitle>{character}</ContextMenuTitle>
      <ContextMenuItem href={zkill} target="_blank" rel="noreferrer">
        zKillboard
      </ContextMenuItem>
      <ContextMenuItem href={eveWho} target="_blank" rel="noreferrer">
        Eve Who
      </ContextMenuItem>
    </ContextMenuList>
  );
}
