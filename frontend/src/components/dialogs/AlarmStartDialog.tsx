import Modal from "./Modal";
import { useUIStore } from "@/app/store/uiStore";

export default function AlarmStartDialog() {
  const dialogs = useUIStore((s) => s.dialogs);
  const setDialog = useUIStore((s) => s.setDialog);

  return (
    <Modal
      open={dialogs.alarmStart}
      title="Autoplay Disabled: Click anywhere"
      onClose={() => setDialog("alarmStart", false)}
      actions={
        <button
          className="btn btn-sm btn-outline"
          onClick={() => setDialog("alarmStart", false)}
        >
          Close
        </button>
      }
    >
      <p>
        Unless autoplay is allowed for Sentinel, the intel alarm requires user
        input before it can play.
      </p>
      <p>
        You can enable sound (Chrome) / autoplay (Firefox) in the info menu on
        the left of the address bar.
      </p>
    </Modal>
  );
}
