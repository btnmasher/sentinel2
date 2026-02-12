import { create } from "zustand";
import { api } from "@/config/api";
import type { UserDetails } from "../types";
import { createModalRegistry } from "@/app/store/modalRegistry";

export const ADMIN_MODAL = {
  Merge: "merge",
  Move: "move",
  Audit: "audit",
  Access: "access",
} as const;

export const ADMIN_MODAL_KEYS = [
  ADMIN_MODAL.Merge,
  ADMIN_MODAL.Move,
  ADMIN_MODAL.Audit,
  ADMIN_MODAL.Access,
] as const;
export type AdminModalKey = (typeof ADMIN_MODAL_KEYS)[number];
export const adminModalRegistry =
  createModalRegistry<AdminModalKey>(ADMIN_MODAL_KEYS);

type AdminState = {
  selectedUser: UserDetails | null;
  modals: Record<AdminModalKey, boolean>;
  setSelectedUser: (user: UserDetails | null) => void;
  clearUser: () => void;
  loadUser: (userId: string) => Promise<UserDetails>;
  setModal: (modal: AdminModalKey, open: boolean) => void;
  openModalKey: (modal: AdminModalKey) => void;
  closeModalKey: (modal: AdminModalKey) => void;
  resetModals: () => void;
};

export const useAdminStore = create<AdminState>((set) => ({
  selectedUser: null,
  modals: adminModalRegistry.initial(),
  setSelectedUser: (user) => set({ selectedUser: user }),
  clearUser: () =>
    set({
      selectedUser: null,
      modals: adminModalRegistry.initial(),
    }),
  loadUser: async (userId) => {
    const res = await api.get(`/admin/users/${userId}`);
    const user = res.data as UserDetails;
    set({ selectedUser: user });
    return user;
  },
  ...adminModalRegistry.actions<AdminState>(set),
}));

export const defineAdminModal =
  adminModalRegistry.defineForStore(useAdminStore);
