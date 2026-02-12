import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import {
  defineUIDialogModal,
  UI_DIALOG,
  useUIStore,
} from "@/app/store/uiStore";

function PermissionRequiredBody() {
  const { close } = useModalBody();
  return (
    <>
      <p>
        Set route allows you to set the route destination in game. However, this
        feature requires new ESI permissions for the character you selected.
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
      <div className="modal-action">
        <button className="btn btn-sm btn-outline" onClick={() => close()}>
          Close
        </button>
      </div>
    </>
  );
}

export const AppModalPermissionRequired = defineUIDialogModal({
  key: UI_DIALOG.PermissionRequired,
  useOpen: () => useUIStore((s) => s.dialogs[UI_DIALOG.PermissionRequired]),
  build: () => ({
    title: "New ESI Permissions Required",
    body: <PermissionRequiredBody />,
  }),
});

export default function PermissionRequiredDialog() {
  useModal(AppModalPermissionRequired);

  return null;
}
