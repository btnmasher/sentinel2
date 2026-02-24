type Props = {
  showInactive: boolean;
  setShowInactive: (value: boolean) => void;
  use24Hour: boolean;
  setUse24Hour: (value: boolean) => void;
  onAddTimer: () => void;
};

export default function TimerBoardHeaderActions({
  showInactive,
  setShowInactive,
  use24Hour,
  setUse24Hour,
  onAddTimer,
}: Props) {
  return (
    <div className="flex items-center gap-3">
      <label className="label cursor-pointer gap-2 text-xs">
        <span className="label-text text-xs">Show inactive</span>
        <input
          type="checkbox"
          className="toggle toggle-xs toggle-primary"
          checked={showInactive}
          onChange={(event) => setShowInactive(event.target.checked)}
        />
      </label>
      <div className="join">
        <button
          className={`btn btn-sm h-9 min-h-9 join-item rounded-r-none! rounded-l-md! ${!use24Hour ? "btn-primary" : "btn-outline"}`}
          onClick={() => setUse24Hour(false)}
        >
          12h
        </button>
        <button
          className={`btn btn-sm h-9 min-h-9 join-item rounded-l-none! rounded-r-md! ${use24Hour ? "btn-primary" : "btn-outline"}`}
          onClick={() => setUse24Hour(true)}
        >
          24h
        </button>
      </div>
      <button
        className="btn btn-success btn-sm h-9 min-h-9"
        onClick={onAddTimer}
      >
        <Plus className="h-4 w-4" /> Add Timer
      </button>
    </div>
  );
}
import { Plus } from "lucide-react";
