import { create } from "zustand";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { useUIStore } from "@/app/store/uiStore";
import { getErrorMessage, getHttpStatus } from "@/utils/httpError";
import type { JobRunGroup } from "../types";

type AdminJobRunsState = {
  jobRuns: JobRunGroup[];
  loading: boolean;
  page: number;
  hasMore: boolean;
  etag?: string;
  startDate?: string;
  endDate?: string;
  startHour?: number;
  endHour?: number;
  includeHidden: boolean;
  jobKindExclusions: string[];
  loadJobs: (page?: number, opts?: { silent?: boolean }) => Promise<void>;
  setDateRange: (range: {
    startDate?: string;
    endDate?: string;
    startHour?: number;
    endHour?: number;
  }) => void;
  setJobKinds: (kinds: string[]) => void;
  setIncludeHidden: (include: boolean) => void;
  subscribe: () => Promise<() => void>;
  cancelJob: (jobId: string) => Promise<void>;
};

export const useAdminJobRunsStore = create<AdminJobRunsState>((set, get) => ({
  jobRuns: [],
  loading: false,
  page: 1,
  hasMore: false,
  etag: undefined,
  startDate: undefined,
  endDate: undefined,
  startHour: undefined,
  endHour: undefined,
  includeHidden: false,
  jobKindExclusions: [],
  loadJobs: async (page = get().page, opts) => {
    const silent = opts?.silent ?? (page === 1 && get().jobRuns.length > 0);
    if (!silent) {
      set({ loading: true });
    }
    try {
      const {
        startDate,
        endDate,
        startHour,
        endHour,
        includeHidden,
        jobKindExclusions,
      } = get();
      const startAt =
        startDate && startHour !== undefined
          ? `${startDate}T${String(startHour).padStart(2, "0")}:00:00Z`
          : undefined;
      const endAt =
        endDate && endHour !== undefined
          ? `${endDate}T${String(endHour).padStart(2, "0")}:59:59Z`
          : undefined;
      const res = await api.get("/admin/job-runs", {
        params: {
          page,
          limit: 20,
          startAt,
          endAt,
          startDate: startDate || undefined,
          endDate: endDate || undefined,
          kinds:
            jobKindExclusions.length > 0
              ? jobKindExclusions.join(",")
              : undefined,
          includeHidden: includeHidden || undefined,
        },
        headers: get().etag ? { "If-None-Match": get().etag } : undefined,
      });
      set({
        jobRuns: res.data.jobs || [],
        hasMore: !!res.data.hasMore,
        page: res.data.page || page,
        loading: false,
        etag: res.headers?.etag ?? get().etag,
      });
    } catch (error: unknown) {
      if (getHttpStatus(error) === 304) {
        if (!silent) {
          set({ loading: false });
        }
        return;
      }
      set({ jobRuns: [], hasMore: false, loading: false });
    }
  },
  setDateRange: ({ startDate, endDate, startHour, endHour }) => {
    set((state) => ({
      startDate,
      endDate,
      startHour,
      endHour,
      etag: undefined,
      page: 1,
      jobRuns: state.page === 1 ? state.jobRuns : [],
    }));
  },
  setJobKinds: (kinds) => {
    set((state) => ({
      jobKindExclusions: kinds,
      etag: undefined,
      page: 1,
      jobRuns: state.page === 1 ? state.jobRuns : [],
    }));
  },
  setIncludeHidden: (includeHidden) => {
    set((state) => ({
      includeHidden,
      etag: undefined,
      page: 1,
      jobRuns: state.page === 1 ? state.jobRuns : [],
    }));
  },
  subscribe: async () => {
    let pending = false;
    let timer: number | null = null;
    const scheduleRefresh = () => {
      if (get().page !== 1) return;
      if (pending) return;
      pending = true;
      timer = window.setTimeout(() => {
        pending = false;
        void get().loadJobs(1, { silent: true });
      }, 300);
    };
    const unsubscribe = await pb.collection("job_runs").subscribe("*", () => {
      scheduleRefresh();
    });
    return async () => {
      if (timer) window.clearTimeout(timer);
      await unsubscribe();
    };
  },
  cancelJob: async (jobId: string) => {
    const { setToast } = useUIStore.getState();
    if (!jobId) {
      setToast({ text: "Missing job id", color: "error" });
      return;
    }
    try {
      await api.post(`/admin/jobs/${jobId}/cancel`);
      setToast({ text: `Cancel requested for job ${jobId}`, color: "info" });
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to cancel job"),
        color: "error",
      });
    }
  },
}));
