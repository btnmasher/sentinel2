import { useEffect, useMemo, useState } from "react";
import ReactMarkdown from "react-markdown";
import rehypeSanitize from "rehype-sanitize";
import remarkGfm from "remark-gfm";
import useModal from "@/app/hooks/useModal";
import { UI_DIALOG, useUIStore } from "@/app/store/uiStore";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import { useAuthStore } from "@/app/store/authStore";
import useAnnouncementsRealtime from "./hooks/useAnnouncementsRealtime";
import { useAnnouncementsStore } from "./store/useAnnouncementsStore";
import { markdownToPlainText } from "./markdown";

const dismissedKeyPrefix = "site-announcement:dismissed:";

const dismissedStorageKey = (userId: string | null): string =>
  `${dismissedKeyPrefix}${userId || "anon"}`;

function AnnouncementModalDismissAction({ id }: { id: string }) {
  const { close } = useModalBody();
  const [countdown, setCountdown] = useState(3);

  useEffect(() => {
    setCountdown(3);
    const timer = window.setInterval(() => {
      setCountdown((value) => {
        if (value <= 1) {
          window.clearInterval(timer);
          return 0;
        }
        return value - 1;
      });
    }, 1000);
    return () => window.clearInterval(timer);
  }, [id]);

  return (
    <button
      className="btn btn-sm btn-outline"
      disabled={countdown > 0}
      onClick={() => close("button")}
    >
      {countdown > 0 ? `${countdown}` : "Dismiss"}
    </button>
  );
}

export default function SiteAnnouncementHost() {
  const userId = useAuthStore((s) => s.userId);
  const helpOpen = useUIStore((s) => s.dialogs[UI_DIALOG.Help]);
  const announcement = useAnnouncementsStore((s) => s.announcement);
  const [dismissedId, setDismissedId] = useState("");
  const [canOpenModal, setCanOpenModal] = useState(true);
  useAnnouncementsRealtime();

  useEffect(() => {
    if (helpOpen) {
      setCanOpenModal(false);
      return;
    }
    const timer = window.setTimeout(() => setCanOpenModal(true), 200);
    return () => window.clearTimeout(timer);
  }, [helpOpen]);

  useEffect(() => {
    const key = dismissedStorageKey(userId);
    try {
      const saved = localStorage.getItem(key) || "";
      setDismissedId(saved.trim());
    } catch {
      setDismissedId("");
    }
  }, [userId]);

  const visible = Boolean(announcement && announcement.id !== dismissedId);
  const plainText = useMemo(
    () => (announcement ? markdownToPlainText(announcement.message) : ""),
    [announcement],
  );

  const dismiss = () => {
    if (!announcement) return;
    const key = dismissedStorageKey(userId);
    try {
      localStorage.setItem(key, announcement.id);
    } catch {
      // ignore storage errors
    }
    setDismissedId(announcement.id);
  };

  const modalAnnouncement =
    visible && announcement?.variant === "modal" && canOpenModal
      ? announcement
      : null;

  useModal({
    open: Boolean(modalAnnouncement),
    onDismiss: dismiss,
    build: () => ({
      title: "Announcement",
      sizeClass: "max-w-2xl",
      closeOnOverlay: false,
      body: (
        <div className="announcement-markdown">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeSanitize]}
            components={{
              a: ({ node, ...props }) => {
                void node;
                return (
                  <a {...props} target="_blank" rel="noopener noreferrer" />
                );
              },
            }}
          >
            {modalAnnouncement?.message || ""}
          </ReactMarkdown>
        </div>
      ),
      actions: (
        <AnnouncementModalDismissAction id={modalAnnouncement?.id || ""} />
      ),
    }),
  });

  if (!visible || !announcement) {
    return null;
  }

  if (announcement.variant === "banner") {
    return (
      <div className="mx-6 mt-3 rounded-lg border border-sky-700/60 bg-sky-900/25 px-3 py-2 text-sm text-sky-100 flex items-start gap-3">
        <div className="flex-1">{plainText}</div>
        <button className="btn btn-xs btn-ghost" onClick={dismiss}>
          Dismiss
        </button>
      </div>
    );
  }

  return null;
}
