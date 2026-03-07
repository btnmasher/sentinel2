import { useCallback, useEffect, useState } from "react";
import type { ComponentType } from "react";
import { Check, Pencil, Plus, Trash2, X } from "lucide-react";
import useConfirm from "@/app/hooks/useConfirm";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { api } from "@/config/api";
import Panel from "@/components/Panel";
import SearchSuggestionField from "@/components/SearchSuggestionField";
import { useAllianceLogo, useCorporationLogo } from "@/hooks/useEveImage";
import {
  formatStanding,
  hostilityOptions,
  organizationBadgeClass,
  type StructureTone,
} from "@/features/timers";
import {
  StandingType,
  standingBadgeClass,
  toneButtonClass,
} from "@/features/shared";

type OrganizationEntityOption = {
  type: "alliance" | "corporation";
  id: number;
  name: string;
  ticker: string;
  parent_alliance?: {
    id: number;
    name: string;
    ticker: string;
  };
};

type OrganizationStanding = {
  id: string;
  owner_type: "alliance" | "corporation";
  hostility: StandingType;
  include_in_sov_sync: boolean;
  corporation_id: number;
  corporation_name: string;
  corporation_ticker: string;
  alliance_id: number;
  alliance_name: string;
  alliance_ticker: string;
};

type HostilityOption = {
  value: StandingType;
  label: string;
  icon: ComponentType<{ className?: string }>;
  tone: StructureTone;
};

export default function OrganizationStandingsCard() {
  const requestConfirm = useConfirm();
  const timersEnabled = useAppConfigStore((s) => s.timersEnabled);
  const timerSource = useAppConfigStore((s) => s.timerSource);
  const [entities, setEntities] = useState<OrganizationStanding[]>([]);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<OrganizationEntityOption | null>(
    null,
  );
  const [isAdding, setIsAdding] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [hostility, setHostility] = useState<StandingType>(
    StandingType.Hostile,
  );
  const [includeInSovSync, setIncludeInSovSync] = useState(false);
  const canManageSovSync = timersEnabled && timerSource === "standalone";

  const loadOrganizationSuggestions = useCallback(async (value: string) => {
    const response = await api.get<{ entities: OrganizationEntityOption[] }>(
      `/organizations/search?query=${encodeURIComponent(value)}&scope=both`,
    );
    return response.data.entities || [];
  }, []);

  const loadEntities = useCallback(async () => {
    try {
      const response = await api.get<{ entities: OrganizationStanding[] }>(
        "/staff/organization-standings",
      );
      setEntities(response.data.entities ?? []);
    } catch {
      setEntities([]);
    }
  }, []);

  useEffect(() => {
    void loadEntities();
  }, [loadEntities]);

  const canIncludeInSovSync =
    selected?.type === "alliance" ||
    (selected?.type === "corporation" &&
      Boolean(selected.parent_alliance && selected.parent_alliance.id > 0));

  const addEntity = async () => {
    if (!selected || submitting) return;
    setSubmitting(true);
    try {
      await api.post("/staff/organization-standings", {
        owner_type: selected.type,
        hostility,
        include_in_sov_sync:
          canManageSovSync && includeInSovSync && canIncludeInSovSync,
        corporation_id: selected.type === "corporation" ? selected.id : 0,
        corporation_name: selected.type === "corporation" ? selected.name : "",
        corporation_ticker:
          selected.type === "corporation" ? selected.ticker : "",
        alliance_id:
          selected.type === "alliance"
            ? selected.id
            : selected.parent_alliance?.id || 0,
        alliance_name:
          selected.type === "alliance"
            ? selected.name
            : selected.parent_alliance?.name || "",
        alliance_ticker:
          selected.type === "alliance"
            ? selected.ticker
            : selected.parent_alliance?.ticker || "",
      });
      setSelected(null);
      setQuery("");
      setIsAdding(false);
      setHostility(StandingType.Hostile);
      setIncludeInSovSync(false);
      await loadEntities();
    } finally {
      setSubmitting(false);
    }
  };

  const deleteEntity = async (id: string) => {
    requestConfirm({
      title: "Delete organization standing?",
      body: "This removes the standing.",
      onConfirm: async () => {
        await api.delete(`/staff/organization-standings/${id}`);
        await loadEntities();
      },
      confirmLabel: "Delete",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  };

  const updateIncludeInSovSync = async (
    entry: OrganizationStanding,
    include: boolean,
  ) => {
    if (submitting) return;
    setSubmitting(true);
    try {
      await api.patch(`/staff/organization-standings/${entry.id}`, {
        hostility: entry.hostility,
        include_in_sov_sync: include,
      });
      await loadEntities();
    } finally {
      setSubmitting(false);
    }
  };

  const updateHostility = async (
    entry: OrganizationStanding,
    nextHostility: StandingType,
  ) => {
    if (submitting) return;
    setSubmitting(true);
    try {
      await api.patch(`/staff/organization-standings/${entry.id}`, {
        hostility: nextHostility,
        include_in_sov_sync: entry.include_in_sov_sync,
      });
      await loadEntities();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Panel
      title="Organization Standings"
      hint={
        canManageSovSync
          ? "Set corp/alliance hostility and opt alliance-backed entities into sovereignty campaign sync."
          : "Set corp/alliance hostility for organization-aware features."
      }
      bodyClassName="space-y-4"
    >
      <ul className="space-y-2 text-sm">
        {entities.length === 0 && (
          <li className="text-slate-500">
            No organization standings configured.
          </li>
        )}

        {entities.map((entry) => (
          <OrganizationStandingRow
            key={entry.id}
            entry={entry}
            submitting={submitting}
            showSovSyncControls={canManageSovSync}
            onToggleInSovSync={(next) =>
              void updateIncludeInSovSync(entry, next)
            }
            onSaveHostility={(nextHostility) =>
              void updateHostility(entry, nextHostility)
            }
            onDelete={() => void deleteEntity(entry.id)}
          />
        ))}

        <li className="rounded-lg border border-dashed border-slate-700/80 bg-base-300/20 px-3 py-2">
          {isAdding ? (
            <div className="space-y-2">
              <SearchSuggestionField<OrganizationEntityOption>
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
                    setIncludeInSovSync(false);
                  }
                }}
                onSelect={(entity) => {
                  setSelected(entity);
                  if (
                    entity.type !== "alliance" &&
                    !(entity.parent_alliance && entity.parent_alliance.id > 0)
                  ) {
                    setIncludeInSovSync(false);
                  }
                }}
                selectionInputMode="set"
                placeholder="Search corporation or alliance"
                inputClassName="input input-xs input-bordered bg-base-300"
                loadSuggestions={loadOrganizationSuggestions}
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
                    <div className="text-[11px] text-slate-400 flex items-center gap-2">
                      <span
                        className={`badge badge-xs ${
                          entity.type === "corporation"
                            ? "border-sky-400/50 bg-sky-500/20 text-sky-200"
                            : "border-violet-400/50 bg-violet-500/20 text-violet-200"
                        }`}
                      >
                        {entity.type === "alliance"
                          ? "Alliance"
                          : "Corporation"}
                      </span>
                      {entity.parent_alliance ? (
                        <span>
                          Alliance: [{entity.parent_alliance.ticker}]{" "}
                          {entity.parent_alliance.name}
                        </span>
                      ) : null}
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
              {canManageSovSync ? (
                <>
                  <label className="label cursor-pointer justify-start gap-2 rounded-md border border-slate-700/70 px-2 py-1.5">
                    <input
                      type="checkbox"
                      className={`checkbox checkbox-xs rounded-[0.2rem] ${
                        !canIncludeInSovSync || submitting
                          ? "cursor-not-allowed opacity-40"
                          : ""
                      }`}
                      checked={includeInSovSync && canIncludeInSovSync}
                      disabled={!canIncludeInSovSync || submitting}
                      onChange={(event) =>
                        setIncludeInSovSync(
                          event.target.checked && canIncludeInSovSync,
                        )
                      }
                    />
                    <span
                      className={`text-xs ${
                        !canIncludeInSovSync || submitting
                          ? "text-slate-500"
                          : "text-slate-300"
                      }`}
                    >
                      Include in sovereignty campaign sync
                    </span>
                  </label>
                  {!canIncludeInSovSync && (
                    <div className="text-[11px] text-slate-500">
                      Requires an alliance-backed entity.
                    </div>
                  )}
                </>
              ) : null}

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
                    aria-label="Save organization standing"
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
                      setHostility(StandingType.Hostile);
                      setIncludeInSovSync(false);
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
              aria-label="Add organization standing"
            >
              <Plus className="h-4 w-4" />
              Add organization
            </button>
          )}
        </li>
      </ul>
    </Panel>
  );
}

function OrganizationStandingRow({
  entry,
  submitting,
  showSovSyncControls,
  onToggleInSovSync,
  onSaveHostility,
  onDelete,
}: {
  entry: OrganizationStanding;
  submitting: boolean;
  showSovSyncControls: boolean;
  onToggleInSovSync: (next: boolean) => void;
  onSaveHostility: (nextHostility: StandingType) => void;
  onDelete: () => void;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const [draftHostility, setDraftHostility] = useState<StandingType>(
    entry.hostility,
  );

  useEffect(() => {
    setDraftHostility(entry.hostility);
  }, [entry.hostility]);

  const allianceIcon = useAllianceLogo(entry.alliance_id, 32);
  const corporationIcon = useCorporationLogo(entry.corporation_id, 32);
  const icon = entry.owner_type === "alliance" ? allianceIcon : corporationIcon;

  const ownerLabel =
    entry.owner_type === "alliance"
      ? entry.alliance_ticker
        ? `[${entry.alliance_ticker}] ${entry.alliance_name}`
        : entry.alliance_name
      : entry.corporation_ticker
        ? `[${entry.corporation_ticker}] ${entry.corporation_name}`
        : entry.corporation_name;
  const canIncludeInSovSync = entry.alliance_id > 0;
  const includeDisabled = submitting || !canIncludeInSovSync;

  return (
    <li className="grid gap-2 rounded-lg border border-slate-800/70 bg-base-300/40 px-3 py-2 [grid-template-areas:'main_actions''editor_editor'] [grid-template-columns:minmax(0,1fr)_auto]">
      <div className="[grid-area:main] min-w-0 flex items-center gap-2">
        {icon ? (
          <img
            src={icon}
            alt="organization logo"
            className="h-6 w-6 rounded object-cover"
            loading="lazy"
          />
        ) : (
          <div className="h-6 w-6 rounded bg-base-200/70 border border-slate-800/80" />
        )}
        <div className="min-w-0">
          <div className="font-medium text-slate-100 truncate">
            {ownerLabel}
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-slate-400">
            <span
              className={`badge badge-xs ${organizationBadgeClass(entry.owner_type)}`}
            >
              {entry.owner_type === "alliance" ? "Alliance" : "Corporation"}
            </span>
            <span
              className={`badge badge-xs ${standingBadgeClass(entry.hostility)}`}
            >
              {formatStanding(entry.hostility)}
            </span>
          </div>
        </div>
      </div>
      <div className="[grid-area:actions] flex items-center gap-2">
        {showSovSyncControls ? (
          <div
            className={`flex items-center gap-1.5 ${
              includeDisabled ? "cursor-not-allowed" : ""
            }`}
          >
            <span
              className={`text-xs whitespace-nowrap ${
                includeDisabled ? "text-slate-500" : "text-slate-300"
              }`}
            >
              Include in Sov Sync
            </span>
            <input
              type="checkbox"
              className={`checkbox checkbox-xs rounded-[0.2rem] ${
                includeDisabled ? "cursor-not-allowed opacity-40" : ""
              }`}
              checked={entry.include_in_sov_sync}
              disabled={includeDisabled}
              onChange={(event) =>
                onToggleInSovSync(event.target.checked && canIncludeInSovSync)
              }
            />
          </div>
        ) : null}
        {isEditing ? (
          <>
            <button
              className="btn btn-xs btn-outline btn-square btn-success"
              onClick={() => {
                onSaveHostility(draftHostility);
                setIsEditing(false);
              }}
              aria-label="Save organization standing edits"
              disabled={submitting}
            >
              <Check className="h-4 w-4" />
            </button>
            <button
              className="btn btn-xs btn-outline btn-square"
              onClick={() => {
                setDraftHostility(entry.hostility);
                setIsEditing(false);
              }}
              aria-label="Cancel organization standing edits"
              disabled={submitting}
            >
              <X className="h-4 w-4" />
            </button>
          </>
        ) : (
          <button
            className="btn btn-xs btn-outline btn-square"
            onClick={() => setIsEditing(true)}
            aria-label="Edit organization standing"
            disabled={submitting}
          >
            <Pencil className="h-4 w-4" />
          </button>
        )}
        <button
          className="btn btn-xs btn-outline btn-square btn-error"
          onClick={onDelete}
          aria-label="Delete organization standing"
          disabled={submitting}
        >
          <Trash2 className="h-4 w-4" />
        </button>
      </div>
      <div className="[grid-area:editor] flex flex-col gap-2">
        {isEditing && (
          <div className="grid grid-cols-2 gap-1.5 md:grid-cols-5">
            {(hostilityOptions as readonly HostilityOption[]).map((option) => {
              const Icon = option.icon;
              const active = draftHostility === option.value;
              return (
                <button
                  key={option.value}
                  className={`btn btn-xs h-auto min-h-8 justify-start py-1.5 text-left leading-tight whitespace-normal ${toneButtonClass(option.tone, active)}`}
                  onClick={() => setDraftHostility(option.value)}
                  disabled={submitting}
                >
                  <Icon className="h-3.5 w-3.5" /> {option.label}
                </button>
              );
            })}
          </div>
        )}
      </div>
    </li>
  );
}
