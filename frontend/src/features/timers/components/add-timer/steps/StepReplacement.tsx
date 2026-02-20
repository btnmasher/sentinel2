import type { ComponentType } from "react";
import type { ReplacementTone } from "../../../config/timerOptions";
import { useTimerFormStore } from "../../../store/useTimerFormStore";
import type { TimerReplacementAction } from "../../../types";

type Props = {
  replacementOptions: ReadonlyArray<{
    value: TimerReplacementAction;
    label: string;
    icon: ComponentType<{ className?: string }>;
    tone: ReplacementTone;
  }>;
  replacementToneClass: (tone: ReplacementTone, active: boolean) => string;
};

export default function StepReplacement({
  replacementOptions,
  replacementToneClass,
}: Props) {
  const form = useTimerFormStore((s) => s.form);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  return (
    <div className="space-y-4 rounded-xl border border-slate-700/70 bg-base-300/20 p-4">
      <div className="space-y-2">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Replacement
        </div>
        <div className="grid grid-cols-2 gap-2">
          {replacementOptions.map((option) => {
            const Icon = option.icon;
            const active = form.replacement_action === option.value;
            return (
              <button
                key={option.value}
                className={`btn btn-sm h-auto min-h-11 justify-start py-2 text-left leading-tight whitespace-normal ${replacementToneClass(option.tone, active)}`}
                onClick={() =>
                  updateForm((state) => ({
                    ...state,
                    replacement_action: option.value,
                  }))
                }
              >
                <Icon className="h-4 w-4" /> {option.label}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
