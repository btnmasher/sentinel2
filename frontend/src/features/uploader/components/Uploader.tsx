import { useEffect, useMemo, useState } from "react";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import useConfirm from "@/app/hooks/useConfirm";
import { useUIStore } from "@/app/store/uiStore";
import Panel from "@/components/Panel";
import { getErrorMessage } from "@/utils/httpError";
import { useShallow } from "zustand/shallow";

const DOWNLOADS = [
  {
    key: "linux",
    label: "Linux",
    icon: "/svg/linux.svg",
    iconClassName: "os-icon os-icon-linux",
  },
  {
    key: "windows",
    label: "Windows",
    icon: "/svg/windows.svg",
    iconClassName: "os-icon os-icon-windows",
  },
  {
    key: "macos",
    label: "macOS",
    icon: "/svg/macos.svg",
    iconClassName: "os-icon os-icon-apple",
  },
] as const;

type DownloadKey = (typeof DOWNLOADS)[number]["key"];
type DownloadLinks = Record<DownloadKey, string>;

export default function Uploader() {
  const [token, setToken] = useState("");
  const [channels, setChannels] = useState<string[]>([]);
  const [downloadLinks, setDownloadLinks] = useState<DownloadLinks>({
    linux: "",
    windows: "",
    macos: "",
  });
  const [releasePageUrl, setReleasePageURL] = useState("");
  const [regenerating, setRegenerating] = useState(false);
  const requestConfirm = useConfirm();
  const { setToast } = useUIStore(
    useShallow((s) => ({
      setToast: s.setToast,
    })),
  );
  const baseUrl = useMemo(() => window.location.origin, []);
  const exampleBase = baseUrl;
  const exampleToken = token || "<YOUR_TOKEN>";
  const defaultReleasePageUrl = useMemo(
    () => "https://github.com/btnmasher/sentinel2-uploader/releases/latest",
    [],
  );

  useEffect(() => {
    api
      .get("/auth/uploader-token")
      .then((res) => setToken(res.data.token || ""))
      .catch((error) => {
        const detail =
          error?.response?.data?.message ||
          error?.response?.data ||
          error?.message;
        setToast({
          text: detail || "Unable to load uploader token",
          color: "error",
        });
      });
    pb.collection("intel_channels")
      .getFullList({ sort: "channel_name" })
      .then((records) =>
        setChannels(
          records
            .map((record) => String(record.channel_name || ""))
            .filter(Boolean),
        ),
      )
      .catch(() => setChannels([]));
    api
      .get("/uploader/download-links")
      .then((res) => {
        setDownloadLinks({
          linux:
            typeof res.data?.linux_url === "string" ? res.data.linux_url : "",
          windows:
            typeof res.data?.windows_url === "string"
              ? res.data.windows_url
              : "",
          macos:
            typeof res.data?.macos_url === "string" ? res.data.macos_url : "",
        });
        setReleasePageURL(
          typeof res.data?.release_page_url === "string"
            ? res.data.release_page_url
            : "",
        );
      })
      .catch(() => {
        setReleasePageURL("");
      });
  }, []);

  const regenerate = async () => {
    if (!token) {
      setToast({
        text: "Uploader token revoked. Contact an admin.",
        color: "warning",
      });
      return;
    }
    setRegenerating(true);
    try {
      const res = await api.post("/auth/uploader-token/rotate");
      setToken(res.data.token || "");
    } catch (error: unknown) {
      const detail = getErrorMessage(error, "Failed to regenerate token");
      setToast({
        text: detail,
        color: "error",
      });
    } finally {
      setRegenerating(false);
    }
  };

  const confirmRegenerate = () =>
    requestConfirm({
      title: "Regenerate token",
      body: "This will invalidate the previous token.",
      onConfirm: () => {
        void regenerate();
      },
      confirmLabel: "Regenerate",
      cancelLabel: "Keep current",
      tone: "danger",
    });

  const copyText = async (label: string, value: string) => {
    if (!value) {
      setToast({ text: `Nothing to copy for ${label}.`, color: "warning" });
      return;
    }
    try {
      await navigator.clipboard.writeText(value);
      setToast({ text: `${label} copied`, color: "info" });
    } catch {
      setToast({ text: `Failed to copy ${label}`, color: "error" });
    }
  };

  return (
    <div className="grid lg:grid-cols-[2fr_1fr] gap-6">
      <Panel
        title="Intel Uploader"
        hint="Download the uploader tool, then point it at this server. The tool will pull the intel channel configuration directly from the app."
      >
        <div className="grid gap-3 mt-4">
          <div className="grid gap-2">
            <div className="text-xs uppercase tracking-[0.2em] text-slate-500">
              Uploader Downloads
            </div>
            <div className="flex flex-wrap gap-2">
              {DOWNLOADS.map((item) => (
                <a
                  key={item.label}
                  className="btn btn-sm btn-outline gap-2"
                  href={
                    downloadLinks[item.key] ||
                    releasePageUrl ||
                    defaultReleasePageUrl
                  }
                  target="_blank"
                  rel="noreferrer"
                  download={downloadLinks[item.key] ? "" : undefined}
                >
                  <img
                    src={item.icon}
                    alt={`${item.label} logo`}
                    className={item.iconClassName}
                    loading="lazy"
                  />
                  {item.label}
                </a>
              ))}
            </div>
            <p className="text-xs text-slate-500">
              Download the uploader for your platform.
            </p>
          </div>

          <div className="grid gap-2">
            <div className="text-xs uppercase tracking-[0.2em] text-slate-500">
              Uploader Base URL
            </div>
            <div className="bg-base-300/60 border border-slate-700 rounded-lg p-3 font-mono text-xs break-all">
              {baseUrl}
            </div>
            <button
              className="btn btn-sm btn-info btn-outline justify-self-start"
              onClick={() => copyText("Base URL", baseUrl)}
            >
              Copy base URL
            </button>
          </div>

          <div className="grid gap-2">
            <div className="text-sm uppercase tracking-[0.2em] text-slate-500">
              Uploader Token
            </div>
            <div className="bg-base-300/60 border border-slate-700 rounded-lg p-3 font-mono text-xs break-all">
              {token || "Loading token..."}
            </div>
            <div className="flex w-full items-center gap-2">
              <button
                className="btn btn-sm btn-info btn-outline"
                onClick={() => copyText("Token", token)}
                disabled={!token}
              >
                Copy token
              </button>
              <button
                className="btn btn-sm btn-error btn-outline ml-auto"
                onClick={confirmRegenerate}
                disabled={regenerating || !token}
              >
                {regenerating ? "Regenerating..." : "Regenerate token"}
              </button>
            </div>
            <p className="text-xs text-slate-500">
              Regenerating invalidates previous tokens.
            </p>
          </div>

          <div className="grid gap-2">
            <div className="text-xs uppercase tracking-[0.2em] text-slate-500">
              Example Usage
            </div>
            <pre className="bg-base-300/60 border border-slate-700 rounded-lg p-3 font-mono text-xs whitespace-pre-wrap">
              {`# Flags
sentinel2-uploader --base-url ${exampleBase} --token ${exampleToken}

# Env vars
SENTINEL_BASE_URL=${exampleBase}
SENTINEL_TOKEN=${exampleToken}
sentinel2-uploader`}
            </pre>
            <div className="flex flex-wrap gap-2">
              <button
                className="btn btn-sm btn-info btn-outline"
                onClick={() =>
                  copyText(
                    "Flags example",
                    `--base-url ${exampleBase} --token ${exampleToken}`,
                  )
                }
              >
                Copy flags
              </button>
              <button
                className="btn btn-sm btn-info btn-outline"
                onClick={() =>
                  copyText(
                    "Env example",
                    `SENTINEL_BASE_URL=${exampleBase}\nSENTINEL_TOKEN=${exampleToken}\nsentinel2-uploader`,
                  )
                }
              >
                Copy env
              </button>
            </div>
          </div>
        </div>
      </Panel>
      <Panel
        title="Active Channels"
        hint="Channel configuration updates automatically."
        className="h-fit"
      >
        <ul className="space-y-2 text-sm">
          {channels.length === 0 && (
            <li className="text-slate-500">No channels configured.</li>
          )}
          {channels.map((channel) => (
            <li
              key={channel}
              className="flex items-center justify-between rounded-lg border border-slate-800/70 bg-base-300/40 px-3 py-2"
            >
              <span className="font-medium text-slate-100">{channel}</span>
            </li>
          ))}
        </ul>
      </Panel>
    </div>
  );
}
