import { useMapStore } from "../store/mapStore";

type JumpbridgesToggleProps = {
  label?: string;
};

export default function JumpbridgesToggle({
  label = "Jumpbridges",
}: JumpbridgesToggleProps) {
  const displayJumpbridges = useMapStore((s) => s.displayJumpbridges);
  const toggleJumpbridges = useMapStore((s) => s.toggleJumpbridges);

  return (
    <label className="flex items-center gap-2">
      <span>{label}</span>
      <input
        type="checkbox"
        className="toggle toggle-sm toggle-primary"
        checked={displayJumpbridges}
        onChange={() => toggleJumpbridges()}
      />
    </label>
  );
}
