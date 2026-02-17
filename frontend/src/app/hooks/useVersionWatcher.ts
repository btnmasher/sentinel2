import { useEffect } from "react";
import { api } from "@/config/api";
import { useAppConfigStore } from "@/app/store/appConfigStore";

const versionCheckIntervalMs = 60_000;

type AppConfigVersionResponse = {
  version?: unknown;
};

async function fetchServerVersion(): Promise<string> {
  const res = await api.get<AppConfigVersionResponse>("/app-config", {
    headers: {
      "Cache-Control": "no-cache",
      Pragma: "no-cache",
    },
    params: { _ts: Date.now() },
  });
  return typeof res.data?.version === "string" ? res.data.version.trim() : "";
}

export default function useVersionWatcher() {
  const loaded = useAppConfigStore((s) => s.loaded);
  const setVersion = useAppConfigStore((s) => s.setVersion);

  useEffect(() => {
    if (!loaded) {
      return;
    }

    let disposed = false;

    const checkVersion = async () => {
      try {
        const serverVersion = await fetchServerVersion();
        if (disposed || serverVersion === "") {
          return;
        }

        const currentVersion = useAppConfigStore.getState().version.trim();
        if (currentVersion === "") {
          setVersion(serverVersion);
          return;
        }
        if (serverVersion !== currentVersion) {
          window.location.reload();
        }
      } catch {
        // Ignore transient failures and try again on the next interval/focus.
      }
    };

    const onVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        void checkVersion();
      }
    };

    const timer = window.setInterval(() => {
      void checkVersion();
    }, versionCheckIntervalMs);
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      disposed = true;
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [loaded, setVersion]);
}
