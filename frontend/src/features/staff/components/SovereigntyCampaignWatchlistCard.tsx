import { useCallback, useEffect, useState } from "react";
import type { ComponentType } from "react";
import { Check, Plus, Trash2, X } from "lucide-react";
import { api } from "@/config/api";
import Panel from "@/components/Panel";
import SearchSuggestionField from "@/components/SearchSuggestionField";
import { useAllianceLogo } from "@/hooks/useEveImage";
import {
  formatStanding,
  hostilityOptions,
  standingBadgeClass,
  TimerStandingType,
  toneButtonClass,
  type StructureTone,
} from "@/features/timers";

type AllianceEntityOption = {
  type: "alliance";
  id: number;
  name: string;
  ticker: string;
};

type WatchlistEntity = {
  id: string;
  hostility: TimerStandingType;
  alliance_id: number;
  alliance_name: string;
  alliance_ticker: string;
};

type HostilityOption = {
  value: TimerStandingType;
  label: string;
  icon: ComponentType<{ className?: string }>;
  tone: StructureTone;
};

export default function SovereigntyCampaignWatchlistCard() {
  const [entities, setEntities] = useState<WatchlistEntity[]>([]);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<AllianceEntityOption | null>(null);
  const [isAdding, setIsAdding] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [hostility, setHostility] = useState<TimerStandingType>(
    TimerStandingType.Hostile,
  );
  const loadOrganizationSuggestions = useCallback(async (query: string) => {
    const response = await api.get<{ entities: AllianceEntityOption[] }>(
      `/timers/entities?query=${encodeURIComponent(query)}&scope=alliance`,
    );
    return (response.data.entities || []).filter(
      (entity) => entity.type === "alliance",
    );
  }, []);

  const loadEntities = useCallback(async () => {
    try {
      const response = await api.get<{ entities: WatchlistEntity[] }>(
        "/staff/sovereignty-campaign-watchlist",
      );
      setEntities(response.data.entities ?? []);
    } catch {
      setEntities([]);
    }
  }, []);

  useEffect(() => {
    void loadEntities();
  }, [loadEntities]);

  const addEntity = async () => {
    if (!selected || submitting) return;
    setSubmitting(true);
    try {
      await api.post("/staff/sovereignty-campaign-watchlist", {
        hostility,
        alliance_id: selected.id,
        alliance_name: selected.name,
        alliance_ticker: selected.ticker,
      });
      setSelected(null);
      setQuery("");
      setIsAdding(false);
      setHostility(TimerStandingType.Hostile);
      await loadEntities();
    } finally {
      setSubmitting(false);
    }
  };

  const deleteEntity = async (id: string) => {
    await api.delete(`/staff/sovereignty-campaign-watchlist/${id}`);
    await loadEntities();
  };

  return (
    <Panel
      title="Soverignty Campaign Entities"
      hint="Only campaigns for these alliances are monitored and auto-added from ESI."
      bodyClassName="space-y-4"
    >
      <ul className="space-y-2 text-sm">
        {entities.length === 0 && (
          <li className="text-slate-500">No watchlist entities configured.</li>
        )}

        {entities.map((entry) => (
          <SovereigntyWatchlistRow
            key={entry.id}
            entry={entry}
            onDelete={() => void deleteEntity(entry.id)}
          />
        ))}

        <li className="rounded-lg border border-dashed border-slate-700/80 bg-base-300/20 px-3 py-2">
          {isAdding ? (
            <div className="space-y-2">
              <SearchSuggestionField<AllianceEntityOption>
                query={query}
                onQueryChange={(value) => {
                  setQuery(value);
                  const selectedLabel = selected
                    ? selected.ticker
                      ? `[${selected.ticker}] ${selected.name}`
                      : selected.name
                    : "";
                  if (
                    selected &&
                    selectedLabel.trim().toLowerCase() !==
                      value.trim().toLowerCase()
                  ) {
                    setSelected(null);
                  }
                }}
                onSelect={(entity) => {
                  if (entity.type !== "alliance") {
                    return;
                  }
                  setSelected(entity);
                }}
                selectionInputMode="set"
                placeholder="Search alliance"
                inputClassName="input input-xs input-bordered bg-base-300"
                loadSuggestions={async (value) => {
                  return loadOrganizationSuggestions(value);
                }}
                getSuggestionKey={(entity) => `${entity.type}-${entity.id}`}
                getInputValueFromSuggestion={(entity) =>
                  entity.ticker
                    ? `[${entity.ticker}] ${entity.name}`
                    : entity.name
                }
                renderSuggestion={(entity) => (
                  <>
                    <div className="font-semibold text-slate-100">
                      {entity.ticker
                        ? `[${entity.ticker}] ${entity.name}`
                        : entity.name}
                    </div>
                    <div className="text-[11px] text-slate-400">
                      <span className="badge badge-xs border-violet-400/50 bg-violet-500/20 text-violet-200">
                        Alliance
                      </span>
                    </div>
                  </>
                )}
              />
              <div className="flex items-center gap-2">
                <span className="text-xs text-slate-400">Hostility:</span>
              </div>
              <div className="grid grid-cols-2 gap-2 lg:grid-cols-5">
                {(hostilityOptions as readonly HostilityOption[]).map(
                  (option) => {
                    const Icon = option.icon;
                    const active = hostility === option.value;
                    return (
                      <button
                        key={option.value}
                        className={`btn btn-xs h-auto min-h-10 justify-start py-2 text-left leading-tight whitespace-normal ${toneButtonClass(option.tone, active)}`}
                        onClick={() => setHostility(option.value)}
                        disabled={submitting}
                      >
                        <Icon className="h-3.5 w-3.5" /> {option.label}
                      </button>
                    );
                  },
                )}
              </div>

              <div className="flex items-center justify-between gap-2">
                {selected ? (
                  <div className="flex items-center gap-1.5 text-xs text-success">
                    <Check className="h-3.5 w-3.5" />
                    <span>Selected</span>
                  </div>
                ) : (
                  <div className="text-xs text-slate-500">
                    Select an organization
                  </div>
                )}
                <div className="flex items-center gap-1">
                  <button
                    className="btn btn-xs btn-outline btn-square"
                    onClick={() => void addEntity()}
                    aria-label="Save watchlist entity"
                    disabled={!selected || submitting}
                  >
                    <Check className="h-4 w-4" />
                  </button>
                  <button
                    className="btn btn-xs btn-outline btn-square"
                    onClick={() => {
                      setSelected(null);
                      setQuery("");
                      setIsAdding(false);
                      setHostility(TimerStandingType.Hostile);
                    }}
                    aria-label="Cancel add"
                    disabled={submitting}
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <button
              className="flex w-full items-center justify-center gap-2 rounded-md px-2 py-1.5 text-slate-300 hover:bg-base-300/40 hover:text-slate-100 focus:outline-none focus-visible:ring-1 focus-visible:ring-slate-500/70"
              onClick={() => setIsAdding(true)}
              aria-label="Add watchlist entity"
            >
              <Plus className="h-4 w-4" />
              Add alliance
            </button>
          )}
        </li>
      </ul>
    </Panel>
  );
}

function SovereigntyWatchlistRow({
  entry,
  onDelete,
}: {
  entry: WatchlistEntity;
  onDelete: () => void;
}) {
  const icon = useAllianceLogo(entry.alliance_id, 32);
  return (
    <li className="flex items-center justify-between rounded-lg border border-slate-800/70 bg-base-300/40 px-3 py-2">
      <div className="min-w-0 flex items-center gap-2">
        {icon ? (
          <img
            src={icon}
            alt="alliance logo"
            className="h-6 w-6 rounded object-cover"
            loading="lazy"
          />
        ) : (
          <div className="h-6 w-6 rounded bg-base-200/70 border border-slate-800/80" />
        )}
        <div className="min-w-0">
          <div className="font-medium text-slate-100 truncate">
            {entry.alliance_ticker
              ? `[${entry.alliance_ticker}] ${entry.alliance_name}`
              : entry.alliance_name}
          </div>
          <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-400">
            <span className="badge badge-xs border-violet-400/50 bg-violet-500/20 text-violet-200">
              Alliance
            </span>
            <span
              className={`badge badge-xs ${standingBadgeClass(entry.hostility)}`}
            >
              {formatStanding(entry.hostility)}
            </span>
          </div>
        </div>
      </div>
      <button
        className="btn btn-xs btn-outline btn-square btn-error"
        onClick={onDelete}
        aria-label="Delete watchlist entity"
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </li>
  );
}
