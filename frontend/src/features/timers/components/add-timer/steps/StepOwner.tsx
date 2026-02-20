import { Check } from "lucide-react";
import type { ComponentType } from "react";
import SearchSuggestionField from "@/components/SearchSuggestionField";
import type { StructureTone } from "../../../config/timerOptions";
import { useTimerFormStore } from "../../../store/useTimerFormStore";
import type { TimerEntityOption, TimerStandingType } from "../../../types";

type Props = {
  loadOwnerSuggestions: (query: string) => Promise<TimerEntityOption[]>;
  toneButtonClass: (tone: StructureTone, active: boolean) => string;
  hostilityOptions: ReadonlyArray<{
    value: TimerStandingType;
    label: string;
    icon: ComponentType<{ className?: string }>;
    tone: StructureTone;
  }>;
};

export default function StepOwner({
  loadOwnerSuggestions,
  toneButtonClass,
  hostilityOptions,
}: Props) {
  const form = useTimerFormStore((s) => s.form);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  const ownerQuery = useTimerFormStore((s) => s.ownerQuery);
  const setOwnerQuery = useTimerFormStore((s) => s.setOwnerQuery);
  return (
    <div className="space-y-4 rounded-xl border border-slate-700/70 bg-base-300/20 p-4">
      <div className="space-y-2">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Owner
        </div>
        <label className="form-control">
          <SearchSuggestionField<TimerEntityOption>
            query={ownerQuery}
            onQueryChange={(value) => {
              setOwnerQuery(value);
              const selectedLabel = form.owner_corporation_id
                ? `${form.owner_corporation_ticker ? `[${form.owner_corporation_ticker}] ` : ""}${form.owner_corporation_name}`
                    .trim()
                    .toLowerCase()
                : form.owner_alliance_id
                  ? `${form.owner_alliance_ticker ? `[${form.owner_alliance_ticker}] ` : ""}${form.owner_alliance_name}`
                      .trim()
                      .toLowerCase()
                  : "";
              if (
                selectedLabel &&
                value.trim().toLowerCase() !== selectedLabel
              ) {
                updateForm((state) => ({
                  ...state,
                  owner_corporation_id: 0,
                  owner_corporation_name: "",
                  owner_corporation_ticker: "",
                  owner_alliance_id: 0,
                  owner_alliance_name: "",
                  owner_alliance_ticker: "",
                }));
              }
            }}
            onSelect={(entity) => {
              if (entity.type === "corporation") {
                updateForm((state) => ({
                  ...state,
                  owner_corporation_id: entity.id,
                  owner_corporation_name: entity.name,
                  owner_corporation_ticker: entity.ticker,
                  owner_alliance_id:
                    entity.parent_alliance?.id || state.owner_alliance_id,
                  owner_alliance_name:
                    entity.parent_alliance?.name || state.owner_alliance_name,
                  owner_alliance_ticker:
                    entity.parent_alliance?.ticker ||
                    state.owner_alliance_ticker,
                }));
                return;
              }
              if (entity.type === "alliance") {
                updateForm((state) => ({
                  ...state,
                  owner_alliance_id: entity.id,
                  owner_alliance_name: entity.name,
                  owner_alliance_ticker: entity.ticker,
                }));
              }
            }}
            selectionInputMode="set"
            placeholder="Corporation or alliance name/ticker"
            inputClassName={`input input-bordered h-10 ${
              form.owner_corporation_id > 0 || form.owner_alliance_id > 0
                ? "border-emerald-500/80 bg-emerald-500/5 ring-1 ring-emerald-500/25"
                : ""
            }`}
            loadSuggestions={loadOwnerSuggestions}
            getSuggestionKey={(entity) => `${entity.type}-${entity.id}`}
            getInputValueFromSuggestion={(entity) =>
              entity.ticker ? `[${entity.ticker}] ${entity.name}` : entity.name
            }
            renderSuggestion={(entity) => (
              <>
                <div className="font-semibold text-slate-100">
                  {entity.ticker
                    ? `[${entity.ticker}] ${entity.name}`
                    : entity.name}
                </div>
                <div className="text-[11px] text-slate-400">
                  <span
                    className={`badge badge-xs ${
                      entity.type === "corporation"
                        ? "border-sky-400/50 bg-sky-500/20 text-sky-200"
                        : "border-violet-400/50 bg-violet-500/20 text-violet-200"
                    }`}
                  >
                    {entity.type === "corporation" ? "Corp" : "Alliance"}
                  </span>
                </div>
                {entity.type === "corporation" && entity.parent_alliance && (
                  <div className="text-[11px] text-violet-300/90">
                    Alliance: [{entity.parent_alliance.ticker || "?"}]{" "}
                    {entity.parent_alliance.name}
                  </div>
                )}
              </>
            )}
          />
        </label>
        {(form.owner_corporation_id > 0 || form.owner_alliance_id > 0) && (
          <div className="mt-1 flex items-center gap-1.5 text-xs text-success">
            <Check className="h-3.5 w-3.5" />
            <span>Selected</span>
          </div>
        )}
      </div>

      <div className="space-y-2">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Hostility
        </div>
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-5">
          {hostilityOptions.map((option) => {
            const Icon = option.icon;
            const active = form.standing_type === option.value;
            return (
              <button
                key={option.value}
                className={`btn btn-sm h-auto min-h-11 justify-start py-2 text-left leading-tight whitespace-normal ${toneButtonClass(option.tone, active)}`}
                onClick={() =>
                  updateForm((state) => ({
                    ...state,
                    standing_type: option.value,
                  }))
                }
              >
                <Icon className="h-4 w-4" /> {option.label}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
