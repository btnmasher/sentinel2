import { useEffect, useMemo, useState } from "react";
import ReactMarkdown from "react-markdown";
import rehypeSanitize from "rehype-sanitize";
import remarkGfm from "remark-gfm";
import useModal from "@/app/hooks/useModal";
import { UI_DIALOG, useUIStore } from "@/app/store/uiStore";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import { pb } from "@/config/pb";
import { useAuthStore } from "@/app/store/authStore";
import type { SiteAnnouncement } from "./types";
import { markdownToPlainText } from "./markdown";

const announcementCollection = "site_announcements";
const dismissedKeyPrefix = "site-announcement:dismissed:";

type AnnouncementRecord = {
  id: string;
  variant?: string | string[];
  message?: unknown;
  created?: string;
  published_at?: string;
  archived?: boolean;
};

const normalizeVariant = (value: unknown): "banner" | "modal" | null => {
  if (typeof value === "string") {
    const variant = value.trim();
    if (variant === "banner" || variant === "modal") return variant;
    return null;
  }
  if (Array.isArray(value) && value.length > 0) {
    const first = value[0];
    if (typeof first !== "string") return null;
    const variant = first.trim();
    if (variant === "banner" || variant === "modal") return variant;
    return null;
  }
  return null;
};

const normalizeAnnouncement = (
  record: AnnouncementRecord,
): SiteAnnouncement | null => {
  const id = (record?.id || "").trim();
  const variant = normalizeVariant(record?.variant);
  const message = normalizeMessage(record?.message);
  if (!id || !message) {
    return null;
  }
  if (!variant) {
    return null;
  }
  return {
    id,
    variant,
    message,
    created:
      typeof record.published_at === "string" && record.published_at.trim()
        ? record.published_at
        : typeof record.created === "string"
          ? record.created
          : "",
  };
};

const normalizeMessage = (value: unknown): string => {
  if (typeof value === "string") {
    return value.trim();
  }
  if (Array.isArray(value)) {
    for (const item of value) {
      const nested = normalizeMessage(item);
      if (nested) return nested;
    }
    return "";
  }
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    for (const key of ["markdown", "text", "value", "content", "html"]) {
      const field = record[key];
      if (typeof field === "string" && field.trim() !== "") {
        return field.trim();
      }
    }
    try {
      const serialized = JSON.stringify(value);
      return serialized === "{}" ? "" : serialized;
    } catch {
      return "";
    }
  }
  return "";
};

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
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const userId = useAuthStore((s) => s.userId);
  const helpOpen = useUIStore((s) => s.dialogs[UI_DIALOG.Help]);
  const [announcement, setAnnouncement] = useState<SiteAnnouncement | null>(
    null,
  );
  const [dismissedId, setDismissedId] = useState("");
  const [canOpenModal, setCanOpenModal] = useState(true);

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

  useEffect(() => {
    if (!isAuthenticated) {
      setAnnouncement(null);
      return;
    }
    let active = true;

    const fetchLatest = async () => {
      try {
        const record = (await pb
          .collection(announcementCollection)
          .getFirstListItem("archived = false", {
            sort: "-published_at",
          })) as unknown as AnnouncementRecord;
        if (!active) {
          return;
        }
        setAnnouncement(normalizeAnnouncement(record));
      } catch (error) {
        setAnnouncement(null);
        console.warn("[announcements] fetch failed", error);
      }
    };

    void fetchLatest();
    void pb
      .collection(announcementCollection)
      .subscribe("*", () => {
        void fetchLatest();
      })
      .catch((error) => {
        console.warn("[announcements] subscribe failed", error);
      });

    return () => {
      active = false;
      void pb.collection(announcementCollection).unsubscribe("*");
    };
  }, [isAuthenticated]);

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
