import { useEffect, useMemo, useRef, useState } from "react";
import { useShallow } from "zustand/react/shallow";
import { useSettingsStore } from "@/app/store/settingsStore";
import SelectionDropdown from "@/components/SelectionDropdown";

const ALARM_SOUNDS = ["woop", "school", "grocery", "blip", "progod"];

export default function Settings() {
  const { settings, toggle, apply, setTheme, reset } = useSettingsStore(
    useShallow((s) => ({
      settings: s.settings,
      toggle: s.toggle,
      apply: s.apply,
      setTheme: s.setTheme,
      reset: s.reset,
    })),
  );
  const [confirmReset, setConfirmReset] = useState(false);
  const [confirmClearData, setConfirmClearData] = useState(false);
  const previewRef = useRef<HTMLAudioElement | null>(null);
  const prevSoundRef = useRef(settings.alarm.sound);
  const prevVolumeRef = useRef(settings.alarm.volume);

  const themeOptions = useMemo(
    () => [
      { id: "sentinel", label: "Sentinel Dark" },
      { id: "sentinel-light", label: "Sentinel Light" },
    ],
    [],
  );

  const soundOptions = useMemo(
    () => ALARM_SOUNDS.map((sound) => ({ id: sound, label: sound })),
    [],
  );

  const resetSettings = () => {
    reset();
    window.location.reload();
  };

  const clearSavedData = () => {
    localStorage.clear();
    window.location.reload();
  };

  useEffect(() => {
    if (!settings.alarm.enabled) {
      prevSoundRef.current = settings.alarm.sound;
      prevVolumeRef.current = settings.alarm.volume;
      return;
    }
    const soundChanged = prevSoundRef.current !== settings.alarm.sound;
    const volumeChanged = prevVolumeRef.current !== settings.alarm.volume;
    if (!soundChanged && !volumeChanged) return;
    prevSoundRef.current = settings.alarm.sound;
    prevVolumeRef.current = settings.alarm.volume;
    const audio = previewRef.current;
    if (!audio) return;
    audio.volume = settings.alarm.volume / 100;
    audio.currentTime = 0;
    void audio.play().catch(() => undefined);
  }, [settings.alarm.enabled, settings.alarm.sound, settings.alarm.volume]);

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="card bg-base-200/70 border border-slate-800">
        <div className="card-body space-y-4">
          <h3 className="font-display text-lg">Appearance</h3>
          <label className="flex items-center justify-between">
            <span>Theme</span>
            <SelectionDropdown
              items={themeOptions}
              selected={[settings.theme]}
              onChange={(next) =>
                setTheme((next[0] ?? "sentinel") as typeof settings.theme)
              }
              label="Theme"
              buttonClassName="min-w-[160px]"
            />
          </label>
        </div>
      </div>

      <div className="card bg-base-200/70 border border-slate-800">
        <div className="card-body space-y-4">
          <h3 className="font-display text-lg">Map</h3>
          <label className="flex items-center justify-between">
            <span>Invert Zoom</span>
            <input
              className="toggle toggle-primary"
              type="checkbox"
              checked={settings.map.invertZoom}
              onChange={() => toggle("map", "invertZoom")}
            />
          </label>
          <label className="flex items-center justify-between">
            <span>Always Show System Icons</span>
            <input
              className="toggle toggle-primary"
              type="checkbox"
              checked={settings.map.alwaysShowSystems}
              onChange={() => toggle("map", "alwaysShowSystems")}
            />
          </label>
        </div>
      </div>

      <div className="card bg-base-200/70 border border-slate-800">
        <div className="card-body space-y-4">
          <h3 className="font-display text-lg">Alarm</h3>
          <label className="flex items-center justify-between">
            <span>Enable Alarm</span>
            <input
              className="toggle toggle-primary"
              type="checkbox"
              checked={settings.alarm.enabled}
              onChange={() => toggle("alarm", "enabled")}
            />
          </label>
          <div>
            <label className="label text-xs mb-1 block">Volume</label>
            <input
              type="range"
              min={0}
              max={100}
              value={settings.alarm.volume}
              onChange={(e) => apply("alarm", "volume", Number(e.target.value))}
              className="range range-xs"
              disabled={!settings.alarm.enabled}
            />
          </div>
          <div>
            <label className="label text-xs">Alarm tone</label>
            <SelectionDropdown
              items={soundOptions}
              selected={[settings.alarm.sound]}
              onChange={(next) => apply("alarm", "sound", next[0] ?? "")}
              label="Alarm tone"
              disabled={!settings.alarm.enabled}
              buttonClassName="min-w-[160px]"
            />
          </div>
        </div>
      </div>

      <div className="card bg-base-200/70 border border-slate-800">
        <div className="card-body space-y-4">
          <h3 className="font-display text-lg">Reset</h3>
          <div className="flex flex-wrap gap-2">
            <button
              className="btn btn-error btn-outline btn-sm"
              onClick={() => setConfirmReset(true)}
            >
              Reset all saved settings
            </button>
            <button
              className="btn btn-outline btn-sm"
              onClick={() => setConfirmClearData(true)}
            >
              Clear browser saved data
            </button>
          </div>
        </div>
      </div>

      {confirmReset && (
        <div className="modal modal-open">
          <div className="modal-box bg-base-200 border border-slate-700">
            <h3 className="font-display text-lg">Are you sure?</h3>
            <div className="modal-action">
              <button
                className="btn btn-error btn-outline btn-sm"
                onClick={resetSettings}
              >
                Yes
              </button>
              <button
                className="btn btn-sm btn-outline"
                onClick={() => setConfirmReset(false)}
              >
                No
              </button>
            </div>
          </div>
        </div>
      )}

      {confirmClearData && (
        <div className="modal modal-open">
          <div className="modal-box bg-base-200 border border-slate-700">
            <h3 className="font-display text-lg">Clear saved data?</h3>
            <p className="text-sm text-slate-400">
              This clears all stored data in your browser for Sentinel.
            </p>
            <div className="modal-action">
              <button
                className="btn btn-error btn-outline btn-sm"
                onClick={clearSavedData}
              >
                Yes
              </button>
              <button
                className="btn btn-sm btn-outline"
                onClick={() => setConfirmClearData(false)}
              >
                No
              </button>
            </div>
          </div>
        </div>
      )}

      <audio ref={previewRef} src={`/audio/${settings.alarm.sound}.mp3`} />
    </div>
  );
}
