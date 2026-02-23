import {
  useAllianceLogo,
  useCharacterPortrait,
  useCorporationLogo,
} from "@/hooks/useEveImage";
import { useAuthStore } from "@/app/store/authStore";
import HoverCard from "@/components/HoverCard";

type CharacterInfo = {
  name: string;
  character_id: number;
  is_main?: boolean;
  corp_name?: string;
  corp_id?: number;
  alliance_name?: string;
  alliance_id?: number;
  esi_token_valid?: boolean;
  esi_last_error?: string;
  esi_last_refresh_at?: string;
};

type CharacterCardProps = {
  character: CharacterInfo;
  onSetMain?: () => void;
  onRefresh?: () => void;
  onRevoke?: () => void;
  onRemove?: () => void;
  disableRemove?: boolean;
};

const formatTimestamp = (value?: string) => {
  if (!value) return "";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString();
};

export default function CharacterCard({
  character,
  onSetMain,
  onRefresh,
  onRevoke,
  onRemove,
  disableRemove,
}: CharacterCardProps) {
  const {
    name,
    character_id: characterId,
    is_main: isMain,
    corp_name: corpName,
    corp_id: corpId,
    alliance_name: allianceName,
    alliance_id: allianceId,
    esi_token_valid: esiTokenValid,
    esi_last_error: esiLastError,
    esi_last_refresh_at: esiLastRefreshAt,
  } = character;
  const isAdmin = useAuthStore((s) => s.isAdmin);
  const showAdminActions =
    isAdmin && (onSetMain || onRefresh || onRevoke || onRemove);
  const portraitUrl = useCharacterPortrait(characterId, 128);
  const allianceLogo = useAllianceLogo(allianceId, 32);
  const corpLogo = useCorporationLogo(corpId, 32);
  const allianceLabel = allianceName || (allianceId ? String(allianceId) : "");
  const corpLabel = corpName || (corpId ? String(corpId) : "");
  const hasEsiStatus = typeof esiTokenValid === "boolean";
  const esiBadgeLabel = hasEsiStatus
    ? esiTokenValid
      ? "Valid"
      : "Invalid"
    : "Unknown";
  const esiBadgeStyle = hasEsiStatus
    ? esiTokenValid
      ? "badge-success"
      : "badge-warning"
    : "badge-ghost";
  const formattedRefresh = formatTimestamp(esiLastRefreshAt);
  const esiTooltip = hasEsiStatus
    ? esiTokenValid
      ? formattedRefresh
        ? `Last refreshed at: ${formattedRefresh}`
        : "ESI token valid"
      : `${esiLastError ? `Invalid: ${esiLastError}` : "ESI token invalid"}${
          formattedRefresh ? `\nLast refreshed at: ${formattedRefresh}` : ""
        }`
    : "ESI status unavailable";

  return (
    <div className="character-card rounded-xl border border-slate-800/70 bg-base-300/60 px-4 py-3">
      <img
        src={portraitUrl}
        alt={name || "Character portrait"}
        className="character-portrait"
        loading="lazy"
      />
      <div className="character-card-header">
        <div className="flex items-center gap-2">
          <p className="text-base font-semibold">
            {name || "Unknown character"}
          </p>
          {isMain && <span className="badge badge-primary badge-sm">Main</span>}
          {!isMain && <span className="badge badge-ghost badge-sm">Alt</span>}
        </div>
        <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
          <span>Character ID {characterId}</span>
          <span className="flex items-center gap-1">
            <span>ESI:</span>
            <HoverCard
              trigger={
                <span
                  className={`badge badge-sm cursor-help ${esiBadgeStyle}`}
                  tabIndex={0}
                >
                  {esiBadgeLabel}
                </span>
              }
              className="hover-card-surface rounded-md p-2 text-xs max-w-80"
            >
              <div className="whitespace-pre-line">{esiTooltip}</div>
            </HoverCard>
          </span>
        </div>
      </div>
      <div className="character-card-affiliation text-xs text-slate-400 space-y-1">
        <div className="flex items-center gap-2">
          {allianceLogo && (
            <img
              src={allianceLogo}
              alt="Alliance logo"
              className="character-affiliation-logo"
              loading="lazy"
            />
          )}
          <p>Alliance {allianceLabel || "No Alliance"}</p>
        </div>
        <div className="flex items-center gap-2">
          {corpLogo && (
            <img
              src={corpLogo}
              alt="Corporation logo"
              className="character-affiliation-logo"
              loading="lazy"
            />
          )}
          <p>Corporation {corpLabel || "No Corporation"}</p>
        </div>
      </div>
      <div className="character-card-actions text-xs">
        {showAdminActions && (
          <div className="flex flex-wrap gap-2">
            {onSetMain && !isMain && (
              <button className="btn btn-xs btn-outline" onClick={onSetMain}>
                Set main
              </button>
            )}
            {onRefresh && (
              <button className="btn btn-xs btn-outline" onClick={onRefresh}>
                Refresh
              </button>
            )}
            {onRevoke && (
              <button className="btn btn-xs btn-outline" onClick={onRevoke}>
                Revoke keys
              </button>
            )}
            {onRemove && (
              <button
                className="btn btn-xs btn-outline btn-error"
                onClick={onRemove}
                disabled={disableRemove}
              >
                Remove
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
