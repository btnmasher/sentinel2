import { create } from "zustand";
import { pb } from "@/config/pb";
import type { SiteAnnouncement } from "../types";

const announcementCollection = "site_announcements";

type AnnouncementRecord = {
  id: string;
  variant?: string | string[];
  message?: unknown;
  created?: string;
  published_at?: string;
  archived?: boolean;
};

type AnnouncementsState = {
  announcement: SiteAnnouncement | null;
  realtimeUnsubscribe?: () => Promise<void>;
  loadLatest: () => Promise<void>;
  startRealtime: () => Promise<void>;
  stopRealtime: () => Promise<void>;
  clear: () => void;
};

export const useAnnouncementsStore = create<AnnouncementsState>((set, get) => ({
  announcement: null,
  realtimeUnsubscribe: undefined,
  loadLatest: async () => {
    try {
      const record = (await pb
        .collection(announcementCollection)
        .getFirstListItem("archived = false", {
          sort: "-published_at",
        })) as unknown as AnnouncementRecord;
      set({ announcement: normalizeAnnouncement(record) });
    } catch (error) {
      set({ announcement: null });
      console.warn("[announcements] fetch failed", error);
    }
  },
  startRealtime: async () => {
    if (get().realtimeUnsubscribe) {
      return;
    }
    try {
      const unsubscribeRaw = await pb
        .collection(announcementCollection)
        .subscribe("*", () => {
          void get().loadLatest();
        });
      set({ realtimeUnsubscribe: unsubscribeRaw });
    } catch (error) {
      console.warn("[announcements] subscribe failed", error);
    }
  },
  stopRealtime: async () => {
    const unsubscribe = get().realtimeUnsubscribe;
    set({ realtimeUnsubscribe: undefined });
    if (unsubscribe) {
      await unsubscribe().catch(() => undefined);
    }
  },
  clear: () => set({ announcement: null }),
}));

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

const normalizeAnnouncement = (
  record: AnnouncementRecord,
): SiteAnnouncement | null => {
  const id = (record?.id || "").trim();
  const variant = normalizeVariant(record?.variant);
  const message = normalizeMessage(record?.message);
  if (!id || !message || !variant) {
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
