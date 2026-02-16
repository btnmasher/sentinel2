import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";

type ShadowedScrollAreaProps = {
  children: ReactNode;
  className?: string;
  scrollClassName?: string;
};

export default function ShadowedScrollArea({
  children,
  className,
  scrollClassName,
}: ShadowedScrollAreaProps) {
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const [scrollState, setScrollState] = useState({
    hasOverflow: false,
    scrolled: false,
    atBottom: true,
  });

  const updateScrollState = () => {
    const element = scrollRef.current;
    if (!element) return;
    const { scrollTop, scrollHeight, clientHeight } = element;
    const hasOverflow = scrollHeight > clientHeight + 1;
    const scrolled = scrollTop > 1;
    const atBottom = scrollTop + clientHeight >= scrollHeight - 1;
    setScrollState({ hasOverflow, scrolled, atBottom });
  };

  useEffect(() => {
    const frame = window.requestAnimationFrame(updateScrollState);
    const onResize = () => updateScrollState();
    window.addEventListener("resize", onResize);
    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("resize", onResize);
    };
  }, [children]);

  return (
    <div
      className={`relative h-full min-h-0 overflow-hidden ${className ?? ""}`.trim()}
    >
      <div
        ref={scrollRef}
        onScroll={updateScrollState}
        className={`h-full min-h-0 overflow-y-auto overscroll-contain ${scrollClassName ?? ""}`.trim()}
      >
        {children}
      </div>
      <div
        className={[
          "pointer-events-none absolute inset-x-0 top-0 h-5 bg-gradient-to-b from-base-200/85 to-transparent transition-opacity duration-150",
          scrollState.hasOverflow && scrollState.scrolled
            ? "opacity-100"
            : "opacity-0",
        ]
          .filter(Boolean)
          .join(" ")}
      />
      <div
        className={[
          "pointer-events-none absolute inset-x-0 bottom-0 h-5 bg-gradient-to-t from-base-200/85 to-transparent transition-opacity duration-150",
          scrollState.hasOverflow && !scrollState.atBottom
            ? "opacity-100"
            : "opacity-0",
        ]
          .filter(Boolean)
          .join(" ")}
      />
    </div>
  );
}
