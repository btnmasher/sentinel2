import { useEffect, useMemo, useRef, useState } from "react";
import { DayPicker } from "react-day-picker";
import type { DateRange } from "react-day-picker";

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

type DateRangePickerProps = {
  value?: DateRange;
  startHour?: number;
  endHour?: number;
  onApply: (range?: DateRange, startHour?: number, endHour?: number) => void;
  onClear: () => void;
  disableFuture?: boolean;
  buttonClassName?: string;
  showTimeSelect?: boolean;
};

export default function DateRangePicker({
  value,
  startHour,
  endHour,
  onApply,
  onClear,
  disableFuture = true,
  buttonClassName = "btn btn-xs btn-ghost",
  showTimeSelect = false,
}: DateRangePickerProps) {
  const [open, setOpen] = useState(false);
  const [pendingRange, setPendingRange] = useState<DateRange | undefined>(
    undefined,
  );
  const [pendingStartHour, setPendingStartHour] = useState<number | undefined>(
    undefined,
  );
  const [pendingEndHour, setPendingEndHour] = useState<number | undefined>(
    undefined,
  );
  const pickerRef = useRef<HTMLDivElement | null>(null);

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
    setPendingStartHour(startHour);
    setPendingEndHour(endHour);
  }, [open, value, startHour, endHour]);

  useEffect(() => {
    if (!open) return;
    const handleClick = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (pickerRef.current?.contains(target)) return;
      setOpen(false);
      setPendingRange(value);
      setPendingStartHour(startHour);
      setPendingEndHour(endHour);
    };
    const handleKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpen(false);
      setPendingRange(value);
      setPendingStartHour(startHour);
      setPendingEndHour(endHour);
    };
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open, value, startHour, endHour]);

  return (
    <div className="relative">
      <button
        className={buttonClassName}
        onClick={() => setOpen((prev) => !prev)}
        type="button"
      >
        {formatRangeLabel(value)}
      </button>
      {open && (
        <div
          ref={pickerRef}
          className="absolute z-30 mt-2 top-full left-0 rounded-lg border border-slate-800 bg-base-200 shadow-lg p-3"
        >
          <DayPicker
            mode="range"
            selected={pendingRange}
            disabled={disableFuture ? { after: new Date() } : undefined}
            captionLayout="dropdown"
            endMonth={new Date()}
            onSelect={(range) => {
              setPendingRange(range);
            }}
            numberOfMonths={1}
            weekStartsOn={0}
            showOutsideDays
          />
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
            <span>Select a start and end date.</span>
            <div className="flex items-center gap-2">
              <button className="btn btn-xs btn-outline" onClick={onClear}>
                Clear
              </button>
              <button
                className="btn btn-xs btn-outline"
                onClick={() => {
                  onApply(pendingRange, pendingStartHour, pendingEndHour);
                  setOpen(false);
                }}
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
