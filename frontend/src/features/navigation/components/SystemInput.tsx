import type { SystemSearch } from "../types";

type SystemInputProps = {
  placeholder: string;
  value: string;
  loading: boolean;
  cursor: number;
  suggestions: SystemSearch[];
  onChange: (value: string) => void;
  onCursorChange: (value: number) => void;
  onSelectSuggestion: (index: number) => void;
  onSubmit: () => void;
  onClearSuggestions: () => void;
  onPasteSystems: (text: string) => void;
  badgeClassName?: string;
};

export default function SystemInput({
  placeholder,
  value,
  loading,
  cursor,
  suggestions,
  onChange,
  onCursorChange,
  onSelectSuggestion,
  onSubmit,
  onClearSuggestions,
  onPasteSystems,
  badgeClassName,
}: SystemInputProps) {
  return (
    <div className="space-y-2 text-xs text-slate-300">
      <input
        className="input input-bordered input-xs bg-base-300"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "ArrowDown") {
            e.preventDefault();
            if (suggestions.length > 0) {
              onCursorChange(Math.min(cursor + 1, suggestions.length - 1));
            }
            return;
          }
          if (e.key === "ArrowUp") {
            e.preventDefault();
            if (suggestions.length > 0) {
              onCursorChange(Math.max(cursor - 1, 0));
            }
            return;
          }
          if (e.key === "Escape") {
            if (suggestions.length > 0) {
              e.preventDefault();
              onClearSuggestions();
              return;
            }
          }
          if (e.key === "Enter") {
            e.preventDefault();
            if (cursor >= 0) {
              onSelectSuggestion(cursor);
              return;
            }
            onSubmit();
          }
        }}
        onPaste={(e) => {
          const text = e.clipboardData.getData("text");
          if (text) {
            e.preventDefault();
            onPasteSystems(text);
          }
        }}
      />
      {loading && <div className="text-[10px] text-slate-500">Searching...</div>}
      {suggestions.length > 0 && (
        <div
          className={`border border-slate-800 bg-base-300/80 rounded-md max-h-40 overflow-auto ${
            badgeClassName ?? ""
          }`}
        >
          {suggestions.map((system, index) => (
            <button
              key={`${system.id}-${system.name}`}
              className={`w-full text-left px-2 py-1 hover:bg-base-200 ${
                index === cursor ? "bg-base-200" : ""
              }`}
              onClick={() => {
                onSelectSuggestion(index);
              }}
            >
              <div className="text-xs text-slate-200">{system.name}</div>
              {system.region && (
                <div className="text-[10px] text-slate-500">
                  {system.region}
                </div>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
