import { create } from "zustand";
import type { ReactNode } from "react";
import { createModalRegistry } from "@/app/store/modalRegistry";

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

export const UI_DIALOG = {
  Help: "help",
  ShareLink: "shareLink",
  PermissionRequired: "permissionRequired",
  AlarmStart: "alarmStart",
  Confirm: "confirm",
} as const;
export const UI_DIALOG_KEYS = [
  UI_DIALOG.Help,
  UI_DIALOG.ShareLink,
  UI_DIALOG.PermissionRequired,
  UI_DIALOG.AlarmStart,
  UI_DIALOG.Confirm,
] as const;
export type DialogStateKey = (typeof UI_DIALOG_KEYS)[number];
const uiDialogRegistry = createModalRegistry<DialogStateKey>(UI_DIALOG_KEYS);
type DialogState = Record<DialogStateKey, boolean>;

export type ModalConfig = {
  title?: string;
  body: ReactNode;
  actions?: ReactNode;
  sizeClass?: string;
  dismissible?: boolean;
  closeOnOverlay?: boolean;
  closeDisabled?: boolean;
  onClose?:
    | ((reason?: ModalCloseReason) => boolean | void | Promise<boolean | void>)
    | null;
};

export type ModalCloseReason = "button" | "escape" | "overlay" | "programmatic";
export type ConfirmTone = "default" | "danger";
export type ConfirmRequest = {
  title: string;
  body: string;
  onConfirm: () => void | Promise<void>;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: ConfirmTone;
};

type UIState = {
  contextMenu: ContextMenuState | null;
  toast: ToastState;
  dialogs: DialogState;
  confirmTitle?: string;
  confirmBody?: string;
  confirmAction?: (() => void | Promise<void>) | null;
  confirmConfirmLabel?: string;
  confirmCancelLabel?: string;
  confirmTone?: ConfirmTone;
  modal: {
    open: boolean;
    title?: string;
    body?: ReactNode;
    actions?: ReactNode;
    sizeClass?: string;
    dismissible?: boolean;
    closeOnOverlay?: boolean;
    closeDisabled?: boolean;
    onClose?:
      | ((
          reason?: ModalCloseReason,
        ) => boolean | void | Promise<boolean | void>)
      | null;
  };
  setContextMenu: (menu: ContextMenuState | null) => void;
  setToast: (toast: ToastState) => void;
  clearToast: () => void;
  setModal: (modal: DialogStateKey, open: boolean) => void;
  requestConfirm: (request: ConfirmRequest) => void;
  clearConfirm: () => void;
  openModal: (config: ModalConfig) => void;
  closeModal: () => void;
};

export const useUIStore = create<UIState>((set) => ({
  contextMenu: null,
  toast: null,
  dialogs: uiDialogRegistry.initial(),
  confirmTitle: undefined,
  confirmBody: undefined,
  confirmAction: null,
  confirmConfirmLabel: undefined,
  confirmCancelLabel: undefined,
  confirmTone: undefined,
  modal: {
    open: false,
    title: undefined,
    body: undefined,
    sizeClass: undefined,
    dismissible: true,
    closeOnOverlay: false,
    closeDisabled: false,
    onClose: null,
  },
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
  setModal: (modal, open) =>
    set((state) => ({
      dialogs: {
        ...state.dialogs,
        [modal]: open,
      },
    })),
  requestConfirm: (request) =>
    set({
      confirmTitle: request.title,
      confirmBody: request.body,
      confirmAction: request.onConfirm,
      confirmConfirmLabel: request.confirmLabel,
      confirmCancelLabel: request.cancelLabel,
      confirmTone: request.tone ?? "danger",
      dialogs: {
        ...uiDialogRegistry.initial(),
        [UI_DIALOG.Confirm]: true,
      },
    }),
  clearConfirm: () =>
    set((state) => ({
      confirmTitle: undefined,
      confirmBody: undefined,
      confirmAction: null,
      confirmConfirmLabel: undefined,
      confirmCancelLabel: undefined,
      confirmTone: undefined,
      dialogs: { ...state.dialogs, [UI_DIALOG.Confirm]: false },
    })),
  openModal: (config) =>
    set({
      modal: {
        open: true,
        title: config.title,
        body: config.body,
        actions: config.actions,
        sizeClass: config.sizeClass,
        dismissible: config.dismissible ?? true,
        closeOnOverlay: config.closeOnOverlay ?? false,
        closeDisabled: config.closeDisabled ?? false,
        onClose: config.onClose ?? null,
      },
    }),
  closeModal: () =>
    set({
      modal: {
        open: false,
        title: undefined,
        body: undefined,
        actions: undefined,
        sizeClass: undefined,
        dismissible: true,
        closeOnOverlay: false,
        closeDisabled: false,
        onClose: null,
      },
    }),
}));

export const defineUIDialogModal = uiDialogRegistry.defineForStore(useUIStore);
