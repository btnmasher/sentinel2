import { useEffect } from "react";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { useAuthStore } from "@/app/store/authStore";
import { getHttpStatus } from "@/utils/httpError";
import { useIntelStore } from "../store/intelStore";
import { normalizeIntelReport } from "../utils/intelReportUtils";
import {
  INTEL_UPLOADER_COUNT_TOPIC,
  normalizeUploaderCountMessage,
} from "../utils/intelRealtimeUtils";

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
              // Ignore stale initial fetch if reports were populated meanwhile (e.g. local debug seed).
              if (useIntelStore.getState().reportsFetchedAt) {
                return;
              }
              const reports = Array.isArray(res.data.intel)
                ? res.data.intel
                    .map((report: unknown) => normalizeIntelReport(report))
                    .filter(
                      (report): report is NonNullable<typeof report> =>
                        report !== null,
                    )
                : [];
              setReports(reports);
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
      } catch (error: unknown) {
        const status = getHttpStatus(error);
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
        await pb.realtime.subscribe(INTEL_UPLOADER_COUNT_TOPIC, (data) => {
          const payload = normalizeUploaderCountMessage(data);
          if (payload) {
            setUploaders(payload.uploaders);
          }
        });
      } catch (error: unknown) {
        const status = getHttpStatus(error);
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
      const ignoreMissingClient = (error: unknown) => {
        if (getHttpStatus(error) === 404) {
          return;
        }
        throw error;
      };
      if (pb.realtime?.isConnected) {
        void pb
          .collection("intel_reports")
          .unsubscribe()
          .catch(ignoreMissingClient);
        void pb.realtime
          .unsubscribe(INTEL_UPLOADER_COUNT_TOPIC)
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
