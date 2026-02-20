import {
  timerFieldsFromContextSelection,
  type ContextOption,
  type ContextTone,
} from "../../../config/timerOptions";
import { useTimerFormStore } from "../../../store/useTimerFormStore";
import { TimerKind, TimerStructureType } from "../../../types";

type Props = {
  contextOptions: ReadonlyArray<ContextOption>;
  contextToneClass: (tone: ContextTone, active: boolean) => string;
};

export default function StepContext({
  contextOptions,
  contextToneClass,
}: Props) {
  const form = useTimerFormStore((s) => s.form);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  const showContextNote =
    form.timer_kind === TimerKind.Extraction ||
    form.timer_kind === TimerKind.Custom;
  const showSkyhookFullness =
    form.structure_type === TimerStructureType.OrbitalSkyhook &&
    form.timer_kind === TimerKind.Extraction;

  const setSkyhookFullness = (raw: string) => {
    const digitsOnly = raw.replace(/\D+/g, "");
    if (digitsOnly === "") {
      updateForm((state) => ({
        ...state,
        skyhook_fullness_pct: "",
      }));
      return;
    }
    const clamped = Math.min(100, Math.max(0, Number(digitsOnly)));
    updateForm((state) => ({
      ...state,
      skyhook_fullness_pct: String(clamped),
    }));
  };

  return (
    <div className="space-y-4 rounded-xl border border-slate-700/70 bg-base-300/20 p-4">
      <div className="space-y-3">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Timer Type
        </div>
        <div className="grid grid-cols-2 gap-2">
          {contextOptions.map((option) => {
            const active = form.context_selection === option.value;
            return (
              <button
                key={option.value}
                className={`btn btn-sm h-auto min-h-11 justify-start py-2 text-left leading-tight whitespace-normal ${contextToneClass(option.tone, active)}`}
                onClick={() => {
                  const next = timerFieldsFromContextSelection(option.value);
                  updateForm((state) => ({
                    ...state,
                    context_selection: option.value,
                    timer_kind: next.timerKind,
                    stage_label: next.stageLabel,
                  }));
                }}
                type="button"
              >
                {option.label}
              </button>
            );
          })}
        </div>

        {showContextNote || showSkyhookFullness ? (
          <div className="grid grid-cols-[9.5rem_minmax(0,1fr)] items-center gap-x-3 gap-y-4 pt-1">
            {showContextNote ? (
              <>
                <div className="text-xs uppercase tracking-wide text-slate-400">
                  {form.timer_kind === TimerKind.Extraction
                    ? "Extraction note"
                    : "Custom note"}
                </div>
                <input
                  className="input input-bordered w-full"
                  value={form.timer_kind_note}
                  onChange={(event) =>
                    updateForm((state) => ({
                      ...state,
                      timer_kind_note: event.target.value,
                    }))
                  }
                  placeholder="Short context note"
                />
              </>
            ) : null}

            {showSkyhookFullness ? (
              <>
                <div className="text-xs uppercase tracking-wide text-slate-400">
                  Skyhook fullness %
                </div>
                <input
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  maxLength={3}
                  className="input input-bordered w-24"
                  value={form.skyhook_fullness_pct}
                  onChange={(event) => setSkyhookFullness(event.target.value)}
                  onPaste={(event) => {
                    event.preventDefault();
                    const pasted = event.clipboardData.getData("text");
                    setSkyhookFullness(pasted);
                  }}
                  placeholder="0-100"
                />
              </>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}
