import Modal from "./Modal";
import { useUIStore } from "@/app/store/uiStore";

export default function PermissionRequiredDialog() {
  const dialogs = useUIStore((s) => s.dialogs);
  const setDialog = useUIStore((s) => s.setDialog);

  return (
    <Modal
      open={dialogs.permissionRequired}
      title="New ESI Permissions Required"
      onClose={() => setDialog("permissionRequired", false)}
      actions={
        <button
          className="btn btn-sm btn-outline"
          onClick={() => setDialog("permissionRequired", false)}
        >
          Close
        </button>
      }
    >
      <p>
        Set route allows you to set the route destination in game. However,
        this feature requires new ESI permissions for the character you
        selected.
      </p>
      <p>
        If you have already set this, you may be seeing this message by mistake.
      </p>
      <p>
        To set the new permissions, re-add your ESI key (do not delete it, it
        will be replaced) on TEST Auth.
      </p>
      <p className="text-sky-300">
        https://forum.pleaseignore.com/topic/107471-esi-scopes-and-you-for-the-privacy-and-security-oriented/
      </p>
    </Modal>
  );
}
