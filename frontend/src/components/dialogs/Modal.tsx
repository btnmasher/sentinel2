import type { ReactNode } from "react";

type ModalProps = {
  open: boolean;
  title: string;
  children: ReactNode;
  onClose?: () => void;
  actions?: ReactNode;
};

export default function Modal({
  open,
  title,
  children,
  onClose,
  actions,
}: ModalProps) {
  if (!open) return null;
  return (
    <div className="modal modal-open">
      <div className="modal-box bg-base-200 border border-slate-700 max-w-lg">
        <h3 className="font-display text-lg mb-3">{title}</h3>
        <div className="text-sm text-slate-300 space-y-3">{children}</div>
        {actions && <div className="modal-action">{actions}</div>}
        {onClose && (
          <button
            className="btn btn-outline btn-sm btn-square absolute right-3 top-3"
            onClick={onClose}
          >
            ✕
          </button>
        )}
      </div>
    </div>
  );
}
