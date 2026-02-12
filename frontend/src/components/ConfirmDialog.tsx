import { useUIStore } from "@/app/store/uiStore";
import useModal from "@/app/hooks/useModal";
import { defineUIDialogModal, UI_DIALOG } from "@/app/store/uiStore";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";

function ConfirmDialogBody() {
  const { close } = useModalBody();
  const confirmBody = useUIStore((s) => s.confirmBody);
  const confirmAction = useUIStore((s) => s.confirmAction);
  const confirmLabel = useUIStore((s) => s.confirmConfirmLabel) || "Confirm";
  const cancelLabel = useUIStore((s) => s.confirmCancelLabel) || "Cancel";
  const tone = useUIStore((s) => s.confirmTone) || "danger";
  const clearConfirm = useUIStore((s) => s.clearConfirm);
  const confirmButtonClass =
    tone === "danger"
      ? "btn btn-sm btn-error btn-outline"
      : "btn btn-sm btn-info btn-outline";

  return (
    <>
      <p>{confirmBody}</p>
      <div className="modal-action">
        <button className="btn btn-sm btn-outline" onClick={() => close()}>
          {cancelLabel}
        </button>
        <button
          className={confirmButtonClass}
          onClick={() => {
            const action = confirmAction;
            clearConfirm();
            close();
            if (action) action();
          }}
        >
          {confirmLabel}
        </button>
      </div>
    </>
  );
}

export const AppModalConfirm = defineUIDialogModal({
  key: UI_DIALOG.Confirm,
  useOpen: () => useUIStore((s) => s.dialogs[UI_DIALOG.Confirm]),
  build: () => ({
    title: useUIStore.getState().confirmTitle || "Confirm",
    body: <ConfirmDialogBody />,
  }),
});

export default function ConfirmDialog() {
  const clearConfirm = useUIStore((s) => s.clearConfirm);
  useModal(AppModalConfirm, {
    onDismiss: clearConfirm,
  });

  return null;
}
