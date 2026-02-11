import { create } from "zustand";
import { api } from "@/config/api";
import { useUIStore } from "@/app/store/uiStore";
import { useAdminStore } from "./adminStore";
import type { Character, SearchResult } from "../types";
import { hasSearchMain } from "../utils/formatters";

type AccessLevel = "user" | "staff";

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
    requestConfirm(
      "Revoke Sessions",
      "Revoke all active sessions for this user?",
      async () => {
        try {
          await api.post(
            `/admin/users/${selectedUser.user_id}/revoke-sessions`,
          );
          await loadUser(selectedUser.user_id);
          setToast({ text: "Sessions revoked", color: "info" });
        } catch (error: any) {
          setToast({
            text: error?.response?.data || "Failed to revoke sessions",
            color: "error",
          });
        }
      },
    );
  },
  revokeUploadTokens: () => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    if (!selectedUser) return;
    requestConfirm(
      "Revoke Uploader Tokens",
      "Revoke all uploader tokens for this user?",
      async () => {
        try {
          await api.post(
            `/admin/users/${selectedUser.user_id}/revoke-upload-tokens`,
          );
          await loadUser(selectedUser.user_id);
          setToast({ text: "Uploader tokens revoked", color: "info" });
        } catch (error: any) {
          setToast({
            text: error?.response?.data || "Failed to revoke uploader tokens",
            color: "error",
          });
        }
      },
    );
  },
  regenerateUploadToken: () => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    if (!selectedUser) return;
    requestConfirm(
      "Regenerate Uploader Token",
      "Generate a new uploader token for this user?",
      async () => {
        try {
          await api.post(
            `/admin/users/${selectedUser.user_id}/regenerate-upload-token`,
          );
          await loadUser(selectedUser.user_id);
          setToast({ text: "Uploader token regenerated", color: "info" });
        } catch (error: any) {
          setToast({
            text:
              error?.response?.data || "Failed to regenerate uploader token",
            color: "error",
          });
        }
      },
    );
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
      setModal("access", false);
    } catch (error: any) {
      setToast({
        text: error?.response?.data || "Failed to update access level",
        color: "error",
      });
    }
  },
  setMain: (character) => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    if (!selectedUser) return;
    requestConfirm(
      "Set Main Character",
      `Set ${character.name} as main?`,
      async () => {
        try {
          await api.post(`/admin/users/${selectedUser.user_id}/main`, {
            character_record_id: character.id,
          });
          await loadUser(selectedUser.user_id);
          setToast({ text: "Main character updated", color: "info" });
        } catch (error: any) {
          setToast({
            text: error?.response?.data || "Failed to set main character",
            color: "error",
          });
        }
      },
    );
  },
  refreshCharacter: async (character) => {
    const { setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    try {
      await api.post(`/admin/characters/${character.id}/refresh`);
      setToast({
        text: `Refreshed ${character.name}`,
        color: "info",
      });
      if (selectedUser) {
        await loadUser(selectedUser.user_id);
      }
    } catch (error: any) {
      setToast({
        text: error?.response?.data || "Failed to refresh character",
        color: "error",
      });
    }
  },
  revokeCharacter: (character) => {
    const { requestConfirm, setToast } = useUIStore.getState();
    const { selectedUser, loadUser } = useAdminStore.getState();
    requestConfirm(
      "Revoke Character Keys",
      `Revoke ESI keys for ${character.name}?`,
      async () => {
        try {
          await api.post(`/admin/characters/${character.id}/revoke`);
          if (selectedUser) {
            await loadUser(selectedUser.user_id);
          }
          setToast({ text: "Character keys revoked", color: "info" });
        } catch (error: any) {
          setToast({
            text: error?.response?.data || "Failed to revoke character keys",
            color: "error",
          });
        }
      },
    );
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
    requestConfirm("Remove Character", summary, async () => {
      try {
        await api.delete(`/admin/characters/${character.id}`);
        if (selectedUser) {
          await loadUser(selectedUser.user_id);
        }
        setToast({ text: "Character removed", color: "info" });
      } catch (error: any) {
        setToast({
          text: error?.response?.data || "Failed to remove character",
          color: "error",
        });
      }
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

    requestConfirm("Merge Accounts", summary, async () => {
      try {
        await api.post(`/admin/users/${selectedUser.user_id}/merge`, {
          target_user_id: target.user_id,
        });
        clearUser();
        setModal("merge", false);
        setToast({ text: "Accounts merged", color: "info" });
      } catch (error: any) {
        setToast({
          text: error?.response?.data || "Failed to merge account",
          color: "error",
        });
      }
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
    requestConfirm("Move Character", summary, async () => {
      try {
        await api.post(`/admin/characters/${character.id}/move`, {
          target_user_id: target.user_id,
        });
        await loadUser(selectedUser.user_id);
        setToast({ text: "Character moved", color: "info" });
        setModal("move", false);
      } catch (error: any) {
        setToast({
          text: error?.response?.data || "Failed to move character",
          color: "error",
        });
      }
    });
  },
}));
