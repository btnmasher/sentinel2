import { useEffect, useState } from "react";
import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import {
  ADMIN_MODAL,
  defineAdminModal,
  useAdminStore,
} from "../store/adminStore";
import { useAdminActionsStore } from "../store/adminActionsStore";
import SelectionDropdown from "@/components/SelectionDropdown";

type AccessLevel = "user" | "staff" | "admin";

const getAccessLevel = (level?: string): AccessLevel => {
  if (level === "staff" || level === "admin") return level;
  return "user";
};

function AccessLevelModalBody() {
  const { close } = useModalBody();
  const user = useAdminStore((s) => s.selectedUser);
  const updateAccessLevel = useAdminActionsStore((s) => s.updateAccessLevel);
  const accessLevel = getAccessLevel(user?.access_level);
  const [nextLevel, setNextLevel] = useState<"user" | "staff">(
    accessLevel === "staff" ? "staff" : "user",
  );

  const accessOptions = [
    { id: "user", label: "User" },
    { id: "staff", label: "Staff" },
  ];

  useEffect(() => {
    setNextLevel(accessLevel === "staff" ? "staff" : "user");
  }, [accessLevel]);

  if (!user) return null;

  return (
    <div className="space-y-3 text-sm text-slate-300">
      <p className="text-xs text-slate-400">
        Access level is limited to user or staff. Admin access must be granted
        through the PocketBase superuser console.
      </p>
      <div className="space-y-2">
        <label className="text-xs text-slate-400">Access level</label>
        <SelectionDropdown
          items={accessOptions}
          selected={[nextLevel]}
          onChange={(next) =>
            setNextLevel((next[0] ?? "user") as "user" | "staff")
          }
          label="Access level"
          buttonClassName="w-full"
        />
      </div>
      <div className="modal-action">
        <button className="btn btn-xs btn-outline" onClick={() => close()}>
          Cancel
        </button>
        <button
          className="btn btn-xs btn-outline"
          onClick={() => void updateAccessLevel(nextLevel)}
        >
          Update access
        </button>
      </div>
    </div>
  );
}

export const AdminModalAccess = defineAdminModal({
  key: ADMIN_MODAL.Access,
  useOpen: () => {
    const open = useAdminStore((s) => s.modals[ADMIN_MODAL.Access]);
    const user = useAdminStore((s) => s.selectedUser);
    return open && Boolean(user);
  },
  build: () => ({
    title: "Change access",
    sizeClass: "max-w-md",
    body: <AccessLevelModalBody />,
  }),
});

export default function AccessLevelModal() {
  useModal(AdminModalAccess);

  return null;
}
