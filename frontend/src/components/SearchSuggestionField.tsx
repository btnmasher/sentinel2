import { useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";

type SearchSuggestionFieldProps<T> = {
  query: string;
  onQueryChange: (value: string) => void;
  onSelect: (item: T) => void;
  loadSuggestions: (query: string) => Promise<T[]>;
  getSuggestionKey: (item: T) => string | number;
  getInputValueFromSuggestion?: (item: T) => string;
  selectionInputMode?: "set" | "clear" | "preserve";
  renderSuggestion: (item: T, active: boolean) => ReactNode;
  placeholder?: string;
  inputClassName?: string;
  containerClassName?: string;
  panelClassName?: string;
  minQueryLength?: number;
  debounceMs?: number;
  usePortal?: boolean;
};

export default function SearchSuggestionField<T>({
  query,
  onQueryChange,
  onSelect,
  loadSuggestions,
  getSuggestionKey,
  getInputValueFromSuggestion,
  selectionInputMode = "set",
  renderSuggestion,
  placeholder = "Search",
  inputClassName = "input input-bordered w-full",
  containerClassName = "w-full",
  panelClassName = "max-h-44 overflow-auto rounded-lg border border-slate-700/70 bg-base-300/95 shadow-xl",
  minQueryLength = 2,
  debounceMs = 250,
  usePortal = true,
}: SearchSuggestionFieldProps<T>) {
  const [loading, setLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<T[]>([]);
  const [cursor, setCursor] = useState(-1);
  const [isInputActive, setIsInputActive] = useState(false);
  const [panelPos, setPanelPos] = useState({ top: 0, left: 0, width: 0 });
  const rootRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const panelRef = useRef<HTMLDivElement | null>(null);
  const requestIdRef = useRef(0);

  const trimmedQuery = query.trim();
  const showSuggestions = useMemo(
    () =>
      isInputActive &&
      trimmedQuery.length >= minQueryLength &&
      suggestions.length > 0,
    [isInputActive, minQueryLength, suggestions.length, trimmedQuery.length],
  );
  const showPanel = useMemo(
    () =>
      isInputActive &&
      trimmedQuery.length >= minQueryLength &&
      (loading || suggestions.length > 0),
    [
      isInputActive,
      loading,
      minQueryLength,
      suggestions.length,
      trimmedQuery.length,
    ],
  );

  useEffect(() => {
    if (!isInputActive) {
      setLoading(false);
      return;
    }
    if (trimmedQuery.length < minQueryLength) {
      setSuggestions([]);
      setCursor(-1);
      setLoading(false);
      return;
    }
    const timeout = window.setTimeout(async () => {
      const requestId = ++requestIdRef.current;
      setLoading(true);
      try {
        const next = await loadSuggestions(trimmedQuery);
        if (requestId !== requestIdRef.current) return;
        setSuggestions(next || []);
        setCursor(next?.length ? 0 : -1);
      } catch {
        if (requestId !== requestIdRef.current) return;
        setSuggestions([]);
        setCursor(-1);
      } finally {
        if (requestId === requestIdRef.current) {
          setLoading(false);
        }
      }
    }, debounceMs);
    return () => window.clearTimeout(timeout);
  }, [
    debounceMs,
    isInputActive,
    loadSuggestions,
    minQueryLength,
    trimmedQuery,
  ]);

  useEffect(() => {
    if (!showPanel) return;
    const updatePosition = () => {
      const input = inputRef.current;
      const root = rootRef.current;
      if (!input || !root) return;
      const inputRect = input.getBoundingClientRect();
      const rootRect = root.getBoundingClientRect();

      if (usePortal) {
        setPanelPos({
          top: inputRect.bottom + 4,
          left: inputRect.left,
          width: inputRect.width,
        });
        return;
      }

      setPanelPos({
        top: input.offsetTop + input.offsetHeight + 4,
        left: input.offsetLeft + (inputRect.left - rootRect.left),
        width: input.offsetWidth,
      });
    };
    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [showPanel, usePortal]);

  useEffect(() => {
    if (!showPanel) return;
    const handleOutsideClick = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if (panelRef.current?.contains(target)) return;
      setSuggestions([]);
      setCursor(-1);
      setIsInputActive(false);
    };
    document.addEventListener("mousedown", handleOutsideClick);
    return () => document.removeEventListener("mousedown", handleOutsideClick);
  }, [showPanel]);

  const commitSelection = (item: T) => {
    // Treat selection as completion of the current query interaction.
    setIsInputActive(false);
    requestIdRef.current += 1;
    setLoading(false);
    if (selectionInputMode === "clear") {
      onQueryChange("");
    } else if (selectionInputMode === "set" && getInputValueFromSuggestion) {
      onQueryChange(getInputValueFromSuggestion(item));
    }
    onSelect(item);
    setSuggestions([]);
    setCursor(-1);
    inputRef.current?.blur();
  };

  const panelNode = showPanel ? (
    <div
      ref={panelRef}
      className={
        usePortal ? panelClassName : `absolute z-[11000] ${panelClassName}`
      }
      style={
        usePortal
          ? {
              position: "fixed",
              top: panelPos.top,
              left: panelPos.left,
              width: panelPos.width,
              zIndex: 11000,
            }
          : {
              top: panelPos.top,
              left: panelPos.left,
              width: panelPos.width,
            }
      }
    >
      {loading && (
        <div className="flex items-center gap-2 px-3 py-2 text-sm text-slate-300">
          <span className="loading loading-spinner loading-sm text-success" />
          <span>Searching...</span>
        </div>
      )}
      {suggestions.map((item, index) => (
        <button
          key={String(getSuggestionKey(item))}
          className={`w-full px-3 py-2 text-left text-sm hover:bg-base-300 ${
            index === cursor ? "bg-base-200 ring-1 ring-sky-500/35" : ""
          }`}
          onMouseEnter={() => setCursor(index)}
          onClick={() => commitSelection(item)}
          type="button"
        >
          {renderSuggestion(item, index === cursor)}
        </button>
      ))}
    </div>
  ) : null;

  return (
    <div ref={rootRef} className={`relative ${containerClassName}`}>
      <input
        ref={inputRef}
        className={`${inputClassName} pr-9`}
        value={query}
        onFocus={() => setIsInputActive(true)}
        onChange={(event) => onQueryChange(event.target.value)}
        onKeyDown={(event) => {
          if (!showSuggestions) return;
          if (event.key === "ArrowDown") {
            event.preventDefault();
            setCursor((value) =>
              Math.min(value + 1, Math.max(suggestions.length - 1, 0)),
            );
            return;
          }
          if (event.key === "ArrowUp") {
            event.preventDefault();
            setCursor((value) => Math.max(value - 1, 0));
            return;
          }
          if (event.key === "Escape") {
            event.preventDefault();
            setSuggestions([]);
            setCursor(-1);
            return;
          }
          if (event.key === "Enter") {
            event.preventDefault();
            if (cursor >= 0 && cursor < suggestions.length) {
              commitSelection(suggestions[cursor]);
            }
          }
        }}
        placeholder={placeholder}
      />
      {panelNode && !usePortal && panelNode}
      {panelNode && usePortal && typeof document !== "undefined"
        ? createPortal(panelNode, document.body)
        : null}
    </div>
  );
}
