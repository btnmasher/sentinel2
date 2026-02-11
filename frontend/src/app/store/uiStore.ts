import { create } from "zustand";

const safeStringify = (value: unknown) => {
  const seen = new WeakSet<object>();
  return JSON.stringify(
    value,
    (_key, val) => {
      if (val instanceof Error) {
        return {
          name: val.name,
          message: val.message,
          stack: val.stack,
        };
      }
      if (typeof val === "function") {
        return `[Function${val.name ? ` ${val.name}` : ""}]`;
      }
      if (typeof val === "object" && val !== null) {
        if (seen.has(val)) {
          return "[Circular]";
        }
        seen.add(val);
      }
      return val;
    },
    2,
  );
};

export type ContextMenuState = {
  x: number;
  y: number;
  anchorRect?: { left: number; top: number; width: number; height: number };
  routeMode?: "set" | "add";
  type:
    | "map"
    | "system"
    | "map-jumprange"
    | "system-jumprange"
    | "route-character"
    | "character-search"
    | "character"
    | "text"
    | null;
  systemId?: number;
  text?: string;
  character?: string;
  characterId?: number;
};

export type ToastState = {
  text: string;
  color?: "secondary" | "error" | "success" | "warning" | "info";
  timeout?: number;
  meta?: Record<string, unknown>;
} | null;

type DialogState = {
  help: boolean;
  shareLink: boolean;
  permissionRequired: boolean;
  alarmStart: boolean;
  confirm: boolean;
};

type UIState = {
  contextMenu: ContextMenuState | null;
  toast: ToastState;
  dialogs: DialogState;
  confirmTitle?: string;
  confirmBody?: string;
  confirmAction?: (() => void) | null;
  setContextMenu: (menu: ContextMenuState | null) => void;
  setToast: (toast: ToastState) => void;
  clearToast: () => void;
  setDialog: (dialog: keyof DialogState, show: boolean) => void;
  requestConfirm: (title: string, body: string, action: () => void) => void;
  clearConfirm: () => void;
};

export const useUIStore = create<UIState>((set) => ({
  contextMenu: null,
  toast: null,
  dialogs: {
    help: false,
    shareLink: false,
    permissionRequired: false,
    alarmStart: false,
    confirm: false,
  },
  confirmTitle: undefined,
  confirmBody: undefined,
  confirmAction: null,
  setContextMenu: (menu) => set({ contextMenu: menu }),
  setToast: (toast) =>
    set(() => {
      if (toast) {
        const payload = {
          timeout: 3000,
          color: "secondary" as const,
          ...toast,
        };
        if (payload.color === "error") {
          const metaText = payload.meta ? safeStringify(payload.meta) : "";
          console.error("[notification]", payload.text, metaText || undefined);
        }
        return { toast: payload };
      }
      return { toast: null };
    }),
  clearToast: () => set({ toast: null }),
  setDialog: (dialog, show) =>
    set((state) => ({
      dialogs: {
        ...state.dialogs,
        [dialog]: show,
      },
    })),
  requestConfirm: (title, body, action) =>
    set({
      confirmTitle: title,
      confirmBody: body,
      confirmAction: action,
      dialogs: {
        help: false,
        shareLink: false,
        permissionRequired: false,
        alarmStart: false,
        confirm: true,
      },
    }),
  clearConfirm: () =>
    set((state) => ({
      confirmTitle: undefined,
      confirmBody: undefined,
      confirmAction: null,
      dialogs: { ...state.dialogs, confirm: false },
    })),
}));
