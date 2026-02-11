import { create } from "zustand";
import { persist } from "zustand/middleware";
import { ensurePersistReset } from "@/app/store/persistReset";
import type { IntelReport } from "../types";

export const INTEL_STORE_VERSION = 1;

type IntelFilters = {
  includeUnknownLogs: boolean;
  includeUnknownAlarm: boolean;
  includeUnloadedRegionsLogs: boolean;
  includeUnloadedRegionsAlarm: boolean;
  system: number[];
};

type IntelState = {
  reports: IntelReport[];
  lastReports: IntelReport[];
  reportsFetchedAt?: number;
  uploaders: number;
  version: string;
  intelStatus: "connecting" | "connected" | "disconnected";
  logFilters: IntelFilters;
  lastIntelSystems: Record<number, number>;
  setReports: (reports: IntelReport[]) => void;
  pushReport: (report: IntelReport) => void;
  setUploaders: (count: number) => void;
  setVersion: (version: string) => void;
  setIntelStatus: (status: IntelState["intelStatus"]) => void;
  setLogFilters: (filters: Partial<IntelFilters>) => void;
  toggleSystemFilter: (systemId: number) => void;
  clearFilters: () => void;
};

const computeLastIntelSystems = (reports: IntelReport[]) => {
  const now = Date.now();
  const recent = reports
    .map((log) => ({
      ...log,
      age: Math.floor((now - log.time * 1000) / 60000),
    }))
    .filter((log) => log.age < 15)
    .sort((a, b) => b.age - a.age)
    .flatMap((log) => log.systems.map((s) => [s.system, log.age] as const));

  return Object.fromEntries(recent);
};

const normalizeReports = (reports: IntelReport[], max = 100) => {
  const byId = new Map<string, IntelReport>();
  for (const report of reports) {
    const reportId = report?.id;
    if (!reportId) continue;
    const key = report.recordId ?? String(reportId);
    const existing = byId.get(key);
    if (!existing || report.time > existing.time) {
      byId.set(key, report);
    }
  }
  return Array.from(byId.values())
    .sort((a, b) => b.time - a.time)
    .slice(0, max);
};

ensurePersistReset();

export const useIntelStore = create<IntelState>()(
  persist(
    (set, get) => ({
      reports: [],
      lastReports: [],
      reportsFetchedAt: undefined,
      uploaders: 0,
      version: "",
      intelStatus: "connecting",
      logFilters: {
        includeUnknownLogs: true,
        includeUnknownAlarm: true,
        includeUnloadedRegionsLogs: true,
        includeUnloadedRegionsAlarm: true,
        system: [],
      },
      lastIntelSystems: {},
      setReports: (reports) =>
        set(() => {
          const nextReports = normalizeReports(reports);
          return {
            reports: nextReports,
            lastReports: nextReports,
            lastIntelSystems: computeLastIntelSystems(nextReports),
            reportsFetchedAt: Date.now(),
          };
        }),
      pushReport: (report) => {
        const existing = get().reports;
        const key = report.recordId ?? String(report.id);
        if (
          existing.some((entry) => (entry.recordId ?? String(entry.id)) === key)
        ) {
          return;
        }
        const nextReports = normalizeReports([report, ...existing]);
        set({
          reports: nextReports,
          lastReports: [report],
          lastIntelSystems: computeLastIntelSystems([report]),
        });
      },
      setUploaders: (count) => set({ uploaders: count }),
      setVersion: (version) => set({ version }),
      setIntelStatus: (intelStatus) => set({ intelStatus }),
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
            includeUnknownLogs: true,
            includeUnknownAlarm: true,
            includeUnloadedRegionsLogs: true,
            includeUnloadedRegionsAlarm: true,
            system: [],
          },
        }),
    }),
    {
      name: "intel-map-config/intel",
      version: INTEL_STORE_VERSION,
      partialize: (state) => ({ logFilters: state.logFilters }),
    },
  ),
);
