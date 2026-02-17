import { useMemo, useState } from "react";

type StartsWithFilterProps = {
  selected: string;
  available: string[];
  onSelect: (value: string) => void;
  label?: string;
  tokens?: string[];
  alwaysVisibleTokens?: string[];
  className?: string;
};

const DEFAULT_TOKENS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ".split("");
const DEFAULT_ALWAYS_VISIBLE = ["0", "9", "A", "Z"];

export default function StartsWithFilter({
  selected,
  available,
  onSelect,
  label = "Starts with",
  tokens = DEFAULT_TOKENS,
  alwaysVisibleTokens = DEFAULT_ALWAYS_VISIBLE,
  className,
}: StartsWithFilterProps) {
  const [hoveredToken, setHoveredToken] = useState<string | null>(null);
  const tokenRail = useMemo(
    () => ["all", ...tokens.map((token) => token.toLowerCase())],
    [tokens],
  );
  const alwaysVisible = useMemo(
    () => new Set(alwaysVisibleTokens),
    [alwaysVisibleTokens],
  );

  const tokenScale = (tokenId: string): number => {
    if (!hoveredToken) return 1;
    const hoveredIndex = tokenRail.indexOf(hoveredToken);
    const tokenIndex = tokenRail.indexOf(tokenId);
    if (hoveredIndex < 0 || tokenIndex < 0) return 1;
    const distance = Math.abs(hoveredIndex - tokenIndex);
    if (distance === 0) return 1.55;
    if (distance === 1) return 1.3;
    if (distance === 2) return 1.15;
    if (distance === 3) return 1.06;
    return 1;
  };
  const tokenDisplayScale = (tokenId: string, isSelected: boolean): number => {
    const selectedScale = isSelected ? 1.18 : 1;
    return Math.max(tokenScale(tokenId), selectedScale);
  };

  return (
    <div
      className={`rounded-lg border border-slate-800/80 bg-base-300/25 px-2 pt-1 pb-2 ${
        className ?? ""
      }`.trim()}
    >
      <div className="text-[10px] uppercase tracking-[0.14em] text-slate-400 mb-2">
        {label}
      </div>
      <div className="min-w-0" onMouseLeave={() => setHoveredToken(null)}>
        <div className="w-full flex flex-wrap items-center justify-between gap-y-1.5">
          <button
            className={`btn btn-xs h-4 min-h-0 rounded px-1 text-[10px] border transition-transform duration-150 ${
              selected === ""
                ? "btn-success shadow-[0_0_14px_rgba(34,197,94,0.6)]"
                : "btn-ghost"
            }`}
            style={{
              transform: `scale(${tokenDisplayScale("all", selected === "")})`,
              zIndex: hoveredToken === "all" ? 30 : 10,
            }}
            onClick={() => onSelect("")}
            onMouseEnter={() => setHoveredToken("all")}
          >
            All
          </button>
          {tokens.map((token) => {
            const enabled = available.includes(token);
            const active = selected === token;
            const tokenId = token.toLowerCase();
            if (!enabled && !alwaysVisible.has(token)) {
              return (
                <span
                  key={token}
                  className="inline-block h-1 w-1 rounded-full bg-slate-600/70 transition-transform duration-150"
                  title={`${token}: no matches`}
                  aria-label={`${token}: no matches`}
                  style={{
                    transform: `scale(${tokenScale(tokenId)})`,
                    zIndex: hoveredToken === tokenId ? 30 : 10,
                  }}
                  onMouseEnter={() => setHoveredToken(tokenId)}
                />
              );
            }
            return (
              <button
                key={token}
                className={`h-4 min-h-0 w-4 rounded border text-[9px] font-medium transition-transform duration-150 ${
                  active
                    ? "text-emerald-200 border-emerald-400/90 drop-shadow-[0_0_12px_rgba(34,197,94,0.75)]"
                    : enabled
                      ? "starts-with-token-available border-transparent"
                      : "text-slate-300 border-transparent"
                } ${enabled ? "starts-with-token-available-hover" : "opacity-45"}`}
                style={{
                  transform: `scale(${tokenDisplayScale(tokenId, active)})`,
                  zIndex: hoveredToken === tokenId ? 30 : 10,
                }}
                onClick={() => {
                  if (!enabled) return;
                  onSelect(token);
                }}
                onMouseEnter={() => setHoveredToken(tokenId)}
              >
                {token}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
