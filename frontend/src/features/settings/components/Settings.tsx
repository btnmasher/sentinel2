import { useEffect, useMemo, useRef, useState } from "react";
import { useShallow } from "zustand/react/shallow";
import useModal from "@/app/hooks/useModal";
import { useSettingsStore } from "@/app/store/settingsStore";
import SelectionDropdown from "@/components/SelectionDropdown";
import { INTEL_THREAT_STAGE_COLORS } from "@/features/map";

const ALARM_SOUNDS = ["woop", "school", "grocery", "blip", "progod"];
const SETTINGS_MODAL = {
  Reset: "reset",
  ClearData: "clearData",
} as const;
type SettingsModalKey = (typeof SETTINGS_MODAL)[keyof typeof SETTINGS_MODAL];
const THREAT_STAGE_CONFIG = [
  {
    key: "flash",
    label: "Flashing",
    color: INTEL_THREAT_STAGE_COLORS.flash,
    flashing: true,
  },
  { key: "red", label: "Red", color: INTEL_THREAT_STAGE_COLORS.red },
  { key: "orange", label: "Orange", color: INTEL_THREAT_STAGE_COLORS.orange },
  { key: "yellow", label: "Yellow", color: INTEL_THREAT_STAGE_COLORS.yellow },
  { key: "green", label: "Green", color: INTEL_THREAT_STAGE_COLORS.green },
] as const;

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
  const previewRef = useRef<HTMLAudioElement | null>(null);
  const prevSoundRef = useRef(settings.alarm.sound);
  const prevVolumeRef = useRef(settings.alarm.volume);
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [showClearDataConfirm, setShowClearDataConfirm] = useState(false);
  const setSettingsModal = (modal: SettingsModalKey, open: boolean) => {
    if (modal === SETTINGS_MODAL.Reset) {
      setShowResetConfirm(open);
      if (open) setShowClearDataConfirm(false);
      return;
    }
    setShowClearDataConfirm(open);
    if (open) setShowResetConfirm(false);
  };

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

  useModal({
    open: showResetConfirm,
    modalKey: SETTINGS_MODAL.Reset,
    setOpenByKey: setSettingsModal,
    build: (close) => ({
      title: "Are you sure?",
      body: (
        <div className="modal-action">
          <button
            className="btn btn-error btn-outline btn-sm"
            onClick={resetSettings}
          >
            Yes
          </button>
          <button className="btn btn-sm btn-outline" onClick={() => close()}>
            No
          </button>
        </div>
      ),
    }),
  });

  useModal({
    open: showClearDataConfirm,
    modalKey: SETTINGS_MODAL.ClearData,
    setOpenByKey: setSettingsModal,
    build: (close) => ({
      title: "Clear saved data?",
      body: (
        <>
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
            <button className="btn btn-sm btn-outline" onClick={() => close()}>
              No
            </button>
          </div>
        </>
      ),
    }),
  });

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

  const setThreatTiming = (
    stage: keyof typeof settings.intel.threatTimings,
    value: number,
  ) => {
    const nextValue = Math.max(0, Math.min(900, Math.round(value / 5) * 5));
    apply("intel", "threatTimings", {
      ...settings.intel.threatTimings,
      [stage]: nextValue,
    });
  };

  return (
    <div className="grid gap-6 lg:grid-cols-2 items-start">
      <div className="grid gap-6">
        <div className="card bg-base-200/70 border border-slate-800">
          <div className="card-body space-y-4">
            <h3 className="font-display text-lg">Map & Intel</h3>
            <div className="rounded-md bg-base-300/30 px-3 py-2">
              <label className="flex items-center justify-between">
                <span>Invert Zoom</span>
                <input
                  className="toggle toggle-primary"
                  type="checkbox"
                  checked={settings.map.invertZoom}
                  onChange={() => toggle("map", "invertZoom")}
                />
              </label>
            </div>
            <div className="rounded-md bg-base-300/15 px-3 py-2">
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
            <div className="rounded-md bg-base-300/30 px-3 py-2 space-y-3">
              <div className="text-xs text-slate-400">
                Threat stage timings cascade in order: flashing -&gt; red -&gt;
                orange -&gt; yellow -&gt; green.
              </div>
              {THREAT_STAGE_CONFIG.map((stage) => (
                <div key={stage.key} className="rounded-md bg-base-300/20 p-2">
                  <div className="flex items-center justify-between gap-3">
                    <div className="flex items-center gap-2 min-w-0">
                      <svg
                        width="20"
                        height="20"
                        viewBox="0 0 20 20"
                        aria-hidden
                      >
                        <rect
                          x="1"
                          y="1"
                          width="18"
                          height="18"
                          rx="4"
                          ry="4"
                          fill={stage.color}
                          stroke={stage.color}
                          strokeWidth="1.5"
                          className={stage.flashing ? "map-system-alert" : ""}
                        />
                      </svg>
                      <span className="text-sm">{stage.label}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <input
                        type="number"
                        min={0}
                        max={900}
                        step={5}
                        value={settings.intel.threatTimings[stage.key]}
                        onChange={(e) =>
                          setThreatTiming(stage.key, Number(e.target.value))
                        }
                        className="input input-xs w-20"
                      />
                      <span className="text-xs text-slate-400">sec</span>
                    </div>
                  </div>
                  <div className="mt-2">
                    <input
                      type="range"
                      min={0}
                      max={900}
                      step={5}
                      value={settings.intel.threatTimings[stage.key]}
                      onChange={(e) =>
                        setThreatTiming(stage.key, Number(e.target.value))
                      }
                      className="range range-xs"
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="card bg-base-200/70 border border-slate-800">
          <div className="card-body space-y-4">
            <h3 className="font-display text-lg">Alarm</h3>
            <div className="rounded-md bg-base-300/30 px-3 py-2">
              <label className="flex items-center justify-between">
                <span>Enable Alarm</span>
                <input
                  className="toggle toggle-primary"
                  type="checkbox"
                  checked={settings.alarm.enabled}
                  onChange={() => toggle("alarm", "enabled")}
                />
              </label>
            </div>
            <div className="rounded-md bg-base-300/15 px-3 py-2">
              <label className="label text-xs mb-1 block">Volume</label>
              <input
                type="range"
                min={0}
                max={100}
                value={settings.alarm.volume}
                onChange={(e) =>
                  apply("alarm", "volume", Number(e.target.value))
                }
                className="range range-xs"
                disabled={!settings.alarm.enabled}
              />
            </div>
            <div className="rounded-md bg-base-300/30 px-3 py-2">
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
      </div>

      <div className="grid gap-6">
        <div className="card bg-base-200/70 border border-slate-800">
          <div className="card-body space-y-4">
            <h3 className="font-display text-lg">Appearance</h3>
            <div className="rounded-md bg-base-300/30 px-3 py-2">
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
        </div>

        <div className="card bg-base-200/70 border border-slate-800">
          <div className="card-body space-y-4">
            <h3 className="font-display text-lg">Reset</h3>
            <div className="flex flex-wrap gap-2">
              <button
                className="btn btn-error btn-outline btn-sm"
                onClick={() => setSettingsModal(SETTINGS_MODAL.Reset, true)}
              >
                Reset all saved settings
              </button>
              <button
                className="btn btn-outline btn-sm"
                onClick={() => setSettingsModal(SETTINGS_MODAL.ClearData, true)}
              >
                Clear browser saved data
              </button>
            </div>
          </div>
        </div>
      </div>

      <audio ref={previewRef} src={`/audio/${settings.alarm.sound}.mp3`} />
    </div>
  );
}
