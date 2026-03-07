import { useEffect, useMemo, useState } from "react";
import { BellRing, X } from "lucide-react";
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
      <div className="pointer-events-none fixed inset-x-0 top-20 z-[65] flex justify-center px-4 md:px-6">
        <section
          aria-label="Site announcement"
          className="announcement-toast pointer-events-auto w-full max-w-4xl rounded-[1.4rem] px-5 py-4 backdrop-blur-md md:px-6 md:py-5"
        >
          <div className="flex items-start gap-4">
            <div className="announcement-toast-icon mt-0.5 hidden h-11 w-11 shrink-0 items-center justify-center rounded-2xl md:inline-flex">
              <BellRing className="h-5 w-5" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="announcement-toast-label mb-1 text-[0.68rem] font-semibold uppercase tracking-[0.28em]">
                Site announcement
              </div>
              <p className="announcement-toast-text text-sm leading-6 md:text-lg md:leading-7">
                {plainText}
              </p>
            </div>
            <button
              type="button"
              className="announcement-toast-dismiss inline-flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-full transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2"
              onClick={dismiss}
              aria-label="Dismiss announcement"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </section>
      </div>
    );
  }

  return null;
}
