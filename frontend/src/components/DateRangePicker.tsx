import { useEffect, useMemo, useRef, useState } from "react";
import { DayPicker } from "react-day-picker";
import { createPortal } from "react-dom";
import type { DateRange, Matcher } from "react-day-picker";

const formatRangeLabel = (range?: DateRange) => {
  if (!range?.from && !range?.to) return "All dates";
  if (range?.from && !range?.to) {
    return `${range.from.toLocaleDateString()} →`;
  }
  if (range?.from && range?.to) {
    return `${range.from.toLocaleDateString()} → ${range.to.toLocaleDateString()}`;
  }
  return "All dates";
};

const formatSingleLabel = (value?: Date) => {
  if (!value) return "Select date";
  return value.toLocaleDateString();
};

type DateRangePickerProps = {
  value?: DateRange;
  singleValue?: Date;
  mode?: "range" | "single";
  startHour?: number;
  endHour?: number;
  onApply?: (range?: DateRange, startHour?: number, endHour?: number) => void;
  onApplySingle?: (date?: Date, startHour?: number, endHour?: number) => void;
  onClear: () => void;
  disableFuture?: boolean;
  disablePast?: boolean;
  buttonClassName?: string;
  showTimeSelect?: boolean;
};

export default function DateRangePicker({
  value,
  singleValue,
  mode = "range",
  startHour,
  endHour,
  onApply,
  onApplySingle,
  onClear,
  disableFuture = true,
  disablePast = false,
  buttonClassName = "btn btn-xs btn-ghost",
  showTimeSelect = false,
}: DateRangePickerProps) {
  const [open, setOpen] = useState(false);
  const [pendingRange, setPendingRange] = useState<DateRange | undefined>(
    undefined,
  );
  const [pendingDate, setPendingDate] = useState<Date | undefined>(undefined);
  const [pendingStartHour, setPendingStartHour] = useState<number | undefined>(
    undefined,
  );
  const [pendingEndHour, setPendingEndHour] = useState<number | undefined>(
    undefined,
  );
  const pickerRef = useRef<HTMLDivElement | null>(null);
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  const hourOptions = useMemo(
    () =>
      Array.from({ length: 24 }, (_, hour) => ({
        value: hour,
        label: `${String(hour).padStart(2, "0")}:00`,
      })),
    [],
  );

  useEffect(() => {
    if (!open) return;
    setPendingRange(value);
    setPendingDate(singleValue);
    setPendingStartHour(startHour);
    setPendingEndHour(endHour);
  }, [open, value, singleValue, startHour, endHour]);

  useEffect(() => {
    if (!open) return;
    const updatePosition = () => {
      const button = buttonRef.current;
      if (!button) return;
      const rect = button.getBoundingClientRect();
      const margin = 8;
      const top = rect.bottom + margin;
      const left = Math.min(
        Math.max(margin, rect.left),
        window.innerWidth - 360,
      );
      setPosition({ top, left });
    };
    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handleClick = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (buttonRef.current?.contains(target)) return;
      if (pickerRef.current?.contains(target)) return;
      setOpen(false);
      setPendingRange(value);
      setPendingDate(singleValue);
      setPendingStartHour(startHour);
      setPendingEndHour(endHour);
    };
    const handleKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpen(false);
      setPendingRange(value);
      setPendingDate(singleValue);
      setPendingStartHour(startHour);
      setPendingEndHour(endHour);
    };
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open, value, singleValue, startHour, endHour]);

  const disabled = useMemo<Matcher[] | undefined>(() => {
    const result: Matcher[] = [];
    if (disableFuture) {
      result.push({ after: new Date() });
    }
    if (disablePast) {
      const today = new Date();
      today.setHours(0, 0, 0, 0);
      result.push({ before: today });
    }
    return result.length > 0 ? result : undefined;
  }, [disableFuture, disablePast]);

  const triggerLabel =
    mode === "single"
      ? formatSingleLabel(singleValue)
      : formatRangeLabel(value);

  return (
    <>
      <button
        ref={buttonRef}
        className={buttonClassName}
        onClick={() => setOpen((prev) => !prev)}
        type="button"
      >
        {triggerLabel}
      </button>
      {open &&
        createPortal(
          <div
            ref={pickerRef}
            className="fixed z-[10000] rounded-lg border border-slate-800 bg-base-200 p-3 shadow-lg"
            style={{ top: position.top, left: position.left }}
          >
            {mode === "single" ? (
              <DayPicker
                mode="single"
                selected={pendingDate}
                disabled={disabled}
                captionLayout="dropdown"
                endMonth={new Date()}
                onSelect={(next: Date | undefined) => {
                  setPendingDate(next);
                }}
                numberOfMonths={1}
                weekStartsOn={0}
                showOutsideDays
              />
            ) : (
              <DayPicker
                mode="range"
                selected={pendingRange}
                disabled={disabled}
                captionLayout="dropdown"
                endMonth={new Date()}
                onSelect={(next: DateRange | undefined) => {
                  setPendingRange(next);
                }}
                numberOfMonths={1}
                weekStartsOn={0}
                showOutsideDays
              />
            )}
            {showTimeSelect && (
              <div className="mt-3 grid gap-2 text-xs text-slate-400 sm:grid-cols-2">
                <label className="grid gap-1">
                  <span className="text-[10px] uppercase tracking-[0.2em] text-slate-500">
                    Start hour
                  </span>
                  <select
                    className="select select-xs select-bordered bg-base-300"
                    value={pendingStartHour ?? ""}
                    onChange={(event) => {
                      const next =
                        event.target.value === ""
                          ? undefined
                          : Number(event.target.value);
                      setPendingStartHour(next);
                    }}
                  >
                    <option value="">All</option>
                    {hourOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="grid gap-1">
                  <span className="text-[10px] uppercase tracking-[0.2em] text-slate-500">
                    End hour
                  </span>
                  <select
                    className="select select-xs select-bordered bg-base-300"
                    value={pendingEndHour ?? ""}
                    onChange={(event) => {
                      const next =
                        event.target.value === ""
                          ? undefined
                          : Number(event.target.value);
                      setPendingEndHour(next);
                    }}
                  >
                    <option value="">All</option>
                    {hourOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            )}
            <div className="mt-2 flex items-center justify-between text-xs text-slate-400">
              <span>
                {mode === "single"
                  ? "Select a date."
                  : "Select a start and end date."}
              </span>
              <div className="flex items-center gap-2">
                <button className="btn btn-xs btn-outline" onClick={onClear}>
                  Clear
                </button>
                <button
                  className="btn btn-xs btn-outline"
                  onClick={() => {
                    if (mode === "single") {
                      onApplySingle?.(
                        pendingDate,
                        pendingStartHour,
                        pendingEndHour,
                      );
                    } else {
                      onApply?.(pendingRange, pendingStartHour, pendingEndHour);
                    }
                    setOpen(false);
                  }}
                >
                  Done
                </button>
              </div>
            </div>
          </div>,
          document.body,
        )}
    </>
  );
}
