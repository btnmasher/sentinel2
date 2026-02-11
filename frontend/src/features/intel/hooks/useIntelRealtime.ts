import { useEffect } from "react";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { useAuthStore } from "@/app/store/authStore";
import { useIntelStore } from "../store/intelStore";
import { normalizeIntelReport } from "../utils/intelReportUtils";

type IntelRecord = {
  report_id?: number | string;
  report_time?: number | string;
  author?: string;
  text?: string;
  systems?: unknown;
  regions?: unknown;
  uploader_user?: string;
  id?: string;
};

export default function useIntelRealtime() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const setIntelStatus = useIntelStore((s) => s.setIntelStatus);
  const pushReport = useIntelStore((s) => s.pushReport);
  const setUploaders = useIntelStore((s) => s.setUploaders);
  const setVersion = useIntelStore((s) => s.setVersion);
  const setReports = useIntelStore((s) => s.setReports);
  const reportsFetchedAt = useIntelStore((s) => s.reportsFetchedAt);

  useEffect(() => {
    if (!isAuthenticated) {
      setIntelStatus("disconnected");
      return;
    }

    let active = true;
    setIntelStatus("connecting");

    const subscribeReports = async () => {
      try {
        if (!reportsFetchedAt) {
          api
            .get("/intel/reports", { headers: { "X-Auth-Check": "1" } })
            .then((res) => {
              const reports = Array.isArray(res.data.intel)
                ? res.data.intel
                    .map((report: unknown) => normalizeIntelReport(report))
                    .filter(Boolean)
                : [];
              setReports(reports as any);
              setUploaders(res.data.uploaders ?? 0);
              setVersion(res.data.version ?? "");
            })
            .catch(() => undefined);
        }
        await pb.collection("intel_reports").subscribe("*", (event) => {
          if (event.action !== "create") return;
          const record = event.record as IntelRecord;
          const report = normalizeIntelReport({
            ...record,
            recordId: event.record?.id,
          });
          if (report) {
            pushReport(report);
          }
        });
        if (active) {
          setIntelStatus("connected");
        }
      } catch (error: any) {
        const status = error?.status || error?.response?.status;
        if (status === 401 || status === 403) {
          useAuthStore
            .getState()
            .forceLogout("Authentication expired, returning to home.");
          return;
        }
        if (active) {
          setIntelStatus("disconnected");
        }
      }
    };

    const subscribeUploaders = async () => {
      try {
        await pb.collection("intel_uploaders").subscribe("*", () => {
          api
            .get("/intel/meta", { headers: { "X-Auth-Check": "1" } })
            .then((res) => {
              setUploaders(res.data.uploaders ?? 0);
              if (res.data.version) {
                setVersion(res.data.version);
              }
            })
            .catch(() => undefined);
        });
      } catch (error: any) {
        const status = error?.status || error?.response?.status;
        if (status === 401 || status === 403) {
          useAuthStore
            .getState()
            .forceLogout("Authentication expired, returning to home.");
        }
        // Ignore; uploaders count will still refresh on page load.
      }
    };

    void subscribeReports();
    void subscribeUploaders();

    return () => {
      active = false;
      setIntelStatus("disconnected");
      const ignoreMissingClient = (error: any) => {
        if (error?.status === 404) {
          return;
        }
        throw error;
      };
      if (pb.realtime?.isConnected) {
        void pb
          .collection("intel_reports")
          .unsubscribe()
          .catch(ignoreMissingClient);
        void pb
          .collection("intel_uploaders")
          .unsubscribe()
          .catch(ignoreMissingClient);
      }
    };
  }, [
    isAuthenticated,
    pushReport,
    reportsFetchedAt,
    setReports,
    setIntelStatus,
    setUploaders,
    setVersion,
  ]);
}
