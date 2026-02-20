import { WandSparkles } from "lucide-react";
import DateTimePicker from "@/components/DateTimePicker";
import { useTimerFormStore } from "../../../store/useTimerFormStore";

type Props = {
  selectedExpiresAt: Date | null;
  parsePastedText: () => void;
  eveDisplayDateToISO: (value: Date) => string;
};

export default function StepTime({
  selectedExpiresAt,
  parsePastedText,
  eveDisplayDateToISO,
}: Props) {
  const form = useTimerFormStore((s) => s.form);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  return (
    <div className="space-y-4 rounded-xl border border-slate-700/70 bg-base-300/20 p-4">
      <label className="form-control">
        <div className="label py-1">
          <span className="label-text text-xs uppercase tracking-wide">
            Paste text (optional)
          </span>
        </div>
        <textarea
          className="textarea textarea-bordered h-24 w-full"
          value={form.raw_text}
          onChange={(event) =>
            updateForm((state) => ({
              ...state,
              raw_text: event.target.value,
            }))
          }
          placeholder="Reinforced until..."
        />
      </label>
      <div className="mt-3 flex w-full justify-end">
        <button className="btn btn-sm btn-outline" onClick={parsePastedText}>
          <WandSparkles className="h-4 w-4" /> Parse
        </button>
      </div>

      <label className="form-control gap-2">
        <div className="label py-1">
          <span className="label-text text-xs uppercase tracking-wide">
            Date + Time (EVE Time)
          </span>
        </div>
        <DateTimePicker
          value={selectedExpiresAt ?? undefined}
          disablePast
          buttonClassName="input input-bordered w-full justify-between"
          onApply={(value) =>
            updateForm((state) => ({
              ...state,
              expires_at: value ? eveDisplayDateToISO(value) : "",
            }))
          }
          onClear={() =>
            updateForm((state) => ({
              ...state,
              expires_at: "",
            }))
          }
        />
      </label>
    </div>
  );
}
