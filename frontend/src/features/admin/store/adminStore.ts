import { create } from "zustand";
import { api } from "@/config/api";
import type { UserDetails } from "../types";

type AdminModalKey = "merge" | "move" | "audit" | "access";

type AdminState = {
  selectedUser: UserDetails | null;
  modals: Record<AdminModalKey, boolean>;
  setSelectedUser: (user: UserDetails | null) => void;
  clearUser: () => void;
  loadUser: (userId: string) => Promise<UserDetails>;
  setModal: (modal: AdminModalKey, open: boolean) => void;
};

export const useAdminStore = create<AdminState>((set) => ({
  selectedUser: null,
  modals: {
    merge: false,
    move: false,
    audit: false,
    access: false,
  },
  setSelectedUser: (user) => set({ selectedUser: user }),
  clearUser: () =>
    set({
      selectedUser: null,
      modals: {
        merge: false,
        move: false,
        audit: false,
        access: false,
      },
    }),
  loadUser: async (userId) => {
    const res = await api.get(`/admin/users/${userId}`);
    const user = res.data as UserDetails;
    set({ selectedUser: user });
    return user;
  },
  setModal: (modal, open) =>
    set((state) => ({
      modals: {
        ...state.modals,
        [modal]: open,
      },
    })),
}));
