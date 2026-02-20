import { useEffect } from "react";
import { useAuthStore } from "@/app/store/authStore";
import { useIntelStore } from "../store/intelStore";

export default function useIntelRealtime() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const setIntelStatus = useIntelStore((s) => s.setIntelStatus);
  const connectRealtime = useIntelStore((s) => s.connectRealtime);
  const disconnectRealtime = useIntelStore((s) => s.disconnectRealtime);

  useEffect(() => {
    if (!isAuthenticated) {
      void disconnectRealtime();
      return;
    }

    setIntelStatus("connecting");
    void connectRealtime().then((result) => {
      if (result === "auth_error") {
        useAuthStore
          .getState()
          .forceLogout("Authentication expired, returning to home.");
      }
    });

    return () => {
      void disconnectRealtime();
    };
  }, [connectRealtime, disconnectRealtime, isAuthenticated, setIntelStatus]);
}
