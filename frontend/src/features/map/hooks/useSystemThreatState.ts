import { useEffect, useMemo, useRef, useState } from "react";
import { useIntelStore } from "@/features/intel";
import { useSettingsStore } from "@/app/store/settingsStore";
import { colorForAge } from "../utils/mapUtils";

const clampSeconds = (value: number) => Math.max(0, Math.floor(value));
const CLEAR_FLASH_SECONDS = 5;

const getThreatActiveWindowSeconds = (timings: {
  flash: number;
  red: number;
  orange: number;
  yellow: number;
  green: number;
}) =>
  clampSeconds(timings.flash) +
  clampSeconds(timings.red) +
  clampSeconds(timings.orange) +
  clampSeconds(timings.yellow) +
  clampSeconds(timings.green);

export function useSystemThreatState(systemId: number) {
  const threatTimings = useSettingsStore((s) => s.settings.intel.threatTimings);
  const lastIntelSystems = useIntelStore((s) => s.lastIntelSystems);
  const lastClearSystems = useIntelStore((s) => s.lastClearSystems);
  const [intelAgeSeconds, setIntelAgeSeconds] = useState<number | undefined>(
    undefined,
  );
  const [clearFlashing, setClearFlashing] = useState(false);
  const timeoutRef = useRef<number | undefined>(undefined);
  const clearTimeoutRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (timeoutRef.current) {
      window.clearTimeout(timeoutRef.current);
      timeoutRef.current = undefined;
    }

    const reportTime = lastIntelSystems[systemId];
    if (reportTime === undefined) {
      setIntelAgeSeconds(undefined);
      return;
    }

    const activeSeconds = getThreatActiveWindowSeconds(threatTimings);
    const flashSeconds = clampSeconds(threatTimings.flash);
    const computeElapsed = () =>
      Math.max(0, Math.floor(Date.now() / 1000) - reportTime);

    const elapsed = computeElapsed();
    setIntelAgeSeconds(elapsed);
    if (activeSeconds <= 0 || elapsed >= activeSeconds) {
      setIntelAgeSeconds(undefined);
      return;
    }

    const tick = () => {
      const nextElapsed = computeElapsed();
      if (nextElapsed >= activeSeconds) {
        setIntelAgeSeconds(undefined);
        timeoutRef.current = undefined;
        return;
      }
      setIntelAgeSeconds(nextElapsed);
      timeoutRef.current = window.setTimeout(
        tick,
        nextElapsed < flashSeconds ? 1000 : 5000,
      );
    };

    timeoutRef.current = window.setTimeout(
      tick,
      elapsed < flashSeconds ? 1000 : 5000,
    );

    return () => {
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
        timeoutRef.current = undefined;
      }
    };
  }, [
    lastIntelSystems,
    systemId,
    threatTimings.flash,
    threatTimings.green,
    threatTimings.orange,
    threatTimings.red,
    threatTimings.yellow,
  ]);

  useEffect(() => {
    if (clearTimeoutRef.current) {
      window.clearTimeout(clearTimeoutRef.current);
      clearTimeoutRef.current = undefined;
    }

    const clearTime = lastClearSystems[systemId];
    if (clearTime === undefined) {
      setClearFlashing(false);
      return;
    }

    const expiresAtMs = (clearTime + CLEAR_FLASH_SECONDS) * 1000;
    const remainingMs = expiresAtMs - Date.now();
    if (remainingMs <= 0) {
      setClearFlashing(false);
      return;
    }

    setClearFlashing(true);
    clearTimeoutRef.current = window.setTimeout(() => {
      setClearFlashing(false);
      clearTimeoutRef.current = undefined;
    }, remainingMs);

    return () => {
      if (clearTimeoutRef.current) {
        window.clearTimeout(clearTimeoutRef.current);
        clearTimeoutRef.current = undefined;
      }
    };
  }, [lastClearSystems, systemId]);

  const systemFill = useMemo(
    () => colorForAge(intelAgeSeconds, threatTimings),
    [intelAgeSeconds, threatTimings],
  );

  const alerting =
    intelAgeSeconds !== undefined &&
    intelAgeSeconds < clampSeconds(threatTimings.flash);

  return { intelAgeSeconds, systemFill, alerting, clearFlashing };
}
