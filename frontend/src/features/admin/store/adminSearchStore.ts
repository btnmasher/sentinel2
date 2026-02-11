import { create } from "zustand";
import { api } from "@/config/api";
import type { SearchResult } from "../types";

type SearchKey = "user" | "merge" | "move";

type SearchState = {
  query: string;
  results: SearchResult[];
  loading: boolean;
};

type AdminSearchState = {
  searches: Record<SearchKey, SearchState>;
  setQuery: (key: SearchKey, query: string) => void;
  clear: (key: SearchKey) => void;
};

const MIN_LENGTH = 2;
const DEBOUNCE_MS = 250;
const timers: Partial<Record<SearchKey, number>> = {};

const emptySearch: SearchState = { query: "", results: [], loading: false };

export const useAdminSearchStore = create<AdminSearchState>((set, get) => ({
  searches: {
    user: { ...emptySearch },
    merge: { ...emptySearch },
    move: { ...emptySearch },
  },
  setQuery: (key, query) => {
    const trimmed = query.trim();

    if (timers[key]) {
      window.clearTimeout(timers[key]);
    }

    set((state) => ({
      searches: {
        ...state.searches,
        [key]: {
          ...state.searches[key],
          query,
          loading: trimmed.length >= MIN_LENGTH,
          results:
            trimmed.length >= MIN_LENGTH ? state.searches[key].results : [],
        },
      },
    }));

    if (trimmed.length < MIN_LENGTH) return;

    timers[key] = window.setTimeout(() => {
      const latestQuery = get().searches[key].query;
      if (latestQuery.trim().length < MIN_LENGTH) return;

      api
        .get("/admin/search", { params: { q: latestQuery } })
        .then((res) => {
          if (get().searches[key].query !== latestQuery) return;
          set((state) => ({
            searches: {
              ...state.searches,
              [key]: {
                ...state.searches[key],
                results: res.data.results || [],
                loading: false,
              },
            },
          }));
        })
        .catch(() => {
          if (get().searches[key].query !== latestQuery) return;
          set((state) => ({
            searches: {
              ...state.searches,
              [key]: {
                ...state.searches[key],
                results: [],
                loading: false,
              },
            },
          }));
        });
    }, DEBOUNCE_MS);
  },
  clear: (key) => {
    if (timers[key]) {
      window.clearTimeout(timers[key]);
    }
    set((state) => ({
      searches: {
        ...state.searches,
        [key]: { ...emptySearch },
      },
    }));
  },
}));
