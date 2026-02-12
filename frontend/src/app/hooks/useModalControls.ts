import { useShallow } from "zustand/shallow";
import { useUIStore } from "@/app/store/uiStore";

export default function useModalControls() {
  const { openModal, closeModal } = useUIStore(
    useShallow((s) => ({
      openModal: s.openModal,
      closeModal: s.closeModal,
    })),
  );

  return {
    openModal,
    closeModal,
  };
}
