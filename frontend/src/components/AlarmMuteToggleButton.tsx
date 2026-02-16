import { Volume2, VolumeX } from "lucide-react";

type AlarmMuteToggleButtonProps = {
  muted: boolean;
  onToggle: () => void;
  className?: string;
};

export default function AlarmMuteToggleButton({
  muted,
  onToggle,
  className,
}: AlarmMuteToggleButtonProps) {
  return (
    <button
      className={`btn btn-xs btn-square ${
        muted ? "btn-error btn-outline" : "btn-ghost"
      } ${className ?? ""}`.trim()}
      onClick={onToggle}
      aria-label={muted ? "Unmute intel alarm" : "Mute intel alarm"}
      title={muted ? "Unmute intel alarm" : "Mute intel alarm"}
    >
      {muted ? <VolumeX className="h-4 w-4" /> : <Volume2 className="h-4 w-4" />}
    </button>
  );
}
