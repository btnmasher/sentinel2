import { useUIStore } from "@/app/store/uiStore";
import { useShallow } from "zustand/shallow";

export default function ConfirmDialog() {
  const { dialogs, confirmTitle, confirmBody, confirmAction, clearConfirm } =
    useUIStore(
      useShallow((s) => ({
        dialogs: s.dialogs,
        confirmTitle: s.confirmTitle,
        confirmBody: s.confirmBody,
        confirmAction: s.confirmAction,
        clearConfirm: s.clearConfirm,
      })),
    );

  if (!dialogs.confirm) return null;

  return (
    <div className="modal modal-open">
      <div className="modal-box bg-base-200 border border-slate-700 max-w-lg">
        <h3 className="font-display text-lg mb-3">
          {confirmTitle || "Confirm"}
        </h3>
        <div className="text-sm text-slate-300 space-y-3">
          <p>{confirmBody}</p>
        </div>
        <div className="modal-action">
          <button className="btn btn-sm btn-outline" onClick={clearConfirm}>
            Cancel
          </button>
          <button
            className="btn btn-sm btn-error btn-outline"
            onClick={() => {
              const action = confirmAction;
              clearConfirm();
              if (action) action();
            }}
          >
            Confirm
          </button>
        </div>
        <button
          className="btn btn-outline btn-sm btn-square absolute right-3 top-3"
          onClick={clearConfirm}
        >
          ✕
        </button>
      </div>
    </div>
  );
}
