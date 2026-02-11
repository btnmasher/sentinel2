import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { useUIStore } from "@/app/store/uiStore";

export default function Toast() {
  const toast = useUIStore((s) => s.toast);
  const clear = useUIStore((s) => s.clearToast);
  const [hovered, setHovered] = useState(false);
  const timeoutRef = useRef<number | null>(null);

  useEffect(() => {
    if (!toast) return;
    if (hovered) return;
    const timeout = toast.timeout ?? 3000;
    if (timeoutRef.current) {
      window.clearTimeout(timeoutRef.current);
    }
    timeoutRef.current = window.setTimeout(() => {
      clear();
      timeoutRef.current = null;
    }, timeout);
    return () => {
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }
    };
  }, [toast, clear, hovered]);

  if (!toast) return null;

  const color = toast.color ?? "secondary";
  const colorClass = {
    secondary: "border-base-content/25 bg-base-100/90 text-base-content",
    error: "border-error/40 bg-error/10 text-error",
    success: "border-success/40 bg-success/10 text-success",
    warning: "border-warning/40 bg-warning/10 text-warning",
    info: "border-info/40 bg-info/10 text-info",
  }[color];

  return (
    <div className="fixed top-4 left-1/2 -translate-x-1/2 z-50">
      <div
        role="button"
        tabIndex={0}
        aria-label="Dismiss notification"
        className={`rounded-xl border px-4 py-2 shadow-lg backdrop-blur cursor-pointer ${colorClass}`}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        onFocus={() => setHovered(true)}
        onBlur={() => setHovered(false)}
        onClick={() => clear()}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            clear();
          }
        }}
      >
        <div className="flex items-center gap-3">
          <span className="text-sm">{toast.text}</span>
          <button
            type="button"
            className="ml-auto inline-flex h-6 w-6 items-center justify-center rounded-full text-slate-300 hover:text-slate-100 hover:bg-white/10"
            onClick={(event) => {
              event.stopPropagation();
              clear();
            }}
            aria-label="Close notification"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
