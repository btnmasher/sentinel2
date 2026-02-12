import { useEffect } from "react";
import { X } from "lucide-react";
import type { ReactNode } from "react";
import type { ModalCloseReason } from "@/app/store/uiStore";

type ModalProps = {
  open: boolean;
  title?: string;
  children: ReactNode;
  onClose?: (
    reason?: ModalCloseReason,
  ) => boolean | void | Promise<boolean | void>;
  actions?: ReactNode;
  className?: string;
  closeDisabled?: boolean;
  dismissible?: boolean;
  closeOnOverlay?: boolean;
};

export default function Modal({
  open,
  title,
  children,
  onClose,
  actions,
  className,
  closeDisabled = false,
  dismissible = true,
  closeOnOverlay = false,
}: ModalProps) {
  useEffect(() => {
    if (!open || !onClose || closeDisabled || !dismissible) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") void onClose("escape");
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [closeDisabled, dismissible, onClose, open]);

  if (!open) return null;
  const canClose = Boolean(onClose) && dismissible && !closeDisabled;
  return (
    <div
      className="modal modal-open"
      onClick={(event) => {
        if (
          closeOnOverlay &&
          canClose &&
          event.target === event.currentTarget &&
          onClose
        ) {
          void onClose("overlay");
        }
      }}
    >
      <div
        className={`modal-box bg-base-200 border border-slate-700 max-w-lg relative ${className ?? ""}`}
        onClick={(event) => event.stopPropagation()}
      >
        {title && <h3 className="font-display text-lg mb-3">{title}</h3>}
        <div className="text-sm text-slate-300 space-y-3">{children}</div>
        {actions && <div className="modal-action">{actions}</div>}
        {onClose && dismissible && (
          <button
            className="btn btn-outline btn-sm btn-square absolute right-3 top-3"
            onClick={() => void onClose("button")}
            disabled={closeDisabled}
            aria-label="Close modal"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}
