import { useState } from "react";
import { api } from "@/config/api";
import { useUIStore } from "@/app/store/uiStore";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";

type JumpbridgeImportModalProps = {
  onImported: () => Promise<void>;
};

const parseJumpbridgeInput = (text: string) =>
  text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const match = line.match(
        /^(.+?)\s*(?:»|->|-->|—>|=>|→)\s*(.+?)(?:\s+-\s+.*)?$/,
      );
      if (!match) return null;
      const from = match[1].trim();
      const to = match[2].trim();
      if (!from || !to) return null;
      return `${from} --> ${to}`;
    })
    .filter((line): line is string => Boolean(line));

const getErrorDetail = (error: unknown): string => {
  if (!error || typeof error !== "object") return "Unknown error";
  const maybeResponse = (error as { response?: unknown }).response;
  if (maybeResponse && typeof maybeResponse === "object") {
    const response = maybeResponse as { data?: unknown };
    if (response.data && typeof response.data === "object") {
      const message = (response.data as { message?: unknown }).message;
      if (typeof message === "string" && message.trim() !== "") {
        return message;
      }
    }
    if (typeof response.data === "string" && response.data.trim() !== "") {
      return response.data;
    }
  }
  const message = (error as { message?: unknown }).message;
  if (typeof message === "string" && message.trim() !== "") {
    return message;
  }
  return "Unknown error";
};

export default function JumpbridgeImportModal({
  onImported,
}: JumpbridgeImportModalProps) {
  const { close } = useModalBody();
  const setToast = useUIStore((s) => s.setToast);
  const [importing, setImporting] = useState(false);
  const [jumpbridgeText, setJumpbridgeText] = useState("");

  const importJumpbridges = async () => {
    const parsed = parseJumpbridgeInput(jumpbridgeText);
    if (parsed.length === 0) {
      setToast({
        text: "Jumpbridge import failed: empty input.",
        color: "error",
      });
      return;
    }
    setImporting(true);
    try {
      await api.post("/staff/jumpbridges/import", {
        jumpbridges: parsed.join("\n"),
      });
      setToast({
        text: "Jumpbridge import succeeded.",
        color: "success",
      });
      await onImported();
      close();
    } catch (error: unknown) {
      const detail = getErrorDetail(error);
      setToast({
        text: `Jumpbridge import failed: ${detail || "Unknown error"}`,
        color: "error",
      });
    } finally {
      setImporting(false);
    }
  };

  return (
    <>
      <p className="text-sm text-slate-400">
        Paste the jumpbridge list (structure_id FROM --&gt; TO). This will
        replace existing jumpbridges.
      </p>
      <textarea
        className="textarea textarea-bordered bg-base-300 flex-1 w-full"
        value={jumpbridgeText}
        onChange={(e) => setJumpbridgeText(e.target.value)}
      />
      <div className="modal-action mt-2">
        <button
          className="btn btn-sm btn-outline"
          onClick={() => close()}
          disabled={importing}
        >
          Cancel
        </button>
        <button
          className="btn btn-sm btn-primary btn-outline"
          onClick={importJumpbridges}
          disabled={importing}
        >
          {importing ? "Importing..." : "Import"}
        </button>
      </div>
    </>
  );
}
