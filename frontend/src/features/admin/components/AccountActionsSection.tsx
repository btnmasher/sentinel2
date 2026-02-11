import CharacterCard from "@/components/CharacterCard";
import { ArrowLeft } from "lucide-react";
import { useAdminStore } from "../store/adminStore";
import { useAdminActionsStore } from "../store/adminActionsStore";
import { formatSessionStatus } from "../utils/formatters";
import type { Character } from "../types";

type AccessLevel = "user" | "staff" | "admin";

const getAccessLevel = (level?: string): AccessLevel => {
  if (level === "staff" || level === "admin") return level;
  return "user";
};

const getAccessLevelClass = (level: AccessLevel): string => {
  switch (level) {
    case "admin":
      return "role-admin";
    case "staff":
      return "role-staff";
    default:
      return "role-user";
  }
};

export default function AccountActionsSection({
  onBack,
}: {
  onBack?: () => void;
}) {
  const user = useAdminStore((s) => s.selectedUser);
  const setModal = useAdminStore((s) => s.setModal);
  const refreshAll = useAdminActionsStore((s) => s.refreshAll);
  const revokeSessions = useAdminActionsStore((s) => s.revokeSessions);
  const revokeUploadTokens = useAdminActionsStore((s) => s.revokeUploadTokens);
  const regenerateUploadToken = useAdminActionsStore(
    (s) => s.regenerateUploadToken,
  );
  const setMain = useAdminActionsStore((s) => s.setMain);
  const refreshCharacter = useAdminActionsStore((s) => s.refreshCharacter);
  const revokeCharacter = useAdminActionsStore((s) => s.revokeCharacter);
  const removeCharacter = useAdminActionsStore((s) => s.removeCharacter);

  const accessLevel = getAccessLevel(user?.access_level);
  const changeAccessDisabled = accessLevel === "admin";
  const uploaderTokenValid = user?.uploader_token_valid ?? false;

  const handleSetMain = (character: Character) => setMain(character);
  const handleRefreshCharacter = (character: Character) =>
    void refreshCharacter(character);
  const handleRevokeCharacter = (character: Character) =>
    revokeCharacter(character);
  const handleRemoveCharacter = (character: Character) =>
    removeCharacter(character);

  return (
    <section className="card bg-base-200/70 border border-slate-800 h-full min-h-0">
      <div className="card-body flex flex-col gap-4 h-full min-h-0 overflow-hidden">
        <div className="flex items-center justify-between gap-2">
          <h2 className="font-display text-2xl">Account Actions</h2>
          {user && onBack && (
            <button className="btn btn-xs btn-ghost gap-1" onClick={onBack}>
              <ArrowLeft className="h-3.5 w-3.5" />
              Back to search
            </button>
          )}
        </div>
        {!user ? (
          <p className="text-sm text-slate-400">Select a user to inspect.</p>
        ) : (
          <>
            <div className="rounded-xl border border-slate-800/70 bg-base-300/50 px-3 py-2">
              <p className="text-xs text-slate-400">User</p>
              <p className="text-sm font-semibold">{user.user_id}</p>
              <p className="text-xs text-slate-400">
                Characters: {user.characters.length} · Session:{" "}
                {formatSessionStatus(user.session_revoked_at)}
              </p>
              <p className="text-xs text-slate-400">
                Access:{" "}
                <span className={getAccessLevelClass(accessLevel)}>
                  {accessLevel}
                </span>
              </p>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto pr-1 space-y-3">
              {user.characters.map((character) => (
                <CharacterCard
                  key={character.id}
                  character={character}
                  onSetMain={() => handleSetMain(character)}
                  onRefresh={() => void handleRefreshCharacter(character)}
                  onRevoke={() => handleRevokeCharacter(character)}
                  onRemove={() => handleRemoveCharacter(character)}
                  disableRemove={
                    character.is_main && user.characters.length > 1
                  }
                />
              ))}
            </div>

            <div className="mt-auto rounded-xl border border-slate-800/70 bg-base-300/60 px-3 py-3">
              <div className="flex flex-wrap items-center gap-2">
                <button
                  className="btn btn-xs btn-success btn-outline"
                  onClick={() => void refreshAll()}
                  title="Auto refresh runs every 15 minutes."
                >
                  Refresh all characters
                </button>
                <button
                  className="btn btn-xs btn-warning btn-outline"
                  onClick={revokeSessions}
                >
                  Revoke sessions
                </button>
                <button
                  className="btn btn-xs btn-warning btn-outline"
                  onClick={revokeUploadTokens}
                  disabled={!uploaderTokenValid}
                  title={
                    uploaderTokenValid
                      ? "Revoke uploader tokens"
                      : "No active uploader token to revoke"
                  }
                >
                  Revoke uploader tokens
                </button>
                <button
                  className="btn btn-xs btn-info btn-outline"
                  onClick={regenerateUploadToken}
                >
                  Regenerate uploader token
                </button>
                <span
                  className="inline-flex"
                  title={
                    changeAccessDisabled ? "Admins cannot be demoted here." : ""
                  }
                >
                  <button
                    className="btn btn-xs btn-outline"
                    onClick={() => setModal("access", true)}
                    disabled={changeAccessDisabled}
                  >
                    Change access
                  </button>
                </span>
                <button
                  className="btn btn-xs btn-warning btn-outline"
                  onClick={() => setModal("move", true)}
                >
                  Move character
                </button>
                <button
                  className="btn btn-xs btn-warning btn-outline"
                  onClick={() => setModal("merge", true)}
                >
                  Merge account
                </button>
                <button
                  className="btn btn-xs btn-outline"
                  onClick={() => setModal("audit", true)}
                >
                  Activity log
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </section>
  );
}
