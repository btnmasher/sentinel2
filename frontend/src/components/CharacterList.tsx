import CharacterCard from "./CharacterCard";

type Character = {
  record_id?: string;
  character_id: number;
  name: string;
  corp_id: number;
  corp_name?: string;
  alliance_id: number;
  alliance_name?: string;
  is_main: boolean;
  esi_token_valid?: boolean;
  esi_last_error?: string;
  esi_last_refresh_at?: string;
};

type CharacterListProps = {
  characters: Character[];
  emptyMessage?: string;
};

export default function CharacterList({
  characters,
  emptyMessage,
}: CharacterListProps) {
  if (!characters.length) {
    return (
      <p className="text-sm text-slate-400">
        {emptyMessage || "No linked characters yet."}
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {characters.map((character) => (
        <CharacterCard key={character.character_id} character={character} />
      ))}
    </div>
  );
}
