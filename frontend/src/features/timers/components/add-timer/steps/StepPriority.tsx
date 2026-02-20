import type { ComponentType } from "react";
import type { StructureTone } from "../../../config/timerOptions";
import { useTimerFormStore } from "../../../store/useTimerFormStore";
import type { TimerSeverity } from "../../../types";

type Props = {
  severityOptions: ReadonlyArray<{
    value: TimerSeverity;
    label: string;
    tone: StructureTone;
    icon: ComponentType<{ className?: string }>;
  }>;
  severityToneClass: (tone: StructureTone, active: boolean) => string;
};

export default function StepPriority({
  severityOptions,
  severityToneClass,
}: Props) {
  const form = useTimerFormStore((s) => s.form);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  return (
    <div className="space-y-4 rounded-xl border border-slate-700/70 bg-base-300/20 p-4">
      <div className="space-y-2">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Priority
        </div>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {severityOptions.map((option) => (
            <button
              key={option.value}
              className={`btn btn-sm h-auto min-h-11 justify-center py-2 text-center leading-tight whitespace-normal ${severityToneClass(
                option.tone,
                form.severity === option.value,
              )}`}
              onClick={() =>
                updateForm((state) => ({
                  ...state,
                  severity: option.value,
                }))
              }
            >
              <option.icon className="h-3.5 w-3.5" />
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-2">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Details
        </div>
        <div className="grid w-full gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="text-xs uppercase tracking-wide text-slate-400">
              Title
            </span>
            <input
              className="input input-bordered w-full"
              value={form.title}
              onChange={(event) =>
                updateForm((state) => ({
                  ...state,
                  title: event.target.value,
                }))
              }
              placeholder="Structure or operation"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-xs uppercase tracking-wide text-slate-400">
              Notes
            </span>
            <textarea
              className="textarea textarea-bordered h-24 w-full"
              value={form.notes}
              onChange={(event) =>
                updateForm((state) => ({
                  ...state,
                  notes: event.target.value,
                }))
              }
            />
          </div>
        </div>
      </div>
    </div>
  );
}
