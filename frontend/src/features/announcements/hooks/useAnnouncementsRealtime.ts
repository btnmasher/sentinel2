import { useEffect } from "react";
import { useAuthStore } from "@/app/store/authStore";
import { useAnnouncementsStore } from "../store/useAnnouncementsStore";

export default function useAnnouncementsRealtime() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const clear = useAnnouncementsStore((s) => s.clear);
  const loadLatest = useAnnouncementsStore((s) => s.loadLatest);
  const startRealtime = useAnnouncementsStore((s) => s.startRealtime);
  const stopRealtime = useAnnouncementsStore((s) => s.stopRealtime);

  useEffect(() => {
    if (!isAuthenticated) {
      void stopRealtime();
      clear();
      return;
    }
    void loadLatest();
    void startRealtime();
    return () => {
      void stopRealtime();
    };
  }, [clear, isAuthenticated, loadLatest, startRealtime, stopRealtime]);
}
