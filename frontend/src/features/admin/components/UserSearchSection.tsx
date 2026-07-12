import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "@/config/api";
import CursorPagination from "@/components/CursorPagination";
import StartsWithFilter from "@/components/StartsWithFilter";
import ShadowedScrollArea from "@/components/ShadowedScrollArea";
import { useUIStore } from "@/app/store/uiStore";
import Panel from "@/components/Panel";
import { getErrorMessage, getHttpData, getHttpStatus } from "@/utils/httpError";
import { useAdminStore } from "../store/adminStore";
import { buildSearchLabel, formatAuthProvider } from "../utils/formatters";
import type { SearchResult } from "../types";

export default function UserSearchSection() {
  type DebugAPI = {
    generateFakeAdminSearchUsers?: (options?: {
      count?: number;
      prefix?: string;
    }) => Promise<{ created: number; prefix: string } | null>;
  };

  const PAGE_SIZE = 25;
  const SEARCH_DEBOUNCE_MS = 250;
  const setToast = useUIStore((s) => s.setToast);
  const loadUser = useAdminStore((s) => s.loadUser);
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [selectedLetter, setSelectedLetter] = useState("");
  const [availableLetters, setAvailableLetters] = useState<string[]>([]);
  const [refreshNonce, setRefreshNonce] = useState(0);
  const requestIdRef = useRef(0);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedQuery(query.trim());
      setPage(1);
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    if (!selectedLetter) return;
    if (availableLetters.includes(selectedLetter)) return;
    setSelectedLetter("");
    setPage(1);
  }, [availableLetters, selectedLetter]);

  useEffect(() => {
    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;
    setLoading(true);
    api
      .get("/admin/search", {
        params: {
          q: debouncedQuery,
          page,
          limit: PAGE_SIZE,
          startsWith: selectedLetter || undefined,
        },
      })
      .then((res) => {
        if (requestIdRef.current !== requestId) return;
        setResults(res.data?.results || []);
        setHasMore(Boolean(res.data?.hasMore));
        setAvailableLetters(
          Array.isArray(res.data?.availableLetters)
            ? res.data.availableLetters
            : [],
        );
      })
      .catch((error: unknown) => {
        if (requestIdRef.current !== requestId) return;
        setResults([]);
        setHasMore(false);
        setAvailableLetters([]);
        setToast({
          text: getErrorMessage(error, "Failed to load users"),
          color: "error",
          meta: {
            scope: "admin-user-search",
            operation: "search_users",
            status: getHttpStatus(error),
            data: getHttpData(error),
          },
        });
      })
      .finally(() => {
        if (requestIdRef.current !== requestId) return;
        setLoading(false);
      });
  }, [debouncedQuery, page, refreshNonce, selectedLetter, setToast]);

  useEffect(() => {
    const win = window as Window & { sentinelDebug?: DebugAPI };
    const debug = win.sentinelDebug ?? {};
    const generateFakeAdminSearchUsers: DebugAPI["generateFakeAdminSearchUsers"] =
      async (options) => {
        try {
          const response = await api.post("/admin/debug/seed-search-users", {
            count: options?.count,
            prefix: options?.prefix,
          });
          const result = {
            created: Number(response.data?.created || 0),
            prefix: String(response.data?.prefix || ""),
          };
          setPage(1);
          setRefreshNonce((value) => value + 1);
          console.info(
            "[sentinelDebug] generated fake admin search users",
            result,
          );
          return result;
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to seed fake admin users"),
            color: "error",
            meta: {
              scope: "admin-user-search",
              operation: "debug_seed_users",
              status: getHttpStatus(error),
              data: getHttpData(error),
            },
          });
          return null;
        }
      };
    debug.generateFakeAdminSearchUsers = generateFakeAdminSearchUsers;
    win.sentinelDebug = debug;

    return () => {
      const current = win.sentinelDebug;
      if (
        current?.generateFakeAdminSearchUsers === generateFakeAdminSearchUsers
      ) {
        delete current.generateFakeAdminSearchUsers;
      }
    };
  }, [setToast]);

  const formattedResults = useMemo(
    () =>
      results.map((result) => ({
        result,
        label: buildSearchLabel(result),
      })),
    [results],
  );
  const handleSelectUser = async (userId: string) => {
    try {
      await loadUser(userId);
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to load user"),
        color: "error",
        meta: {
          scope: "admin-user-search",
          operation: "load_user",
          userId,
          status: getHttpStatus(error),
          data: getHttpData(error),
        },
      });
    }
  };
  const searchAction = (
    <input
      className="input input-xs input-bordered bg-base-300 w-48 sm:w-56"
      placeholder="Search by character name or user ID"
      value={query}
      onChange={(e) => setQuery(e.target.value)}
      aria-label="Search by character name or user ID"
    />
  );

  return (
    <Panel
      title="User Search"
      actions={searchAction}
      className="h-full min-h-0"
      bodyClassName="flex flex-col gap-2 h-full min-h-0"
    >
      <StartsWithFilter
        selected={selectedLetter}
        available={availableLetters}
        onSelect={(value) => {
          setSelectedLetter(value);
          setPage(1);
        }}
      />
      <div className="relative flex-1 min-h-0 overflow-hidden">
        {formattedResults.length > 0 && (
          <ShadowedScrollArea>
            <ul className="space-y-2 text-sm">
              {formattedResults.map(({ result, label }) => (
                <li key={result.character_record_id}>
                  <button
                    className="w-full text-left rounded-lg border border-slate-800 bg-base-300/40 px-3 py-2 transition hover:bg-base-300/70"
                    onClick={() => void handleSelectUser(result.user_id)}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-semibold text-slate-100">
                          {label}
                        </span>
                        <span className="badge badge-xs badge-outline">
                          {formatAuthProvider(result.auth_provider)}
                        </span>
                      </div>
                      <span
                        className={`badge badge-xs ${result.is_main ? "badge-primary" : "badge-ghost"}`}
                      >
                        {result.is_main ? "Main" : "Alt"}
                      </span>
                    </div>
                    {result.main_name && !result.is_main && (
                      <div className="text-[11px] text-slate-400">
                        Main: {result.main_name}
                      </div>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          </ShadowedScrollArea>
        )}
        {formattedResults.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center text-center">
            <p className="text-sm font-medium text-slate-400">
              {loading ? "Loading users..." : "No results"}
            </p>
          </div>
        )}
        {loading && formattedResults.length > 0 && (
          <div className="pointer-events-none absolute right-2 top-1 text-[11px] text-slate-400 bg-base-300/60 rounded px-1.5 py-0.5">
            Updating...
          </div>
        )}
      </div>
      {(page > 1 || hasMore) && (
        <CursorPagination
          page={page}
          hasMore={hasMore}
          loading={loading}
          onPrev={() => setPage((prev) => Math.max(1, prev - 1))}
          onNext={() => setPage((prev) => prev + 1)}
        />
      )}
    </Panel>
  );
}
