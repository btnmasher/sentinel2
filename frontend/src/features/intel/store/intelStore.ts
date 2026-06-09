import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { getHttpStatus } from "@/utils/httpError";
import type { IntelReport } from "../types";
import {
  isClearIntelReport,
  normalizeIntelReport,
} from "../utils/intelReportUtils";
import {
  INTEL_UPLOADER_COUNT_TOPIC,
  REALTIME_KEEPALIVE_TOPIC,
  zkillRegionTopic,
  normalizeUploaderCountMessage,
} from "../utils/intelRealtimeUtils";

export const INTEL_STORE_VERSION = 8;
const MAX_REPORT_AGE_SECONDS = 60 * 60;
const UPLOADER_RESYNC_THROTTLE_MS = 15_000;
let uploaderResyncInFlight: Promise<void> | null = null;
let lastUploaderResyncAt = 0;

type IntelFilters = {
  includeSystemLogs: boolean;
  includeSystemAlarm: boolean;
  includeUnknownLogs: boolean;
  includeUnknownAlarm: boolean;
  includeUnloadedRegionsLogs: boolean;
  includeUnloadedRegionsAlarm: boolean;
  system: number[];
};

type IntelState = {
  reports: IntelReport[];
  reportCount: number;
  lastReports: IntelReport[];
  reportsFetchedAt: number;
  uploaders: number;
  version: string;
  intelStatus: "connecting" | "connected" | "disconnected";
  reportsRealtimeUnsubscribe?: () => Promise<void>;
  uploadersRealtimeUnsubscribe?: () => Promise<void>;
  keepaliveRealtimeUnsubscribe?: () => Promise<void>;
  zkillRegionRealtimeUnsubscribes: Record<number, () => Promise<void>>;
  logFilters: IntelFilters;
  // Maps system_id -> latest report unix timestamp (seconds)
  lastIntelSystems: Record<number, number>;
  // Maps system_id -> latest clear report unix timestamp (seconds)
  lastClearSystems: Record<number, number>;
  setReports: (reports: IntelReport[]) => void;
  pushReport: (report: IntelReport) => void;
  setUploaders: (count: number) => void;
  setVersion: (version: string) => void;
  setIntelStatus: (status: IntelState["intelStatus"]) => void;
  ensureUploaderPresenceFresh: () => Promise<void>;
  setLogFilters: (filters: Partial<IntelFilters>) => void;
  toggleSystemFilter: (systemId: number) => void;
  clearFilters: () => void;
  connectRealtime: () => Promise<"ok" | "auth_error" | "error">;
  disconnectRealtime: () => Promise<void>;
  syncZKillRealtime: (regionIds: number[], enabled: boolean) => Promise<void>;
};

const computeIntelSystemSignals = (reports: IntelReport[]) => {
  const latestBySystem: Record<number, number> = {};
  const latestClearBySystem: Record<number, number> = {};
  const ordered = [...reports].sort((a, b) => {
    if (a.time !== b.time) return a.time - b.time;
    return a.id - b.id;
  });
  for (const log of ordered) {
    if (isClearIntelReport(log)) {
      for (const system of log.systems) {
        latestClearBySystem[system.system] = log.time;
        delete latestBySystem[system.system];
      }
      continue;
    }
    for (const system of log.systems) {
      latestBySystem[system.system] = log.time;
    }
  }
  return {
    latestBySystem,
    latestClearBySystem,
  };
};

const normalizeReports = (reports: IntelReport[], max = 100) => {
  const nowSeconds = Math.floor(Date.now() / 1000);
  const oldestAllowed = nowSeconds - MAX_REPORT_AGE_SECONDS;
  const byId = new Map<string, IntelReport>();
  for (const report of reports) {
    const reportId = report?.id;
    if (!reportId) continue;
    if (!Number.isFinite(report.time) || report.time < oldestAllowed) {
      continue;
    }
    const key = report.recordId ?? String(reportId);
    const existing = byId.get(key);
    if (!existing || report.time > existing.time) {
      byId.set(key, report);
    }
  }
  return Array.from(byId.values())
    .sort((a, b) => {
      if (b.time !== a.time) return b.time - a.time;
      return b.id - a.id;
    })
    .slice(0, max);
};

const rebuildIntelReportState = (
  reports: IntelReport[],
  reportCount?: number,
) => {
  const nextReports = normalizeReports(reports);
  const { latestBySystem, latestClearBySystem } =
    computeIntelSystemSignals(nextReports);

  return {
    reports: nextReports,
    reportCount:
      typeof reportCount === "number" && reportCount > 0
        ? Math.max(reportCount, nextReports.length)
        : nextReports.length,
    lastIntelSystems: latestBySystem,
    lastClearSystems: latestClearBySystem,
    reportsFetchedAt: nextReports.length > 0 ? Date.now() : 0,
  };
};

const isZKillReport = (report: IntelReport) =>
  report.meta && typeof report.meta === "object"
    ? report.meta.source === "zkill_feed"
    : false;

type IntelRecord = {
  report_id?: number | string;
  report_time?: number | string;
  author?: string;
  text?: string;
  systems?: unknown;
  regions?: unknown;
  channel?: string;
  channel_id?: string;
  uploader_user?: string;
  id?: string;
};

export const useIntelStore = create<IntelState>()(
  persist(
    (set, get) => {
      const ensureUploadersRealtimeSubscription = async () => {
        const existing = get().uploadersRealtimeUnsubscribe;
        if (existing) {
          return existing;
        }
        const uploadersUnsubscribe = await pb.realtime.subscribe(
          INTEL_UPLOADER_COUNT_TOPIC,
          (data) => {
            const payload = normalizeUploaderCountMessage(data);
            if (payload) {
              get().setUploaders(payload.uploaders);
            }
          },
        );
        set({ uploadersRealtimeUnsubscribe: uploadersUnsubscribe });
        return uploadersUnsubscribe;
      };

      return {
        reports: [],
        reportCount: 0,
        lastReports: [],
        reportsFetchedAt: 0,
        uploaders: 0,
        version: "",
        intelStatus: "connecting",
        reportsRealtimeUnsubscribe: undefined,
        uploadersRealtimeUnsubscribe: undefined,
        keepaliveRealtimeUnsubscribe: undefined,
        zkillRegionRealtimeUnsubscribes: {},
        logFilters: {
          includeSystemLogs: true,
          includeSystemAlarm: true,
          includeUnknownLogs: true,
          includeUnknownAlarm: true,
          includeUnloadedRegionsLogs: true,
          includeUnloadedRegionsAlarm: true,
          system: [],
        },
        lastIntelSystems: {},
        lastClearSystems: {},
        setReports: (reports) =>
          set((state) => {
            const nextState = rebuildIntelReportState(reports, state.reportCount);
            return {
              ...nextState,
              lastReports: [],
            };
          }),
        pushReport: (report) => {
          const existing = get().reports;
          const key = report.recordId ?? String(report.id);
          if (
            existing.some(
              (entry) => (entry.recordId ?? String(entry.id)) === key,
            )
          ) {
            return;
          }
          const nextReports = normalizeReports([report, ...existing]);
          const { latestBySystem, latestClearBySystem } =
            computeIntelSystemSignals(nextReports);
          set({
            reports: nextReports,
            reportCount: nextReports.length,
            lastReports: [report],
            lastIntelSystems: latestBySystem,
            lastClearSystems: latestClearBySystem,
            reportsFetchedAt: Date.now(),
          });
        },
        setUploaders: (count) => set({ uploaders: count }),
        setVersion: (version) => set({ version }),
        setIntelStatus: (intelStatus) => set({ intelStatus }),
        ensureUploaderPresenceFresh: async () => {
          const snapshot = get();
          if (snapshot.intelStatus !== "connected" || snapshot.uploaders > 0) {
            return;
          }
          const now = Date.now();
          if (uploaderResyncInFlight) {
            return uploaderResyncInFlight;
          }
          if (now - lastUploaderResyncAt < UPLOADER_RESYNC_THROTTLE_MS) {
            return;
          }
          lastUploaderResyncAt = now;
          uploaderResyncInFlight = (async () => {
            try {
              await ensureUploadersRealtimeSubscription();
              const res = await api.get("/intel/reports", {
                headers: { "X-Auth-Check": "1" },
              });
              const raw = res?.data?.uploaders;
              const count =
                typeof raw === "number"
                  ? raw
                  : typeof raw === "string"
                    ? Number(raw)
                    : NaN;
              if (Number.isFinite(count) && count >= 0) {
                get().setUploaders(count);
              }
            } catch {
              // best-effort self-heal for uploader count drift
            } finally {
              uploaderResyncInFlight = null;
            }
          })();
          return uploaderResyncInFlight;
        },
        setLogFilters: (filters) =>
          set((state) => ({ logFilters: { ...state.logFilters, ...filters } })),
        toggleSystemFilter: (systemId) => {
          const current = get().logFilters.system;
          const next = current.includes(systemId)
            ? current.filter((id) => id !== systemId)
            : [...current, systemId];
          set({ logFilters: { ...get().logFilters, system: next } });
        },
        clearFilters: () =>
          set({
            logFilters: {
              includeSystemLogs: true,
              includeSystemAlarm: true,
              includeUnknownLogs: true,
              includeUnknownAlarm: true,
              includeUnloadedRegionsLogs: true,
              includeUnloadedRegionsAlarm: true,
              system: [],
            },
          }),
        connectRealtime: async () => {
          if (
            get().reportsRealtimeUnsubscribe ||
            get().uploadersRealtimeUnsubscribe ||
            get().keepaliveRealtimeUnsubscribe
          ) {
            return "ok";
          }
          set({ intelStatus: "connecting" });
          try {
            if (!get().reportsFetchedAt) {
              const res = await api.get("/intel/reports", {
                headers: { "X-Auth-Check": "1" },
              });
              if (!get().reportsFetchedAt) {
                const reports = Array.isArray(res.data.intel)
                  ? res.data.intel
                      .map((report: unknown) => normalizeIntelReport(report))
                      .filter(
                        (report: IntelReport | null): report is IntelReport =>
                          report !== null,
                      )
                  : [];
                get().setReports(reports);
                get().setUploaders(res.data.uploaders ?? 0);
                get().setVersion(res.data.version ?? "");
              }
            }
            const reportsUnsubscribe = await pb
              .collection("intel_reports")
              .subscribe("*", (event) => {
                if (event.action !== "create") return;
                const record = event.record as IntelRecord;
                const report = normalizeIntelReport({
                  ...record,
                  recordId: event.record?.id,
                });
                if (report) {
                  get().pushReport(report);
                  if (get().uploaders === 0) {
                    void get().ensureUploaderPresenceFresh();
                  }
                }
              });
            const uploadersUnsubscribe =
              await ensureUploadersRealtimeSubscription();
            const keepaliveUnsubscribe = await pb.realtime.subscribe(
              REALTIME_KEEPALIVE_TOPIC,
              () => {
                // Keepalive events intentionally carry no UI state updates.
              },
            );
            set({
              reportsRealtimeUnsubscribe: reportsUnsubscribe,
              uploadersRealtimeUnsubscribe: uploadersUnsubscribe,
              keepaliveRealtimeUnsubscribe: keepaliveUnsubscribe,
              intelStatus: "connected",
            });
            return "ok";
          } catch (error: unknown) {
            await get().disconnectRealtime();
            const status = getHttpStatus(error);
            if (status === 401 || status === 403) {
              return "auth_error";
            }
            set({ intelStatus: "disconnected" });
            return "error";
          }
        },
        disconnectRealtime: async () => {
          const reportsUnsubscribe = get().reportsRealtimeUnsubscribe;
          const uploadersUnsubscribe = get().uploadersRealtimeUnsubscribe;
          const keepaliveUnsubscribe = get().keepaliveRealtimeUnsubscribe;
          const zkillUnsubscribes = Object.values(
            get().zkillRegionRealtimeUnsubscribes,
          );
          set({
            reportsRealtimeUnsubscribe: undefined,
            uploadersRealtimeUnsubscribe: undefined,
            keepaliveRealtimeUnsubscribe: undefined,
            zkillRegionRealtimeUnsubscribes: {},
            intelStatus: "disconnected",
          });
          if (reportsUnsubscribe) {
            await reportsUnsubscribe().catch(() => undefined);
          }
          if (uploadersUnsubscribe) {
            await uploadersUnsubscribe().catch(() => undefined);
          }
          if (keepaliveUnsubscribe) {
            await keepaliveUnsubscribe().catch(() => undefined);
          }
          await Promise.all(
            zkillUnsubscribes.map((unsubscribe) =>
              unsubscribe().catch(() => undefined),
            ),
          );
        },
        syncZKillRealtime: async (regionIds, enabled) => {
          const wanted = new Set(
            enabled && get().intelStatus === "connected"
              ? regionIds.filter((id) => Number.isFinite(id) && id > 0)
              : [],
          );
          const current = get().zkillRegionRealtimeUnsubscribes;
          const next = { ...current };

          await Promise.all(
            Object.entries(current).map(async ([regionIdRaw, unsubscribe]) => {
              const regionId = Number(regionIdRaw);
              if (wanted.has(regionId)) {
                return;
              }
              await unsubscribe().catch(() => undefined);
              delete next[regionId];
            }),
          );

          for (const regionId of wanted) {
            if (next[regionId]) {
              continue;
            }
            const unsubscribe = await pb.realtime.subscribe(
              zkillRegionTopic(regionId),
              (data) => {
                const report = normalizeIntelReport(data);
                if (!report) {
                  return;
                }
                get().pushReport(report);
              },
            );
            next[regionId] = unsubscribe;
          }

          const subscribedRegions = new Set(Object.keys(next).map(Number));
          set((state) => {
            const filtered = state.reports.filter((report) => {
              if (!isZKillReport(report)) {
                return true;
              }
              if (!enabled) {
                return false;
              }
              return report.regions.some((region) =>
                subscribedRegions.has(region),
              );
            });
            const nextReports = normalizeReports(filtered);
            const { latestBySystem, latestClearBySystem } =
              computeIntelSystemSignals(nextReports);
            return {
              zkillRegionRealtimeUnsubscribes: next,
              reports: nextReports,
              lastIntelSystems: latestBySystem,
              lastClearSystems: latestClearBySystem,
              reportCount: nextReports.length,
            };
          });
        },
      };
    },
    {
      name: "intel-map-config/intel",
      version: INTEL_STORE_VERSION,
      migrate: (persistedState) => {
        const state = (persistedState ?? {}) as Partial<IntelState>;
        const persistedFilters =
          state.logFilters ?? ({} as Partial<IntelFilters>);
        return {
          reports: [],
          reportCount: 0,
          lastReports: [],
          reportsFetchedAt: 0,
          uploaders: 0,
          version: "",
          intelStatus: "connecting",
          reportsRealtimeUnsubscribe: undefined,
          uploadersRealtimeUnsubscribe: undefined,
          keepaliveRealtimeUnsubscribe: undefined,
          zkillRegionRealtimeUnsubscribes: {},
          logFilters: {
            includeSystemLogs: true,
            includeSystemAlarm: true,
            includeUnknownLogs: true,
            includeUnknownAlarm: true,
            includeUnloadedRegionsLogs: true,
            includeUnloadedRegionsAlarm: true,
            system: [],
            ...persistedFilters,
          },
          lastIntelSystems: {},
          lastClearSystems: {},
        } as unknown as IntelState;
      },
      partialize: (state) => ({ logFilters: state.logFilters }),
    },
  ),
);
