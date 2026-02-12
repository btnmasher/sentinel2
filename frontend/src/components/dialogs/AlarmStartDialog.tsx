import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import {
  defineUIDialogModal,
  UI_DIALOG,
  useUIStore,
} from "@/app/store/uiStore";

function AlarmStartDialogBody() {
  const { close } = useModalBody();
  return (
    <>
      <p>
        Unless autoplay is allowed for Sentinel, the intel alarm requires user
        input before it can play.
      </p>
      <p>
        You can enable sound (Chrome) / autoplay (Firefox) in the info menu on
        the left of the address bar.
      </p>
      <div className="modal-action">
        <button className="btn btn-sm btn-outline" onClick={() => close()}>
          Close
        </button>
      </div>
    </>
  );
}

export const AppModalAlarmStart = defineUIDialogModal({
  key: UI_DIALOG.AlarmStart,
  useOpen: () => useUIStore((s) => s.dialogs[UI_DIALOG.AlarmStart]),
  build: () => ({
    title: "Autoplay Disabled: Click anywhere",
    body: <AlarmStartDialogBody />,
  }),
});

export default function AlarmStartDialog() {
  useModal(AppModalAlarmStart);

  return null;
}
