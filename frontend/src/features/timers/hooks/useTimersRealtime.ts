import { useEffect } from "react";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import { useTimersStore } from "../store/useTimersStore";

export default function useTimersRealtime() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const timersEnabled = useAppConfigStore((s) => s.timersEnabled);
  const loadTimers = useTimersStore((s) => s.loadTimers);
  const startRealtime = useTimersStore((s) => s.startRealtime);
  const stopRealtime = useTimersStore((s) => s.stopRealtime);

  useEffect(() => {
    if (!timersEnabled || !isAuthenticated) {
      void stopRealtime();
      return;
    }
    void loadTimers();
    void startRealtime();
    return () => {
      void stopRealtime();
    };
  }, [isAuthenticated, loadTimers, startRealtime, stopRealtime, timersEnabled]);
}
