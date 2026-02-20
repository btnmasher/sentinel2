import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { ChevronDown, X } from "lucide-react";

type SelectionItem = {
  id: string;
  label: string;
  description?: string;
  disabled?: boolean;
  kind?: "item" | "section";
};

type SelectionDropdownProps = {
  items: SelectionItem[];
  selected: string[];
  onChange: (next: string[]) => void;
  multi?: boolean;
  showTags?: boolean;
  coalesceAllTags?: boolean;
  searchable?: boolean;
  disabled?: boolean;
  label?: string;
  placeholder?: string;
  buttonClassName?: string;
  menuClassName?: string;
};

const normalizeId = (value: string) => value;

const hasSelection = (selected: string[]) => selected.length > 0;

export default function SelectionDropdown({
  items,
  selected,
  onChange,
  multi = false,
  showTags = false,
  coalesceAllTags = false,
  searchable = false,
  disabled = false,
  label,
  placeholder,
  buttonClassName,
  menuClassName,
}: SelectionDropdownProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const itemRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const [menuPosition, setMenuPosition] = useState<{
    top: number;
    left: number;
    width: number;
  } | null>(null);

  useEffect(() => {
    if (!open || disabled) return;
    if (searchable) {
      inputRef.current?.focus();
    }
  }, [disabled, open, searchable]);

  useEffect(() => {
    if (disabled && open) {
      setOpen(false);
    }
  }, [disabled, open]);

  useEffect(() => {
    if (!open) return;
    const handler = (event: MouseEvent) => {
      const target = event.target as Node;
      const inTrigger =
        containerRef.current && containerRef.current.contains(target);
      const inMenu = menuRef.current && menuRef.current.contains(target);
      if (!inTrigger && !inMenu) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handler = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        setOpen(false);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open]);

  useLayoutEffect(() => {
    if (!open) return;
    const updatePosition = () => {
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) return;
      setMenuPosition({
        top: rect.bottom + 8,
        left: rect.left,
        width: rect.width,
      });
    };
    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [open]);

  const selectedSet = useMemo(
    () => new Set(selected.map((id) => normalizeId(id))),
    [selected],
  );

  const filteredItems = useMemo(() => {
    if (!searchable || !search.trim()) return items;
    const needle = search.trim().toLowerCase();
    const filtered: SelectionItem[] = [];
    let pendingSection: SelectionItem | null = null;
    let sectionAdded = false;

    items.forEach((item) => {
      if (item.kind === "section") {
        pendingSection = item;
        sectionAdded = false;
        return;
      }
      const matches = [item.label, item.description]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(needle));
      if (!matches) return;
      if (pendingSection && !sectionAdded) {
        filtered.push(pendingSection);
        sectionAdded = true;
      }
      filtered.push(item);
    });

    return filtered;
  }, [items, search, searchable]);

  const selectedLabel = useMemo(() => {
    if (!hasSelection(selected)) {
      return placeholder ?? label ?? "Select";
    }
    if (multi) {
      if (showTags) return "";
      return label
        ? `${label} (${selected.length})`
        : `${selected.length} selected`;
    }
    const match = items.find((item) => item.id === selected[0]);
    return match?.label ?? selected[0];
  }, [items, label, multi, placeholder, selected, showTags]);

  const toggleItem = (id: string, shouldClose: boolean) => {
    if (multi) {
      const next = selectedSet.has(id)
        ? selected.filter((value) => normalizeId(value) !== id)
        : [...selected, id];
      onChange(next);
      if (shouldClose) setOpen(false);
      return;
    }
    onChange([id]);
    if (shouldClose) setOpen(false);
  };

  const focusFirstItem = () => {
    const index = filteredItems.findIndex(
      (item) => item.kind !== "section" && !item.disabled,
    );
    if (index >= 0) {
      itemRefs.current[index]?.focus();
    }
  };

  const handleInputKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      focusFirstItem();
      return;
    }
    if (event.key === "Enter") {
      if (filteredItems.length === 1) {
        const item = filteredItems[0];
        if (!item.disabled) {
          event.preventDefault();
          toggleItem(item.id, true);
        }
      }
    }
  };

  const handleItemKeyDown = (
    event: React.KeyboardEvent<HTMLButtonElement>,
    index: number,
    item: SelectionItem,
  ) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      const nextIndex = filteredItems.findIndex(
        (candidate, idx) =>
          idx > index && candidate.kind !== "section" && !candidate.disabled,
      );
      if (nextIndex >= 0) {
        itemRefs.current[nextIndex]?.focus();
      }
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      const reversed = [...filteredItems]
        .slice(0, index)
        .reverse()
        .findIndex(
          (candidate) => candidate.kind !== "section" && !candidate.disabled,
        );
      if (reversed >= 0) {
        const targetIndex = index - 1 - reversed;
        itemRefs.current[targetIndex]?.focus();
      } else if (searchable) {
        inputRef.current?.focus();
      }
      return;
    }
    if (event.key === " ") {
      event.preventDefault();
      if (!item.disabled) {
        toggleItem(item.id, false);
      }
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      if (!item.disabled) {
        toggleItem(item.id, true);
      }
    }
  };

  const renderTags = () => {
    if (!showTags || !multi) return null;
    if (
      coalesceAllTags &&
      selected.length > 0 &&
      selected.length === items.length
    ) {
      return (
        <div className="flex flex-wrap gap-1">
          <span className="badge badge-xs border border-slate-700 bg-base-300/70 px-1 min-h-[0.9rem] text-[10px] leading-none inline-flex items-center gap-1">
            All
          </span>
        </div>
      );
    }
    const maxTags = 5;
    const selectedItems = selected
      .map((id) => items.find((item) => item.id === id))
      .filter(Boolean);
    const visibleItems = selectedItems.slice(0, maxTags);
    const hiddenCount = selectedItems.length - visibleItems.length;
    return (
      <div className="flex flex-wrap gap-1">
        {visibleItems.map((item) => (
          <span
            key={item!.id}
            className="badge badge-xs border border-slate-700 bg-base-300/70 px-1 min-h-[0.9rem] text-[10px] leading-none inline-flex items-center gap-1"
          >
            {item!.label}
            <span
              role="button"
              tabIndex={0}
              className="inline-flex h-3.5 w-3.5 items-center justify-center rounded-full border border-slate-600/70 text-slate-300 transition-colors hover:bg-base-content/15 hover:text-base-content"
              onClick={(event) => {
                event.stopPropagation();
                toggleItem(item!.id, false);
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  event.stopPropagation();
                  toggleItem(item!.id, false);
                }
              }}
              aria-label={`Remove ${item!.label}`}
            >
              <X className="h-2.5 w-2.5" />
            </span>
          </span>
        ))}
        {hiddenCount > 0 && (
          <span className="text-[10px] text-slate-400">
            +{hiddenCount} more
          </span>
        )}
        {!hasSelection(selected) && (
          <span className="text-slate-400 text-[11px]">
            {placeholder ?? label ?? "Select"}
          </span>
        )}
      </div>
    );
  };

  const menu =
    open && !disabled && menuPosition
      ? createPortal(
          <div
            ref={menuRef}
            className={`rounded-lg border border-slate-800 bg-base-200/95 shadow-lg backdrop-blur z-1000 ${
              menuClassName ?? ""
            }`}
            style={{
              position: "fixed",
              top: menuPosition.top,
              left: menuPosition.left,
              width: menuPosition.width,
            }}
          >
            {searchable && (
              <div className="border-b border-slate-800 p-2">
                <input
                  ref={inputRef}
                  className="input input-xs input-bordered bg-base-300 w-full"
                  placeholder={placeholder ?? label ?? "Search"}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  onKeyDown={handleInputKeyDown}
                />
              </div>
            )}
            <div className="max-h-56 overflow-auto px-2 py-2 space-y-1">
              {filteredItems.map((item, index) => {
                const checked = selectedSet.has(item.id);
                const disabled = Boolean(item.disabled);
                const isSection = item.kind === "section";
                if (isSection) {
                  return (
                    <div
                      key={item.id}
                      className="mt-1 border-t border-slate-700/70 pt-1 text-[10px] uppercase tracking-[0.2em] text-slate-500"
                    >
                      {item.label}
                    </div>
                  );
                }
                return (
                  <button
                    type="button"
                    key={item.id}
                    ref={(ref) => {
                      itemRefs.current[index] = ref;
                    }}
                    className={`w-full flex items-center gap-2 px-2 py-1 rounded text-left ${
                      disabled ? "opacity-40" : "hover:bg-base-300"
                    } ${checked ? "bg-base-300" : ""}`}
                    onClick={() => {
                      if (disabled) return;
                      toggleItem(item.id, !multi);
                    }}
                    onKeyDown={(event) => handleItemKeyDown(event, index, item)}
                    disabled={disabled}
                  >
                    {multi && (
                      <input
                        type="checkbox"
                        className="checkbox checkbox-xs rounded-[3px]"
                        checked={checked}
                        readOnly
                      />
                    )}
                    <span className="text-xs text-slate-200">{item.label}</span>
                  </button>
                );
              })}
              {filteredItems.length === 0 && (
                <div className="px-2 py-2 text-xs text-slate-500">
                  No results
                </div>
              )}
            </div>
          </div>,
          document.body,
        )
      : null;

  return (
    <div className="relative" ref={containerRef}>
      <button
        type="button"
        disabled={disabled}
        className={`btn btn-xs btn-ghost justify-between min-w-35 ${
          buttonClassName ?? ""
        }`}
        onClick={() => setOpen((prev) => !prev)}
      >
        {showTags && multi ? (
          <span className="flex-1 text-left">{renderTags()}</span>
        ) : (
          <span>{selectedLabel}</span>
        )}
        <ChevronDown className="h-3.5 w-3.5" />
      </button>
      {menu}
    </div>
  );
}
