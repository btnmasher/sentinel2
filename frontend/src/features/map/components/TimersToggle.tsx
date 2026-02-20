import { useMapStore } from "../store/mapStore";

type TimersToggleProps = {
  label?: string;
};

export default function TimersToggle({ label = "Timers" }: TimersToggleProps) {
  const displayTimers = useMapStore((s) => s.displayTimers !== false);
  const toggleTimers = useMapStore((s) => s.toggleTimers);

  return (
    <label className="flex items-center gap-2">
      <span>{label}</span>
      <input
        type="checkbox"
        className="toggle toggle-sm toggle-primary"
        checked={displayTimers}
        onChange={() => toggleTimers()}
      />
    </label>
  );
}
