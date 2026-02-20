import SelectionDropdown from "@/components/SelectionDropdown";
import {
  hostilityOptions,
  severityOptions,
  structureOptions,
  timerKindLabels,
} from "../config/timerOptions";

type Props = {
  searchQuery: string;
  setSearchQuery: (value: string) => void;
  standingFilter: string[];
  setStandingFilter: (value: string[]) => void;
  kindFilter: string[];
  setKindFilter: (value: string[]) => void;
  structureFilter: string[];
  setStructureFilter: (value: string[]) => void;
  severityFilter: string[];
  setSeverityFilter: (value: string[]) => void;
};

export default function TimerBoardFilters({
  searchQuery,
  setSearchQuery,
  standingFilter,
  setStandingFilter,
  kindFilter,
  setKindFilter,
  structureFilter,
  setStructureFilter,
  severityFilter,
  setSeverityFilter,
}: Props) {
  return (
    <div className="mb-4 grid gap-2 md:grid-cols-2 xl:grid-cols-5">
      <label className="form-control">
        <div className="label py-1">
          <span className="label-text text-xs uppercase tracking-wide">
            Search
          </span>
        </div>
        <input
          className="input input-bordered h-9 min-h-9"
          value={searchQuery}
          onChange={(event) => setSearchQuery(event.target.value)}
          placeholder="System, org, stage, title"
        />
      </label>

      <label className="form-control">
        <div className="label py-1">
          <span className="label-text text-xs uppercase tracking-wide">
            Hostility
          </span>
        </div>
        <SelectionDropdown
          items={hostilityOptions.map((option) => ({
            id: option.value,
            label: option.label,
          }))}
          selected={standingFilter}
          onChange={setStandingFilter}
          multi
          searchable
          label="Hostility"
          placeholder="All"
          buttonClassName="input input-bordered h-9 min-h-9 w-full justify-between"
          menuClassName="z-[12000]"
        />
      </label>

      <label className="form-control">
        <div className="label py-1">
          <span className="label-text text-xs uppercase tracking-wide">
            Type
          </span>
        </div>
        <SelectionDropdown
          items={Object.entries(timerKindLabels).map(([value, label]) => ({
            id: value,
            label,
          }))}
          selected={kindFilter}
          onChange={setKindFilter}
          multi
          searchable
          label="Type"
          placeholder="All"
          buttonClassName="input input-bordered h-9 min-h-9 w-full justify-between"
          menuClassName="z-[12000]"
        />
      </label>

      <label className="form-control">
        <div className="label py-1">
          <span className="label-text text-xs uppercase tracking-wide">
            Structure
          </span>
        </div>
        <SelectionDropdown
          items={structureOptions.map((option) => ({
            id: option.value,
            label: option.label,
          }))}
          selected={structureFilter}
          onChange={setStructureFilter}
          multi
          searchable
          label="Structure"
          placeholder="All"
          buttonClassName="input input-bordered h-9 min-h-9 w-full justify-between"
          menuClassName="z-[12000]"
        />
      </label>

      <label className="form-control">
        <div className="label py-1">
          <span className="label-text text-xs uppercase tracking-wide">
            Severity
          </span>
        </div>
        <SelectionDropdown
          items={severityOptions.map((option) => ({
            id: option.value,
            label: option.label,
          }))}
          selected={severityFilter}
          onChange={setSeverityFilter}
          multi
          searchable
          label="Severity"
          placeholder="All"
          buttonClassName="input input-bordered h-9 min-h-9 w-full justify-between"
          menuClassName="z-[12000]"
        />
      </label>
    </div>
  );
}
