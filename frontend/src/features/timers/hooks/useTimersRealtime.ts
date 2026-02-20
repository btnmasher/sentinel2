import { useEffect } from "react";
import { useAuthStore } from "@/app/store/authStore";
import { useTimersStore } from "../store/useTimersStore";

export default function useTimersRealtime() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const loadTimers = useTimersStore((s) => s.loadTimers);
  const startRealtime = useTimersStore((s) => s.startRealtime);
  const stopRealtime = useTimersStore((s) => s.stopRealtime);

  useEffect(() => {
    if (!isAuthenticated) {
      void stopRealtime();
      return;
    }
    void loadTimers();
    void startRealtime();
    return () => {
      void stopRealtime();
    };
  }, [isAuthenticated, loadTimers, startRealtime, stopRealtime]);
}
