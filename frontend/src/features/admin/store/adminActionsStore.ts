import { create } from "zustand";
import { api } from "@/config/api";
import { useUIStore } from "@/app/store/uiStore";
import { getErrorMessage } from "@/utils/httpError";
import { ADMIN_MODAL, useAdminStore } from "./adminStore";
import type { Character, SearchResult } from "../types";
import { hasSearchMain } from "../utils/formatters";

type AccessLevel = "user" | "staff";
type AnnouncementVariant = "banner" | "modal";

type AdminActionsState = {
  refreshAll: () => Promise<void>;
  revokeSessions: () => void;
  revokeUploadTokens: () => void;
  regenerateUploadToken: () => void;
  updateAccessLevel: (level: AccessLevel) => Promise<void>;
  setMain: (character: Character) => void;
  refreshCharacter: (character: Character) => Promise<void>;
  revokeCharacter: (character: Character) => void;
  removeCharacter: (character: Character) => void;
  mergeUser: (target: SearchResult | null) => Promise<void>;
  moveCharacter: (
    characterId: string,
    target: SearchResult | null,
  ) => Promise<void>;
  publishAnnouncement: (
    variant: AnnouncementVariant,
    message: string,
  ) => Promise<void>;
  archiveLatestAnnouncement: () => Promise<void>;
};

export const useAdminActionsStore = create<AdminActionsState>(() => ({
  refreshAll: async () => {
    const { setToast } = useUIStore.getState();
    const { selectedUser } = useAdminStore.getState();
    if (!selectedUser) {
      setToast({ text: "Select a user first", color: "error" });
      return;
    }
    try {
      const res = await api.post("/admin/characters/refresh-all", {
        user_id: selectedUser.user_id,
      });
      const jobId = res.data?.job_id ?? "unknown";
      setToast({
        text: `Refresh started (job ${jobId})`,
        color: "info",
      });
    } catch {
      setToast({ text: "Failed to refresh characters", color: "error" });
    }
  },
  revokeSessions: () => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    if (!selectedUser) return;
    requestConfirm({
      title: "Revoke Sessions",
      body: "Revoke all active sessions for this user?",
      onConfirm: async () => {
        try {
          await api.post(
            `/admin/users/${selectedUser.user_id}/revoke-sessions`,
          );
          await loadUser(selectedUser.user_id);
          setToast({ text: "Sessions revoked", color: "info" });
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to revoke sessions"),
            color: "error",
          });
        }
      },
      confirmLabel: "Revoke",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  },
  revokeUploadTokens: () => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    if (!selectedUser) return;
    requestConfirm({
      title: "Revoke Uploader Tokens",
      body: "Revoke all uploader tokens for this user?",
      onConfirm: async () => {
        try {
          await api.post(
            `/admin/users/${selectedUser.user_id}/revoke-upload-tokens`,
          );
          await loadUser(selectedUser.user_id);
          setToast({ text: "Uploader tokens revoked", color: "info" });
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to revoke uploader tokens"),
            color: "error",
          });
        }
      },
      confirmLabel: "Revoke",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  },
  regenerateUploadToken: () => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    if (!selectedUser) return;
    requestConfirm({
      title: "Regenerate Uploader Token",
      body: "Generate a new uploader token for this user?",
      onConfirm: async () => {
        try {
          await api.post(
            `/admin/users/${selectedUser.user_id}/regenerate-upload-token`,
          );
          await loadUser(selectedUser.user_id);
          setToast({ text: "Uploader token regenerated", color: "info" });
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to regenerate uploader token"),
            color: "error",
          });
        }
      },
      confirmLabel: "Regenerate",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  },
  updateAccessLevel: async (level) => {
    const { setToast } = useUIStore.getState();
    const { selectedUser, loadUser, setModal } = useAdminStore.getState();
    if (!selectedUser) return;
    try {
      await api.post(`/admin/users/${selectedUser.user_id}/access-level`, {
        access_level: level,
      });
      await loadUser(selectedUser.user_id);
      setToast({ text: "Access level updated", color: "info" });
      setModal(ADMIN_MODAL.Access, false);
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to update access level"),
        color: "error",
      });
    }
  },
  setMain: (character) => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    if (!selectedUser) return;
    requestConfirm({
      title: "Set Main Character",
      body: `Set ${character.name} as main?`,
      onConfirm: async () => {
        try {
          await api.post(`/admin/users/${selectedUser.user_id}/main`, {
            character_record_id: character.id,
          });
          await loadUser(selectedUser.user_id);
          setToast({ text: "Main character updated", color: "info" });
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to set main character"),
            color: "error",
          });
        }
      },
      confirmLabel: "Set main",
      cancelLabel: "Cancel",
      tone: "default",
    });
  },
  refreshCharacter: async (character) => {
    const { setToast } = useUIStore.getState();
    try {
      const res = await api.post(`/admin/characters/${character.id}/refresh`);
      const jobId = res.data?.job_id ?? "unknown";
      setToast({
        text: `Refresh started for ${character.name} (job ${jobId})`,
        color: "info",
      });
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to refresh character"),
        color: "error",
      });
    }
  },
  revokeCharacter: (character) => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    requestConfirm({
      title: "Revoke Character Keys",
      body: `Revoke ESI keys for ${character.name}?`,
      onConfirm: async () => {
        try {
          await api.post(`/admin/characters/${character.id}/revoke`);
          if (selectedUser) {
            await loadUser(selectedUser.user_id);
          }
          setToast({ text: "Character keys revoked", color: "info" });
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to revoke character keys"),
            color: "error",
          });
        }
      },
      confirmLabel: "Revoke",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  },
  removeCharacter: (character) => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    const count = selectedUser?.characters.length || 0;
    if (character.is_main && count > 1) {
      setToast({
        text: "Main character cannot be removed while others exist",
        color: "error",
      });
      return;
    }
    const summary = `Remove ${character.name}? ${
      character.is_main ? "This will delete the account." : ""
    }`;
    requestConfirm({
      title: "Remove Character",
      body: summary,
      onConfirm: async () => {
        try {
          await api.delete(`/admin/characters/${character.id}`);
          if (selectedUser) {
            await loadUser(selectedUser.user_id);
          }
          setToast({ text: "Character removed", color: "info" });
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to remove character"),
            color: "error",
          });
        }
      },
      confirmLabel: "Remove",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  },
  mergeUser: async (target) => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, clearUser, setModal } = useAdminStore.getState();
    if (!selectedUser || !target) return;
    if (selectedUser.user_id === target.user_id) {
      setToast({ text: "Target user must be different", color: "error" });
      return;
    }
    if (!hasSearchMain(target)) {
      setToast({
        text: "Target user has no main character. Merge blocked.",
        color: "error",
      });
      return;
    }

    const sourceMain =
      selectedUser.characters.find((character) => character.is_main) || null;
    const targetMain = target.main_name || (target.is_main ? target.name : "");
    if (!targetMain) {
      setToast({
        text: "Target user has no main character. Merge blocked.",
        color: "error",
      });
      return;
    }

    const count = selectedUser.characters.length;
    const characterNames = selectedUser.characters.map(
      (character) => character.name,
    );
    const preview =
      characterNames.length > 3
        ? `${characterNames.slice(0, 3).join(", ")} +${characterNames.length - 3} more`
        : characterNames.join(", ");
    const summary = `Merge ${count} character${count === 1 ? "" : "s"} from ${
      sourceMain?.name || "source account"
    } into ${targetMain}? Characters: ${preview}. Source account will be deleted if empty.`;

    requestConfirm({
      title: "Merge Accounts",
      body: summary,
      onConfirm: async () => {
        try {
          await api.post(`/admin/users/${selectedUser.user_id}/merge`, {
            target_user_id: target.user_id,
          });
          clearUser();
          setModal(ADMIN_MODAL.Merge, false);
          setToast({ text: "Accounts merged", color: "info" });
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to merge account"),
            color: "error",
          });
        }
      },
      confirmLabel: "Merge",
      cancelLabel: "Cancel",
      tone: "danger",
    });
  },
  moveCharacter: async (characterId, target) => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser, setModal } = useAdminStore.getState();
    if (!selectedUser || !characterId || !target) return;
    if (selectedUser.user_id === target.user_id) {
      setToast({ text: "Target user must be different", color: "error" });
      return;
    }
    if (!hasSearchMain(target)) {
      setToast({
        text: "Target user has no main character. Move blocked.",
        color: "error",
      });
      return;
    }

    const character = selectedUser.characters.find(
      (candidate) => candidate.id === characterId,
    );
    if (!character) {
      setToast({ text: "Select a character to move", color: "error" });
      return;
    }
    const targetMain = target.main_name || (target.is_main ? target.name : "");
    if (!targetMain) {
      setToast({
        text: "Target user has no main character. Move blocked.",
        color: "error",
      });
      return;
    }

    const summary = `Move ${character.name} to ${targetMain}? It will be demoted to non-main.`;
    requestConfirm({
      title: "Move Character",
      body: summary,
      onConfirm: async () => {
        try {
          await api.post(`/admin/characters/${character.id}/move`, {
            target_user_id: target.user_id,
          });
          await loadUser(selectedUser.user_id);
          setToast({ text: "Character moved", color: "info" });
          setModal(ADMIN_MODAL.Move, false);
        } catch (error: unknown) {
          setToast({
            text: getErrorMessage(error, "Failed to move character"),
            color: "error",
          });
        }
      },
      confirmLabel: "Move",
      cancelLabel: "Cancel",
      tone: "default",
    });
  },
  publishAnnouncement: async (variant, message) => {
    const { setToast } = useUIStore.getState();
    const { setModal } = useAdminStore.getState();
    const trimmed = message.trim();
    if (!trimmed) {
      setToast({ text: "Announcement message is required", color: "error" });
      return;
    }
    try {
      await api.post("/admin/announcement", {
        variant,
        message: trimmed,
      });
      setToast({ text: "Announcement published", color: "info" });
      setModal(ADMIN_MODAL.Announcement, false);
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to publish announcement"),
        color: "error",
      });
    }
  },
  archiveLatestAnnouncement: async () => {
    const { setToast } = useUIStore.getState();
    try {
      const res = await api.post("/admin/announcement/archive-latest");
      if (res.data?.archived) {
        setToast({ text: "Latest announcement archived", color: "info" });
      } else {
        setToast({ text: "No active announcement to archive", color: "info" });
      }
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to archive announcement"),
        color: "error",
      });
    }
  },
}));
