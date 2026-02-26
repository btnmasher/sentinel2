import { useSettingsStore } from "@/app/store/settingsStore";

type ZKillToggleProps = {
  label?: string;
};

export default function ZKillToggle({ label = "zKill" }: ZKillToggleProps) {
  const enabled = useSettingsStore((s) => s.settings.intel.zkillFeedEnabled);
  const apply = useSettingsStore((s) => s.apply);

  return (
    <label className="flex items-center gap-2">
      <span>{label}</span>
      <input
        type="checkbox"
        className="toggle toggle-sm toggle-primary"
        checked={enabled}
        onChange={() => apply("intel", "zkillFeedEnabled", !enabled)}
      />
    </label>
  );
}
