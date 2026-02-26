import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import SearchSuggestionField from "@/components/SearchSuggestionField";
import { api } from "@/config/api";
import { useUIStore } from "@/app/store/uiStore";
import { useAllianceLogo, useCorporationLogo } from "@/hooks/useEveImage";
import { organizationBadgeClass } from "@/features/timers";
import {
  ADMIN_MODAL,
  defineAdminModal,
  useAdminStore,
} from "../store/adminStore";

type AllowedOrganization = {
  eve_id: number;
  name: string;
};

type AllowedOrganizationsResponse = {
  alliances: AllowedOrganization[];
  corporations: AllowedOrganization[];
};

type AllowedOrganizationListItem = AllowedOrganization & {
  type: "alliance" | "corporation";
};

type OrganizationEntityOption = {
  type: "corporation" | "alliance";
  id: number;
  name: string;
  ticker: string;
};

function AllowedOrganizationsModalBody() {
  const { close } = useModalBody();
  const setToast = useUIStore((s) => s.setToast);
  const [alliances, setAlliances] = useState<AllowedOrganization[]>([]);
  const [corporations, setCorporations] = useState<AllowedOrganization[]>([]);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<OrganizationEntityOption | null>(
    null,
  );
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(true);
  const listItems = useMemo<AllowedOrganizationListItem[]>(() => {
    const mapped = [
      ...alliances.map((entry) => ({ ...entry, type: "alliance" as const })),
      ...corporations.map((entry) => ({
        ...entry,
        type: "corporation" as const,
      })),
    ];
    mapped.sort((a, b) => {
      const nameComparison = a.name.localeCompare(b.name, undefined, {
        sensitivity: "base",
      });
      if (nameComparison !== 0) return nameComparison;
      return a.type.localeCompare(b.type);
    });
    return mapped;
  }, [alliances, corporations]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<AllowedOrganizationsResponse>(
        "/admin/site-settings/allowed-organizations",
      );
      setAlliances(response.data.alliances ?? []);
      setCorporations(response.data.corporations ?? []);
    } catch {
      setAlliances([]);
      setCorporations([]);
      setToast({
        text: "Failed to load allowed organizations",
        color: "error",
      });
    } finally {
      setLoading(false);
    }
  }, [setToast]);

  useEffect(() => {
    void load();
  }, [load]);

  const loadOrganizationSuggestions = useCallback(async (value: string) => {
    const response = await api.get<{ entities: OrganizationEntityOption[] }>(
      `/timers/entities?query=${encodeURIComponent(value)}`,
    );
    return (response.data.entities || []).filter(
      (entity) => entity.type === "corporation" || entity.type === "alliance",
    );
  }, []);

  const addSelected = async () => {
    if (!selected || submitting) return;
    setSubmitting(true);
    try {
      await api.post("/admin/site-settings/allowed-organizations", {
        type: selected.type,
        eve_id: selected.id,
        name: selected.name,
      });
      setToast({
        text: `Added ${selected.type} allow entry`,
        color: "info",
      });
      setSelected(null);
      setQuery("");
      await load();
    } catch {
      setToast({
        text: "Failed to add allowed organization",
        color: "error",
      });
    } finally {
      setSubmitting(false);
    }
  };

  const removeEntry = async (
    type: "alliance" | "corporation",
    eveID: number,
  ) => {
    try {
      await api.delete(
        `/admin/site-settings/allowed-organizations/${type}/${eveID}`,
      );
      setToast({
        text: `Removed ${type} allow entry`,
        color: "info",
      });
      await load();
    } catch {
      setToast({
        text: "Failed to remove allowed organization",
        color: "error",
      });
    }
  };

  return (
    <div className="space-y-4 text-sm text-slate-300">
      <p className="text-xs leading-relaxed text-slate-400">
        Add organizations allowed to authenticate and act as notification
        sources.
      </p>
      <div className="rounded-xl border border-slate-800/70 bg-base-300/40 p-3">
        <div className="flex items-start gap-2">
          <div className="flex-1 min-w-0">
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
                }
              }}
              onSelect={(entity) => {
                if (
                  entity.type !== "corporation" &&
                  entity.type !== "alliance"
                ) {
                  return;
                }
                setSelected(entity);
              }}
              selectionInputMode="set"
              placeholder="Search corporation or alliance"
              inputClassName="input input-sm input-bordered bg-base-300 w-full"
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
                  <div className="text-[11px] text-slate-400">
                    <span
                      className={`badge badge-xs ${organizationBadgeClass(entity.type)}`}
                    >
                      {entity.type === "corporation"
                        ? "Corporation"
                        : "Alliance"}
                    </span>
                  </div>
                </>
              )}
            />
          </div>
          <button
            className="btn btn-sm btn-success btn-outline gap-1"
            onClick={() => void addSelected()}
            disabled={!selected || submitting}
          >
            <Plus className="h-3.5 w-3.5" />
            Add
          </button>
        </div>
      </div>

      <div className="rounded-xl border border-slate-800/70 bg-base-300/30 p-3">
        <ul className="space-y-2 text-sm">
          {!loading && listItems.length === 0 && (
            <li className="text-slate-500">No allowed organizations.</li>
          )}
          {listItems.map((entry) => (
            <AllowedOrganizationRow
              key={`${entry.type}-${entry.eve_id}`}
              entry={entry}
              onDelete={() => void removeEntry(entry.type, entry.eve_id)}
            />
          ))}
        </ul>
      </div>

      <div className="flex items-center justify-end border-t border-slate-800/70 pt-3">
        <button className="btn btn-xs btn-outline" onClick={() => close()}>
          Close
        </button>
      </div>
    </div>
  );
}

export const AdminModalAllowedOrganizations = defineAdminModal({
  key: ADMIN_MODAL.AllowedOrganizations,
  useOpen: () =>
    useAdminStore((s) => s.modals[ADMIN_MODAL.AllowedOrganizations]),
  build: () => ({
    title: "Allowed Organizations",
    sizeClass: "max-w-3xl",
    body: <AllowedOrganizationsModalBody />,
  }),
});

export default function AllowedOrganizationsModal() {
  useModal(AdminModalAllowedOrganizations);
  return null;
}

function AllowedOrganizationRow({
  entry,
  onDelete,
}: {
  entry: AllowedOrganizationListItem;
  onDelete: () => void;
}) {
  const allianceLogo = useAllianceLogo(
    entry.type === "alliance" ? entry.eve_id : undefined,
    32,
  );
  const corporationLogo = useCorporationLogo(
    entry.type === "corporation" ? entry.eve_id : undefined,
    32,
  );
  const icon = entry.type === "alliance" ? allianceLogo : corporationLogo;
  return (
    <li className="flex items-center justify-between rounded-lg border border-slate-800/70 bg-base-300/40 px-3 py-2">
      <div className="min-w-0 flex items-center gap-2">
        {icon ? (
          <img
            src={icon}
            alt={`${entry.type} logo`}
            className="h-6 w-6 rounded object-cover"
            loading="lazy"
          />
        ) : (
          <div className="h-6 w-6 rounded bg-base-200/70 border border-slate-800/80" />
        )}
        <div className="min-w-0">
          <div className="font-medium text-slate-100 truncate">
            {entry.name}
          </div>
          <div className="mt-1">
            <span
              className={`badge badge-xs ${organizationBadgeClass(entry.type)}`}
            >
              {entry.type === "corporation" ? "Corporation" : "Alliance"}
            </span>
          </div>
        </div>
      </div>
      <button
        className="btn btn-xs btn-outline btn-square btn-error"
        onClick={onDelete}
        aria-label={`Remove ${entry.type}`}
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </li>
  );
}
