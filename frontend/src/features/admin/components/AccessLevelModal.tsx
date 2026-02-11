import { useEffect, useState } from "react";
import { useAdminStore } from "../store/adminStore";
import { useAdminActionsStore } from "../store/adminActionsStore";
import SelectionDropdown from "@/components/SelectionDropdown";

type AccessLevel = "user" | "staff" | "admin";

const getAccessLevel = (level?: string): AccessLevel => {
  if (level === "staff" || level === "admin") return level;
  return "user";
};

export default function AccessLevelModal() {
  const open = useAdminStore((s) => s.modals.access);
  const user = useAdminStore((s) => s.selectedUser);
  const setModal = useAdminStore((s) => s.setModal);
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
    if (!open) return;
    setNextLevel(accessLevel === "staff" ? "staff" : "user");
  }, [accessLevel, open]);

  if (!open || !user) return null;

  return (
    <div className="modal modal-open">
      <div className="modal-box bg-base-200 border border-slate-700 max-w-md">
        <h3 className="font-display text-lg mb-3">Change access</h3>
        <div className="space-y-3 text-sm text-slate-300">
          <p className="text-xs text-slate-400">
            Access level is limited to user or staff. Admin access must be
            granted through the PocketBase superuser console.
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
        </div>
        <div className="modal-action">
          <button
            className="btn btn-xs btn-outline"
            onClick={() => setModal("access", false)}
          >
            Cancel
          </button>
          <button
            className="btn btn-xs btn-outline"
            onClick={() => void updateAccessLevel(nextLevel)}
          >
            Update access
          </button>
        </div>
        <button
          className="btn btn-outline btn-sm btn-square absolute right-3 top-3"
          onClick={() => setModal("access", false)}
        >
          ✕
        </button>
      </div>
    </div>
  );
}
