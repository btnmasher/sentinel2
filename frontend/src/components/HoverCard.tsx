import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { ReactNode, RefObject } from "react";

type HoverCardControl = {
  open: boolean;
  show: () => void;
  hide: () => void;
};

export function useHoverCardControl(closeDelayMs = 90): HoverCardControl {
  const [open, setOpen] = useState(false);
  const closeTimerRef = useRef<number | null>(null);

  const show = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
    setOpen(true);
  };

  const hide = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current);
    }
    closeTimerRef.current = window.setTimeout(() => {
      setOpen(false);
      closeTimerRef.current = null;
    }, closeDelayMs);
  };

  useEffect(
    () => () => {
      if (closeTimerRef.current) {
        window.clearTimeout(closeTimerRef.current);
      }
    },
    [],
  );

  return { open, show, hide };
}

type HoverCardProps = {
  anchorRef?: RefObject<HTMLElement | null>;
  open?: boolean;
  trigger?: ReactNode;
  children: ReactNode;
  className?: string;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
  closeDelayMs?: number;
};

export default function HoverCard({
  anchorRef,
  open,
  trigger,
  children,
  className,
  onMouseEnter,
  onMouseLeave,
  closeDelayMs = 90,
}: HoverCardProps) {
  const controlled = Boolean(anchorRef);
  const triggerRef = useRef<HTMLSpanElement | null>(null);
  const control = useHoverCardControl(closeDelayMs);
  const resolvedOpen = controlled ? Boolean(open) : control.open;
  const resolvedAnchorRef = (
    controlled ? anchorRef : triggerRef
  ) as RefObject<HTMLElement | null>;
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  useLayoutEffect(() => {
    if (!resolvedOpen) return;
    const updatePosition = () => {
      const anchor = resolvedAnchorRef.current;
      const card = cardRef.current;
      if (!anchor || !card) return;
      const rect = anchor.getBoundingClientRect();
      const cardWidth = card.offsetWidth;
      const cardHeight = card.offsetHeight;
      const margin = 8;
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      let left = rect.right - cardWidth;
      left = Math.max(
        margin,
        Math.min(left, viewportWidth - cardWidth - margin),
      );

      let top = rect.bottom + margin;
      const wouldOverflowBottom = top + cardHeight > viewportHeight - margin;
      if (wouldOverflowBottom) {
        top = Math.max(margin, rect.top - cardHeight - margin);
      }

      setPosition({ top, left });
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [resolvedAnchorRef, resolvedOpen]);

  if (!controlled && !trigger) {
    return null;
  }

  const triggerNode = !controlled ? (
    <span
      ref={triggerRef}
      className="inline-flex"
      onMouseEnter={() => {
        control.show();
        onMouseEnter?.();
      }}
      onMouseLeave={() => {
        control.hide();
        onMouseLeave?.();
      }}
      onFocusCapture={() => {
        control.show();
        onMouseEnter?.();
      }}
      onBlurCapture={(event) => {
        const next = event.relatedTarget as Node | null;
        if (next && event.currentTarget.contains(next)) return;
        control.hide();
        onMouseLeave?.();
      }}
    >
      {trigger}
    </span>
  ) : null;

  const cardNode =
    resolvedOpen && typeof document !== "undefined"
      ? createPortal(
          <div
            ref={cardRef}
            className={className}
            style={{
              position: "fixed",
              top: position.top,
              left: position.left,
              zIndex: 2147483647,
            }}
            onMouseEnter={() => {
              if (!controlled) {
                control.show();
              }
              onMouseEnter?.();
            }}
            onMouseLeave={() => {
              if (!controlled) {
                control.hide();
              }
              onMouseLeave?.();
            }}
          >
            {children}
          </div>,
          document.body,
        )
      : null;

  if (!controlled) {
    return (
      <>
        {triggerNode}
        {cardNode}
      </>
    );
  }

  return cardNode;
}
