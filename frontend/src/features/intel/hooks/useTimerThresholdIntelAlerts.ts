import { useEffect, useRef } from "react";
import { useAuthStore } from "@/app/store/authStore";
import { useTimersStore } from "@/features/timers";
import { useIntelStore } from "../store/intelStore";
import type { IntelReport } from "../types";

const NEAR_EXPIRY_SECONDS = 30 * 60;
const CRITICAL_EXPIRY_SECONDS = 10 * 60;
const EVALUATION_INTERVAL_MS = 15_000;
type Threshold = "near" | "critical";

type TimerAlertRecord = {
  id: string;
  title: string;
  system_id: number;
  system_name: string;
  region_id: number;
  status: string;
  expires_at: string;
};

export default function useTimerThresholdIntelAlerts() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const timers = useTimersStore((s) => s.timers);
  const timersLoadedAt = useTimersStore((s) => s.loadedAt);
  const pushReport = useIntelStore((s) => s.pushReport);

  const timersRef = useRef<TimerAlertRecord[]>([]);
  const previousRemainingByKeyRef = useRef<Map<string, number>>(new Map());
  const reportSequenceRef = useRef(910_000_000);
  const hasBaselineRef = useRef(false);

  useEffect(() => {
    if (!isAuthenticated) {
      timersRef.current = [];
      previousRemainingByKeyRef.current = new Map();
      hasBaselineRef.current = false;
      return;
    }

    const emitThresholdReport = (
      timer: TimerAlertRecord,
      threshold: Threshold,
    ) => {
      const nextID = reportSequenceRef.current++;
      const report = buildThresholdReport(timer, threshold, nextID);
      pushReport(report);
    };

    const evaluateThresholdCrossings = (emit: boolean) => {
      const nowSeconds = Math.floor(Date.now() / 1000);
      const currentRemainingByKey = new Map<string, number>();

      for (const timer of timersRef.current) {
        if (timer.status !== "active") continue;
        const expiresAtSeconds = Math.floor(
          Date.parse(timer.expires_at) / 1000,
        );
        if (!Number.isFinite(expiresAtSeconds)) continue;
        const remainingSeconds = expiresAtSeconds - nowSeconds;
        if (remainingSeconds <= 0) continue;

        const key = `${timer.id}:${timer.expires_at}`;
        currentRemainingByKey.set(key, remainingSeconds);

        const previousRemaining = previousRemainingByKeyRef.current.get(key);
        if (!emit || previousRemaining === undefined) {
          continue;
        }

        if (
          previousRemaining > NEAR_EXPIRY_SECONDS &&
          remainingSeconds <= NEAR_EXPIRY_SECONDS
        ) {
          emitThresholdReport(timer, "near");
        }
        if (
          previousRemaining > CRITICAL_EXPIRY_SECONDS &&
          remainingSeconds <= CRITICAL_EXPIRY_SECONDS
        ) {
          emitThresholdReport(timer, "critical");
        }
      }

      previousRemainingByKeyRef.current = currentRemainingByKey;
    };

    if (!timersLoadedAt) {
      return;
    }
    timersRef.current = timers.map((timer) => ({
      id: timer.id,
      title: timer.title,
      system_id: timer.system_id,
      system_name: timer.system_name,
      region_id: timer.region_id,
      status: timer.status,
      expires_at: timer.expires_at,
    }));
    evaluateThresholdCrossings(hasBaselineRef.current);
    hasBaselineRef.current = true;

    const evaluateIntervalID = window.setInterval(() => {
      evaluateThresholdCrossings(true);
    }, EVALUATION_INTERVAL_MS);

    return () => {
      window.clearInterval(evaluateIntervalID);
    };
  }, [isAuthenticated, pushReport, timers, timersLoadedAt]);
}

function buildThresholdReport(
  timer: TimerAlertRecord,
  threshold: Threshold,
  id: number,
): IntelReport {
  const thresholdText =
    threshold === "critical"
      ? "critical expiry window (<10m)"
      : "near-expiry window (<30m)";
  const systemName =
    (timer.system_name || "").trim() || `System ${timer.system_id}`;
  const timerName = stripLeadingSystemName(
    (timer.title || "").trim(),
    systemName,
  );
  const text =
    timerName.length > 0
      ? `${systemName} ${timerName} entered ${thresholdText}.`
      : `${systemName} timer entered ${thresholdText}.`;

  return {
    recordId: `timer-threshold:${timer.id}:${threshold}:${timer.expires_at}`,
    id,
    time: Math.floor(Date.now() / 1000),
    author: "Timer Monitor",
    text,
    channel_id: "timers",
    systems: [
      {
        system: timer.system_id,
        name: systemName,
        constellation: 0,
        region: timer.region_id,
      },
    ],
    regions: timer.region_id > 0 ? [timer.region_id] : [],
  };
}

function stripLeadingSystemName(title: string, systemName: string): string {
  if (!title || !systemName) {
    return title;
  }
  const normalizedTitle = title.toLowerCase();
  const normalizedSystem = systemName.toLowerCase();
  if (!normalizedTitle.startsWith(normalizedSystem)) {
    return title;
  }
  const suffix = title.slice(systemName.length).trimStart();
  return suffix.length > 0 ? suffix : title;
}
