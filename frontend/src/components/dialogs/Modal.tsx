import { useEffect, useRef, useState } from "react";
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
  const bodyRef = useRef<HTMLDivElement | null>(null);
  const shadowTimerRef = useRef<number | null>(null);
  const [scrollState, setScrollState] = useState({
    hasOverflow: false,
    scrolled: false,
    atBottom: true,
  });
  const [shadowsArmed, setShadowsArmed] = useState(false);

  const updateScrollState = () => {
    const element = bodyRef.current;
    if (!element) return;
    const { scrollTop, scrollHeight, clientHeight } = element;
    const hasOverflow = scrollHeight > clientHeight + 1;
    const scrolled = scrollTop > 1;
    const atBottom = scrollTop + clientHeight >= scrollHeight - 1;
    setScrollState({ hasOverflow, scrolled, atBottom });
  };

  useEffect(() => {
    if (!open || !onClose || closeDisabled || !dismissible) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") void onClose("escape");
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [closeDisabled, dismissible, onClose, open]);

  useEffect(() => {
    if (!open) return;
    const frame = window.requestAnimationFrame(updateScrollState);
    const onResize = () => updateScrollState();
    window.addEventListener("resize", onResize);
    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("resize", onResize);
    };
  }, [children, open, title, actions]);

  useEffect(() => {
    if (!open) {
      setShadowsArmed(false);
      if (shadowTimerRef.current) {
        window.clearTimeout(shadowTimerRef.current);
        shadowTimerRef.current = null;
      }
      return;
    }
    setShadowsArmed(false);
    shadowTimerRef.current = window.setTimeout(() => {
      setShadowsArmed(true);
      shadowTimerRef.current = null;
    }, 60);
    return () => {
      if (shadowTimerRef.current) {
        window.clearTimeout(shadowTimerRef.current);
        shadowTimerRef.current = null;
      }
    };
  }, [open]);

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
        className={`modal-box relative flex w-[calc(100vw-2rem)] max-h-[calc(100dvh-2rem)] max-w-lg flex-col overflow-hidden border border-slate-700 bg-base-200 p-0 ${className ?? ""}`}
        onClick={(event) => event.stopPropagation()}
      >
        {(title || (onClose && dismissible)) && (
          <div
            className={[
              "sticky top-0 z-10 shrink-0 bg-base-200/95 px-5 py-3 pr-14 backdrop-blur transition-shadow",
              shadowsArmed && scrollState.hasOverflow && scrollState.scrolled
                ? "shadow-[0_10px_20px_rgba(2,6,23,0.3)]"
                : "",
            ]
              .filter(Boolean)
              .join(" ")}
          >
            {title && <h3 className="font-display text-lg">{title}</h3>}
            {onClose && dismissible && (
              <button
                className="btn btn-outline btn-sm btn-square absolute right-3 top-2.5"
                onClick={() => void onClose("button")}
                disabled={closeDisabled}
                aria-label="Close modal"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
        )}
        <div
          ref={bodyRef}
          onScroll={updateScrollState}
          className="min-h-0 overflow-y-auto overscroll-contain px-5 py-4"
        >
          <div className="space-y-3 text-sm text-slate-300">{children}</div>
        </div>
        {actions && (
          <div
            className={[
              "shrink-0 px-5 py-3 transition-shadow",
              shadowsArmed && scrollState.hasOverflow && !scrollState.atBottom
                ? "shadow-[0_-10px_20px_rgba(2,6,23,0.3)]"
                : "",
            ]
              .filter(Boolean)
              .join(" ")}
          >
            <div className="modal-action mt-0">{actions}</div>
          </div>
        )}
      </div>
    </div>
  );
}
