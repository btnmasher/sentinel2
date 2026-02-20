import { useEffect, useMemo, useRef, useState } from "react";
import { DayPicker } from "react-day-picker";
import { createPortal } from "react-dom";
import { CalendarDays } from "lucide-react";
import { enGB } from "date-fns/locale";
import { format, isValid, parse, set } from "date-fns";

type DateTimePickerProps = {
  value?: Date;
  onApply: (value?: Date) => void;
  onClear?: () => void;
  disablePast?: boolean;
  disableFuture?: boolean;
  buttonClassName?: string;
};

const formatLabel = (value?: Date) => {
  if (!value) return "Select date and time";
  return format(value, "yyyy-MM-dd HH:mm:ss");
};

const toDisplayParts = (value?: Date) => {
  if (!value) return null;
  const parts = format(value, "yyyy-MM-dd-HH-mm-ss").split("-");
  return {
    year: parts[0] ?? "0000",
    month: parts[1] ?? "00",
    day: parts[2] ?? "00",
    hour: parts[3] ?? "00",
    minute: parts[4] ?? "00",
    second: parts[5] ?? "00",
  };
};

const buildDateTime = (date?: Date, time?: string) => {
  if (!date) return undefined;
  const parsedTime = parse(time || "00:00:00", "HH:mm:ss", date);
  if (!isValid(parsedTime)) {
    return set(date, { hours: 0, minutes: 0, seconds: 0, milliseconds: 0 });
  }
  return set(date, {
    hours: parsedTime.getHours(),
    minutes: parsedTime.getMinutes(),
    seconds: parsedTime.getSeconds(),
    milliseconds: 0,
  });
};

const parseTimeParts = (value?: string) => {
  const [rawHour, rawMinute, rawSecond] = (value || "00:00:00").split(":");
  return {
    hour: String(Number(rawHour) || 0).padStart(2, "0"),
    minute: String(Number(rawMinute) || 0).padStart(2, "0"),
    second: String(Number(rawSecond) || 0).padStart(2, "0"),
  };
};

const hourOptions = Array.from({ length: 24 }, (_, value) =>
  String(value).padStart(2, "0"),
);
const minuteSecondOptions = Array.from({ length: 60 }, (_, value) =>
  String(value).padStart(2, "0"),
);

const eveNowDisplayDate = () => {
  const now = new Date();
  return new Date(
    now.getUTCFullYear(),
    now.getUTCMonth(),
    now.getUTCDate(),
    now.getUTCHours(),
    now.getUTCMinutes(),
    now.getUTCSeconds(),
    0,
  );
};

export default function DateTimePicker({
  value,
  onApply,
  onClear,
  disablePast = false,
  disableFuture = false,
  buttonClassName = "input input-bordered w-full justify-between",
}: DateTimePickerProps) {
  const [open, setOpen] = useState(false);
  const [displayValue, setDisplayValue] = useState<Date | undefined>(value);
  const [pendingDate, setPendingDate] = useState<Date | undefined>(undefined);
  const [pendingTime, setPendingTime] = useState("00:00:00");
  const pickerRef = useRef<HTMLDivElement | null>(null);
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  const disabled = useMemo(() => {
    const now = eveNowDisplayDate();
    const today = set(now, {
      hours: 0,
      minutes: 0,
      seconds: 0,
      milliseconds: 0,
    });
    if (disablePast && disableFuture) {
      return {
        before: today,
        after: today,
      };
    }
    if (disablePast) {
      return { before: today };
    }
    if (disableFuture) {
      return { after: today };
    }
    return undefined;
  }, [disableFuture, disablePast]);

  useEffect(() => {
    setDisplayValue(value);
  }, [value]);

  useEffect(() => {
    if (!open) return;
    const seed = displayValue ? new Date(displayValue) : eveNowDisplayDate();
    const seedDate = new Date(
      seed.getFullYear(),
      seed.getMonth(),
      seed.getDate(),
    );
    setPendingDate(seedDate);
    setPendingTime(
      displayValue
        ? `${String(displayValue.getHours()).padStart(2, "0")}:${String(displayValue.getMinutes()).padStart(2, "0")}:${String(displayValue.getSeconds()).padStart(2, "0")}`
        : `${String(seed.getHours()).padStart(2, "0")}:${String(seed.getMinutes()).padStart(2, "0")}:${String(seed.getSeconds()).padStart(2, "0")}`,
    );
  }, [displayValue, open]);

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
    };
    const handleKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpen(false);
    };
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open]);

  return (
    <div lang="en-GB">
      {(() => {
        const parts = toDisplayParts(displayValue);
        return (
          <button
            ref={buttonRef}
            className={`flex min-w-[22rem] items-center gap-2 text-left ${buttonClassName}`}
            type="button"
            onClick={() => setOpen((prev) => !prev)}
          >
            <CalendarDays className="h-4 w-4 text-slate-400" />
            {parts ? (
              <span className="flex items-center gap-1 whitespace-nowrap font-mono text-[1rem] font-bold tracking-[0.04em] text-base-content tabular-nums">
                <span>{parts.year}</span>
                <span className="text-success/85">-</span>
                <span>{parts.month}</span>
                <span className="text-success/85">-</span>
                <span>{parts.day}</span>
                <span className="mx-0.5 text-slate-500">|</span>
                <span>{parts.hour}</span>
                <span className="text-success/85">:</span>
                <span>{parts.minute}</span>
                <span className="text-success/85">:</span>
                <span>{parts.second}</span>
              </span>
            ) : (
              <span className="truncate font-display text-[0.95rem] tracking-[0.01em] text-base-content">
                {formatLabel(displayValue)}
              </span>
            )}
          </button>
        );
      })()}
      {open &&
        createPortal(
          <div
            ref={pickerRef}
            className="fixed z-[10000] rounded-xl border border-slate-700/70 bg-base-200/95 p-4 shadow-lg"
            style={{ top: position.top, left: position.left }}
          >
            <DayPicker
              mode="single"
              locale={enGB}
              selected={pendingDate}
              disabled={disabled}
              captionLayout="dropdown"
              numberOfMonths={1}
              weekStartsOn={0}
              showOutsideDays
              onSelect={(date) => {
                if (date) setPendingDate(date);
              }}
            />
            <label className="mt-2 grid gap-1 text-xs text-slate-400">
              <span className="text-[10px] uppercase tracking-[0.2em] text-slate-500">
                Time (EVE / UTC)
              </span>
              <div className="grid grid-cols-[1fr_auto_1fr_auto_1fr] items-center gap-2">
                <select
                  className="select select-bordered h-11 bg-base-300 font-mono text-base"
                  value={parseTimeParts(pendingTime).hour}
                  onChange={(event) => {
                    const { minute, second } = parseTimeParts(pendingTime);
                    setPendingTime(`${event.target.value}:${minute}:${second}`);
                  }}
                >
                  {hourOptions.map((hour) => (
                    <option key={hour} value={hour}>
                      {hour}
                    </option>
                  ))}
                </select>
                <span className="text-base font-mono text-slate-400">:</span>
                <select
                  className="select select-bordered h-11 bg-base-300 font-mono text-base"
                  value={parseTimeParts(pendingTime).minute}
                  onChange={(event) => {
                    const { hour, second } = parseTimeParts(pendingTime);
                    setPendingTime(`${hour}:${event.target.value}:${second}`);
                  }}
                >
                  {minuteSecondOptions.map((minute) => (
                    <option key={minute} value={minute}>
                      {minute}
                    </option>
                  ))}
                </select>
                <span className="text-base font-mono text-slate-400">:</span>
                <select
                  className="select select-bordered h-11 bg-base-300 font-mono text-base"
                  value={parseTimeParts(pendingTime).second}
                  onChange={(event) => {
                    const { hour, minute } = parseTimeParts(pendingTime);
                    setPendingTime(`${hour}:${minute}:${event.target.value}`);
                  }}
                >
                  {minuteSecondOptions.map((second) => (
                    <option key={second} value={second}>
                      {second}
                    </option>
                  ))}
                </select>
              </div>
            </label>
            <div className="mt-3 flex items-center justify-between text-xs text-slate-400">
              <span>Times are saved in EVE Time.</span>
              <div className="flex items-center gap-2">
                <button
                  className="btn btn-xs btn-outline"
                  type="button"
                  onClick={() => {
                    setDisplayValue(undefined);
                    onClear?.();
                    setOpen(false);
                  }}
                >
                  Clear
                </button>
                <button
                  className="btn btn-xs btn-outline"
                  type="button"
                  disabled={!pendingDate}
                  onClick={() => {
                    const applied = buildDateTime(pendingDate, pendingTime);
                    setDisplayValue(applied);
                    onApply(applied);
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
    </div>
  );
}
