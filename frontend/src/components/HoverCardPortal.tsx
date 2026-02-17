import { useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { ReactNode, RefObject } from "react";

type HoverCardPortalProps = {
  anchorRef: RefObject<HTMLElement | null>;
  open: boolean;
  children: ReactNode;
  className?: string;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
};

export default function HoverCardPortal({
  anchorRef,
  open,
  children,
  className,
  onMouseEnter,
  onMouseLeave,
}: HoverCardPortalProps) {
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  useLayoutEffect(() => {
    if (!open) return;
    const updatePosition = () => {
      const anchor = anchorRef.current;
      const card = cardRef.current;
      if (!anchor || !card) return;
      const rect = anchor.getBoundingClientRect();
      const cardWidth = card.offsetWidth;
      const cardHeight = card.offsetHeight;
      const margin = 8;
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      let left = rect.right - cardWidth;
      left = Math.max(margin, Math.min(left, viewportWidth - cardWidth - margin));

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
  }, [anchorRef, open]);

  if (!open || typeof document === "undefined") return null;

  return createPortal(
    <div
      ref={cardRef}
      className={className}
      style={{
        position: "fixed",
        top: position.top,
        left: position.left,
        zIndex: 2147483647,
      }}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
    >
      {children}
    </div>,
    document.body,
  );
}
