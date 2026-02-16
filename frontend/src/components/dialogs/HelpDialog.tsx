import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import {
  defineUIDialogModal,
  useUIStore,
  UI_DIALOG,
} from "@/app/store/uiStore";
import { useSettingsStore } from "@/app/store/settingsStore";
import { INTEL_THREAT_STAGE_COLORS } from "@/features/map";

const MAP_BORDER_COLORS = {
  route: "#22c55e",
  undocked: "#38bdf8",
  docked: "#94a3b8",
} as const;

function HelpDialogBody() {
  const threatTimings = useSettingsStore((s) => s.settings.intel.threatTimings);

  return (
    <>
      <div>
        <h4 className="font-semibold">Basic controls</h4>
        <ul className="list-disc ml-4">
          <li>click and drag - move around the map</li>
          <li>
            mouse scroll wheel - zoom in/out (do not hold control/command)
          </li>
        </ul>
      </div>
      <div>
        <h4 className="font-semibold">Intel Threat Colors</h4>
        <p className="text-sm text-slate-400">
          A reported system starts at flashing, then cascades through red,
          orange, yellow, and green as time expires. These values are your live
          settings and each stage starts after the previous one ends.
        </p>
        <div className="mt-2 space-y-2">
          {[
            {
              label: "Flashing",
              color: INTEL_THREAT_STAGE_COLORS.flash,
              seconds: threatTimings.flash,
              flashing: true,
            },
            {
              label: "Red",
              color: INTEL_THREAT_STAGE_COLORS.red,
              seconds: threatTimings.red,
            },
            {
              label: "Orange",
              color: INTEL_THREAT_STAGE_COLORS.orange,
              seconds: threatTimings.orange,
            },
            {
              label: "Yellow",
              color: INTEL_THREAT_STAGE_COLORS.yellow,
              seconds: threatTimings.yellow,
            },
            {
              label: "Green",
              color: INTEL_THREAT_STAGE_COLORS.green,
              seconds: threatTimings.green,
            },
          ].map((stage) => (
            <div key={stage.label} className="flex items-center gap-2">
              <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden>
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
              <span>
                {stage.label}: {stage.seconds}s
              </span>
            </div>
          ))}
        </div>
      </div>
      <div>
        <h4 className="font-semibold">Map Legend</h4>
        <div className="flex items-center gap-2">
          <svg width="32" height="10">
            <line
              x1="4"
              y1="5"
              x2="28"
              y2="5"
              className="map-gate constellation"
            />
          </svg>
          Constellation gate
        </div>
        <div className="flex items-center gap-2">
          <svg width="32" height="10">
            <line x1="4" y1="5" x2="28" y2="5" className="map-gate region" />
          </svg>
          Region gate
        </div>
        <div className="flex items-center gap-2">
          <svg width="32" height="10">
            <line
              x1="4"
              y1="5"
              x2="28"
              y2="5"
              className="map-gate jumpbridge"
            />
          </svg>
          Jumpbridge
        </div>
        <div className="mt-2 text-sm text-slate-300">
          Shift + click systems to filter logs. Filtered systems also highlight
          their map label color.
        </div>
        <div className="mt-2 flex items-center gap-2">
          <svg width="104" height="60" viewBox="0 0 104 60" aria-hidden>
            <rect
              x="4"
              y="4"
              width="32"
              height="32"
              rx="6"
              ry="6"
              fill={INTEL_THREAT_STAGE_COLORS.normal}
              stroke={INTEL_THREAT_STAGE_COLORS.normal}
              strokeWidth="1"
            />
            <text
              x="20"
              y="56"
              textAnchor="middle"
              fontSize="14"
              fill="#66bb6a"
            >
              Jita
            </text>
          </svg>
        </div>
      </div>
      <div>
        <h4 className="font-semibold">System Filters and Jump Ranges</h4>
        <ul className="list-disc ml-4 text-sm text-slate-300">
          <li>
            Shift + left click on a system toggles that system in Intel filters.
          </li>
          <li>
            Left click on a system selects it for jump ranges (primary and
            secondary).
          </li>
          <li>
            Use right click on the map, then Jumprange to enable or clear range
            overlays.
          </li>
        </ul>
      </div>
      <div>
        <h4 className="font-semibold">System Border States</h4>
        <p className="text-sm text-slate-400">
          Border color priority is route head, then undocked, then docked. Route
          waypoints use a steady route border.
        </p>
        <div className="mt-2 space-y-2 text-sm">
          <div className="flex items-center gap-2">
            <svg
              width="24"
              height="24"
              viewBox="-4 -4 28 28"
              aria-hidden
              style={{ overflow: "visible" }}
            >
              <rect
                x="2"
                y="2"
                width="16"
                height="16"
                rx="3"
                ry="3"
                fill={INTEL_THREAT_STAGE_COLORS.normal}
                stroke={INTEL_THREAT_STAGE_COLORS.normal}
                strokeWidth="1.2"
              />
              <rect
                className="map-system-border map-system-border-pulse"
                x="1"
                y="1"
                width="18"
                height="18"
                rx="4"
                ry="4"
                fill="none"
                stroke={MAP_BORDER_COLORS.route}
                strokeWidth="2"
                style={{
                  filter: `drop-shadow(0 0 3px ${MAP_BORDER_COLORS.route}) drop-shadow(0 0 4px ${MAP_BORDER_COLORS.route})`,
                }}
              />
            </svg>
            <span>Route head (pulsing green)</span>
          </div>
          <div className="flex items-center gap-2">
            <svg
              width="24"
              height="24"
              viewBox="-4 -4 28 28"
              aria-hidden
              style={{ overflow: "visible" }}
            >
              <rect
                x="2"
                y="2"
                width="16"
                height="16"
                rx="3"
                ry="3"
                fill={INTEL_THREAT_STAGE_COLORS.normal}
                stroke={INTEL_THREAT_STAGE_COLORS.normal}
                strokeWidth="1.2"
              />
              <rect
                className="map-system-border"
                x="1"
                y="1"
                width="18"
                height="18"
                rx="4"
                ry="4"
                fill="none"
                stroke={MAP_BORDER_COLORS.route}
                strokeWidth="2"
                style={{
                  filter: `drop-shadow(0 0 3px ${MAP_BORDER_COLORS.route}) drop-shadow(0 0 4px ${MAP_BORDER_COLORS.route})`,
                }}
              />
            </svg>
            <span>Route waypoint (steady green)</span>
          </div>
          <div className="flex items-center gap-2">
            <svg
              width="24"
              height="24"
              viewBox="-4 -4 28 28"
              aria-hidden
              style={{ overflow: "visible" }}
            >
              <rect
                x="2"
                y="2"
                width="16"
                height="16"
                rx="3"
                ry="3"
                fill={INTEL_THREAT_STAGE_COLORS.normal}
                stroke={INTEL_THREAT_STAGE_COLORS.normal}
                strokeWidth="1.2"
              />
              <rect
                className="map-system-border map-system-border-pulse"
                x="1"
                y="1"
                width="18"
                height="18"
                rx="4"
                ry="4"
                fill="none"
                stroke={MAP_BORDER_COLORS.undocked}
                strokeWidth="2"
                style={{
                  filter: `drop-shadow(0 0 3px ${MAP_BORDER_COLORS.undocked}) drop-shadow(0 0 4px ${MAP_BORDER_COLORS.undocked})`,
                }}
              />
            </svg>
            <span>Character undocked (pulsing blue)</span>
          </div>
          <div className="flex items-center gap-2">
            <svg
              width="24"
              height="24"
              viewBox="-4 -4 28 28"
              aria-hidden
              style={{ overflow: "visible" }}
            >
              <rect
                x="2"
                y="2"
                width="16"
                height="16"
                rx="3"
                ry="3"
                fill={INTEL_THREAT_STAGE_COLORS.normal}
                stroke={INTEL_THREAT_STAGE_COLORS.normal}
                strokeWidth="1.2"
              />
              <rect
                className="map-system-border map-system-border-pulse"
                x="1"
                y="1"
                width="18"
                height="18"
                rx="4"
                ry="4"
                fill="none"
                stroke={MAP_BORDER_COLORS.docked}
                strokeWidth="2"
                style={{
                  filter: `drop-shadow(0 0 3px ${MAP_BORDER_COLORS.docked}) drop-shadow(0 0 4px ${MAP_BORDER_COLORS.docked})`,
                }}
              />
            </svg>
            <span>Character docked (pulsing slate)</span>
          </div>
        </div>
      </div>
      <div>
        <h4 className="font-semibold">Context menus</h4>
        <ul className="list-disc ml-4">
          <li>Right click on the map - set jumprange settings, clear route</li>
          <li>
            Right click on a system - show jumpranges, set route, show
            zkill/dotlan
          </li>
          <li>
            Right click on selected text in intel logs - search player names
          </li>
        </ul>
      </div>
      <div>
        <h4 className="font-semibold">Saved settings</h4>
        <p>
          Map regions, intel log filters, and settings are stored in your
          browser.
        </p>
      </div>
    </>
  );
}

function HelpDialogActions() {
  const { close } = useModalBody();
  const setIntroduction = useSettingsStore((s) => s.setIntroduction);

  return (
    <>
      <button
        className="btn btn-sm btn-outline"
        onClick={() => {
          setIntroduction(true);
          close();
        }}
      >
        Show intro
      </button>
      <button className="btn btn-sm btn-outline" onClick={() => close()}>
        Close
      </button>
    </>
  );
}

export const AppModalHelp = defineUIDialogModal({
  key: UI_DIALOG.Help,
  useOpen: () => useUIStore((s) => s.dialogs[UI_DIALOG.Help]),
  build: () => ({
    title: "Help",
    body: <HelpDialogBody />,
    actions: <HelpDialogActions />,
  }),
});

export default function HelpDialog() {
  useModal(AppModalHelp);

  return null;
}
