import { useShallow } from "zustand/shallow";
import Modal from "@/components/dialogs/Modal";
import { ModalBodyProvider } from "@/components/dialogs/ModalBodyContext";
import { useUIStore } from "@/app/store/uiStore";
import type { ModalCloseReason } from "@/app/store/uiStore";

export default function GlobalModalHost() {
  const { modal, closeModal } = useUIStore(
    useShallow((s) => ({
      modal: s.modal,
      closeModal: s.closeModal,
    })),
  );

  const handleClose = (reason: ModalCloseReason = "programmatic") => {
    if (modal.onClose) {
      void Promise.resolve(modal.onClose(reason)).then((shouldClose) => {
        if (shouldClose === false) return;
        closeModal();
      });
      return;
    }
    closeModal();
  };

  return (
    <ModalBodyProvider close={handleClose}>
      <Modal
        open={modal.open}
        title={modal.title}
        onClose={handleClose}
        actions={modal.actions}
        className={modal.sizeClass}
        dismissible={modal.dismissible}
        closeOnOverlay={modal.closeOnOverlay}
        closeDisabled={modal.closeDisabled}
      >
        {modal.body}
      </Modal>
    </ModalBodyProvider>
  );
}
