import { Moon, Sun } from "lucide-react";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useShallow } from "zustand/shallow";

type ThemeToggleProps = {
  className?: string;
  iconClassName?: string;
  title?: string;
};

export default function ThemeToggle({
  className = "btn btn-xs btn-ghost btn-square",
  iconClassName = "h-4 w-4",
  title = "Toggle theme",
}: ThemeToggleProps) {
  const { theme, setTheme } = useSettingsStore(
    useShallow((s) => ({
      theme: s.settings.theme,
      setTheme: s.setTheme,
    })),
  );
  const isLight = theme === "sentinel-light";

  return (
    <button
      className={className}
      onClick={() => setTheme(isLight ? "sentinel" : "sentinel-light")}
      aria-label={title}
      title={title}
      type="button"
    >
      {isLight ? (
        <Moon className={iconClassName} />
      ) : (
        <Sun className={iconClassName} />
      )}
    </button>
  );
}
