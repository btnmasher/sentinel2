import SearchSuggestionField from "@/components/SearchSuggestionField";
import {
  useMapSearchSuggestions,
  type MapSearchSuggestion,
} from "../hooks/useMapSearchSuggestions";

export type MapSystemSearchResult = {
  id: number;
  name: string;
  region: string;
  region_id: number;
};

export type MapRegionSearchResult = {
  id: string;
  name: string;
};

type SystemSearchFieldProps = {
  query: string;
  onQueryChange: (value: string) => void;
  onSelect: (system: MapSystemSearchResult) => void;
  includeRegions?: boolean;
  onSelectRegion?: (region: MapRegionSearchResult) => void;
  selectionInputMode?: "set" | "clear" | "preserve";
  placeholder?: string;
  inputClassName?: string;
  containerClassName?: string;
  panelClassName?: string;
  minQueryLength?: number;
};

export default function SystemSearchField({
  query,
  onQueryChange,
  onSelect,
  includeRegions = false,
  onSelectRegion,
  selectionInputMode = "set",
  placeholder = includeRegions
    ? "Search systems or regions"
    : "Type system name",
  inputClassName = "input input-bordered w-full",
  containerClassName = "w-full",
  panelClassName = "max-h-44 overflow-auto rounded-lg border border-slate-700/70 bg-base-300/95 shadow-xl",
  minQueryLength = 2,
}: SystemSearchFieldProps) {
  const { loadSuggestions } = useMapSearchSuggestions({ includeRegions });

  return (
    <SearchSuggestionField<MapSearchSuggestion>
      query={query}
      onQueryChange={onQueryChange}
      onSelect={(item) => {
        if (item.kind === "region") {
          onSelectRegion?.({ id: item.id, name: item.name });
          return;
        }
        onSelect(item);
      }}
      placeholder={placeholder}
      inputClassName={inputClassName}
      containerClassName={containerClassName}
      panelClassName={panelClassName}
      minQueryLength={minQueryLength}
      selectionInputMode={selectionInputMode}
      loadSuggestions={loadSuggestions}
      getSuggestionKey={(item) =>
        item.kind === "region" ? `region-${item.id}` : `system-${item.id}`
      }
      getInputValueFromSuggestion={(item) => item.name}
      renderSuggestion={(item) =>
        item.kind === "region" ? (
          <>
            <div className="font-semibold text-slate-100">{item.name}</div>
            <div className="text-[11px] timer-region-name">Region</div>
          </>
        ) : (
          <>
            <div>{item.name}</div>
            <div className="text-[11px]">
              <span className="timer-region-name">{item.region}</span>
              <span className="text-slate-500"> &gt; </span>
              <span className="timer-system-name">System</span>
            </div>
          </>
        )
      }
    />
  );
}
