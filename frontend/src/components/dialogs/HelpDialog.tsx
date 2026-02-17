import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  ChevronDown,
  ChevronUp,
  Compass,
  Filter,
  LayoutGrid,
  LocateFixed,
  MenuIcon,
  Map,
  Menu,
  Maximize2,
  MousePointer2,
  Save,
  ScrollText,
  Upload,
  UserRound,
  Volume2,
  VolumeX,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import useModal from "@/app/hooks/useModal";
import ThemeToggle from "@/components/ThemeToggle";
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

type HelpSectionProps = {
  title: string;
  icon?: ReactNode;
  children: ReactNode;
};

function HelpSection({ title, icon, children }: HelpSectionProps) {
  return (
    <section className="mb-3 inline-block w-full break-inside-avoid rounded-xl border border-slate-700/80 bg-base-200/70 p-4">
      <h3 className="mb-3 inline-flex items-center gap-2 font-display leading-none">
        {icon}
        <span>{title}</span>
      </h3>
      <div className="space-y-2">{children}</div>
    </section>
  );
}

function HelpDialogBody() {
  const threatTimings = useSettingsStore((s) => s.settings.intel.threatTimings);
  const introMode = useSettingsStore((s) => s.settings.introduction);
  const sectionTextClass = "text-sm text-slate-300";

  return (
    <>
      {introMode && (
        <div className="mb-3 rounded-lg border border-primary/35 bg-primary/10 px-3 py-2 text-sm text-primary">
          Welcome to Sentinel. This quick guide covers the core controls and how
          to read map/intel status at a glance.
        </div>
      )}
      <div className="columns-1 gap-3 xl:columns-2">
        <HelpSection
          title="Basic controls"
          icon={<MousePointer2 className="h-4 w-4 text-slate-400" />}
        >
          <ul className="ml-4 list-disc space-y-1">
            <li>
              <span className="font-semibold text-slate-100">
                Map Movement:
              </span>{" "}
              click and drag to move around the map.
            </li>
            <li>
              <span className="font-semibold text-slate-100">Zoom:</span> mouse
              scroll wheel zooms in/out (do not hold control/command).
            </li>
          </ul>
        </HelpSection>
        <HelpSection
          title="Navigation bar"
          icon={<Compass className="h-4 w-4 text-slate-400" />}
        >
          <p className={sectionTextClass}>
            The top bar combines map controls and live status indicators.
          </p>
          <div className="space-y-4">
            <div>
              <p className="text-sm font-medium text-slate-100">
                Site navigation menu
              </p>
              <p className="mt-1 text-xs text-slate-300">
                In full-page map view, use the hamburger button to open app
                navigation.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <span className="btn btn-sm btn-square btn-primary btn-outline pointer-events-none">
                  <MenuIcon className="h-4 w-4" />
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">
                Region controls
              </p>
              <ul className="ml-4 mt-1 list-disc space-y-1 text-sm text-slate-300">
                <li>
                  <span className="font-semibold text-slate-100">Search:</span>{" "}
                  find systems quickly and center them in the map.
                </li>
                <li>
                  <span className="font-semibold text-slate-100">
                    Region select:
                  </span>{" "}
                  choose one or more regions to load.
                </li>
                <li>
                  <span className="font-semibold text-slate-100">
                    Layout select:
                  </span>{" "}
                  switch map layout style.
                </li>
                <li>
                  <span className="font-semibold text-slate-100">
                    Jumpbridge toggle:
                  </span>{" "}
                  show or hide jumpbridge lines.
                </li>
              </ul>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">
                Zoom and center controls
              </p>
              <p className="mt-1 text-xs text-slate-300">
                Use these buttons to zoom out, zoom in, or center/fit the map
                view.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <span className="btn btn-xs btn-square btn-ghost pointer-events-none">
                  <ZoomOut className="h-4 w-4" />
                </span>
                <span className="btn btn-xs btn-square btn-ghost pointer-events-none">
                  <ZoomIn className="h-4 w-4" />
                </span>
                <span className="btn btn-xs btn-square btn-ghost pointer-events-none">
                  <LocateFixed className="h-4 w-4" />
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">
                Uploader status badge
              </p>
              <p className="mt-1 text-xs text-slate-300">
                Green indicates active uploaders. Red pulse indicates no active
                uploaders.
              </p>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <span className="flex items-center rounded-full bg-base-300/70 px-2 py-1 text-xs">
                  <span className="intel-badge-icon-bg inline-flex h-5 w-5 items-center justify-center rounded-full">
                    <Upload className="intel-status-text-active h-3 w-3" />
                  </span>
                </span>
                <span className="flex items-center rounded-full bg-base-300/70 px-2 py-1 text-xs">
                  <span className="intel-badge-icon-bg intel-status-icon--alert inline-flex h-5 w-5 items-center justify-center rounded-full">
                    <Upload className="intel-status-text-stale h-3 w-3" />
                  </span>
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">
                Report staleness badge
              </p>
              <p className="mt-1 text-xs text-slate-300">
                Green means fresh, yellow means stale, red pulse means very
                stale or no reports.
              </p>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <span className="flex items-center rounded-full bg-base-300/70 px-2 py-1 text-xs">
                  <span className="intel-badge-icon-bg inline-flex h-5 w-5 items-center justify-center rounded-full">
                    <ScrollText className="intel-status-text-active h-3 w-3" />
                  </span>
                </span>
                <span className="flex items-center rounded-full bg-base-300/70 px-2 py-1 text-xs">
                  <span className="intel-badge-icon-bg inline-flex h-5 w-5 items-center justify-center rounded-full">
                    <ScrollText className="intel-status-text-warn h-3 w-3" />
                  </span>
                </span>
                <span className="flex items-center rounded-full bg-base-300/70 px-2 py-1 text-xs">
                  <span className="intel-badge-icon-bg intel-status-icon--alert inline-flex h-5 w-5 items-center justify-center rounded-full">
                    <ScrollText className="intel-status-text-stale h-3 w-3" />
                  </span>
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">Mute control</p>
              <p className="mt-1 text-xs text-slate-300">
                Toggles intel alarm sound between muted and enabled states.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <span className="btn btn-xs btn-ghost pointer-events-none">
                  <Volume2 className="h-4 w-4" />
                </span>
                <span className="btn btn-xs btn-square btn-error btn-outline pointer-events-none">
                  <VolumeX className="h-4 w-4" />
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">
                Intel panel toggle
              </p>
              <p className="mt-1 text-xs text-slate-300">
                In full-page map mode, use this to hide or show the intel
                console panel.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <span className="btn btn-xs btn-ghost pointer-events-none">
                  <ChevronDown className="h-4 w-4" />
                </span>
                <span className="btn btn-xs btn-ghost pointer-events-none">
                  <ChevronUp className="h-4 w-4" />
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">
                Panel and full-page view toggle
              </p>
              <p className="mt-1 text-xs text-slate-300">
                Switch between split-panel and full-page map viewing modes.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <span className="btn btn-xs btn-ghost pointer-events-none">
                  <LayoutGrid className="h-3.5 w-3.5" />
                </span>
                <span className="btn btn-xs btn-ghost pointer-events-none">
                  <Maximize2 className="h-3.5 w-3.5" />
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">Theme toggle</p>
              <p className="mt-1 text-xs text-slate-300">
                Switch between dark and light mode. Try it out!
              </p>
              <div className="mt-2 flex items-center gap-2">
                <ThemeToggle />
              </div>
            </div>
          </div>
        </HelpSection>
        <HelpSection
          title="Filter controls"
          icon={<Filter className="h-4 w-4 text-slate-400" />}
        >
          <div className="space-y-4">
            <div>
              <p className="text-sm font-medium text-slate-100">
                System filters
              </p>
              <p className="mt-1 text-xs text-slate-300">
                Shift + left click toggles a system in Intel filters. Selected
                systems show a green map label.
              </p>
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
              <p className="text-sm font-medium text-slate-100">
                Intel feed targets
              </p>
              <p className="mt-1 text-xs text-slate-300">
                Logs/Alarm toggles choose where system filters apply.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <span className="btn btn-xs btn-info pointer-events-none">
                  Logs
                </span>
                <span className="btn btn-xs btn-ghost pointer-events-none">
                  Alarm
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-slate-100">
                Character-based filters
              </p>
              <p className="mt-1 text-xs text-slate-300">
                Build filters from active character locations by same region or
                within a jump range.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <span className="btn btn-xs btn-outline pointer-events-none">
                  Same region
                </span>
                <span className="btn btn-xs btn-outline pointer-events-none">
                  Within
                </span>
                <span className="rounded border border-slate-700 bg-slate-900 px-2 py-0.5 text-xs text-slate-300">
                  3
                </span>
              </div>
            </div>
          </div>
        </HelpSection>
        <HelpSection
          title="Map legend"
          icon={<Map className="h-4 w-4 text-slate-400" />}
        >
          <h5 className="text-sm font-medium text-slate-200">Lines</h5>
          <div className="space-y-1 text-sm">
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
                <line
                  x1="4"
                  y1="5"
                  x2="28"
                  y2="5"
                  className="map-gate region"
                  style={{
                    stroke: "var(--map-unloaded-region-color, #9c27b0)",
                  }}
                />
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
            <div className="flex items-center gap-2">
              <svg width="32" height="10">
                <line x1="4" y1="5" x2="28" y2="5" className="map-gate route" />
                <line
                  x1="4"
                  y1="5"
                  x2="28"
                  y2="5"
                  className="map-gate route route-blink"
                  style={{
                    stroke: "#62c474",
                  }}
                />
              </svg>
              Navigation route
            </div>
          </div>
          <h5 className="pt-1 text-sm font-medium text-slate-200">
            Threat colors
          </h5>
          <p className="text-sm text-slate-400">
            A reported system starts at flashing, then cascades through red,
            orange, yellow, and green as time expires. These values reflect your
            current settings and can be changed on the Settings page; each stage
            starts after the previous one ends.
          </p>
          <div className="space-y-2">
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
          <h5 className="pt-1 text-sm font-medium text-slate-200">
            System border states
          </h5>
          <p className="text-sm text-slate-400">
            Border color priority is route head, then undocked, then docked.
            Route waypoints use a steady route border.
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
                  className="map-system-border map-system-border-pulse map-system-border-pulse-slow"
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
        </HelpSection>
        <HelpSection
          title="Character visibility"
          icon={<UserRound className="h-4 w-4 text-slate-400" />}
        >
          <ul className="ml-4 list-disc space-y-1 text-sm text-slate-300">
            <li>
              <span className="font-semibold text-slate-100">Checkboxes:</span>{" "}
              control whether each character and their route are shown on the
              map.
            </li>
            <li>
              <span className="font-semibold text-slate-100">
                Docked/undocked:
              </span>{" "}
              undocked characters pulse blue, docked characters pulse slate.
            </li>
            <li>
              <span className="font-semibold text-slate-100">
                Name click focus:
              </span>{" "}
              clicking a character name centers the map on that character.
            </li>
          </ul>
        </HelpSection>
        <HelpSection
          title="Context menus"
          icon={<Menu className="h-4 w-4 text-slate-400" />}
        >
          <ul className="ml-4 list-disc space-y-1 text-sm text-slate-300">
            <li>
              <span className="font-semibold text-slate-100">Map menu:</span>{" "}
              set jumprange options or clear route.
            </li>
            <li>
              <span className="font-semibold text-slate-100">System menu:</span>{" "}
              show jumpranges, set route, or open zkill/dotlan.
            </li>
            <li>
              <span className="font-semibold text-slate-100">
                Intel Report Item Menu:
              </span>{" "}
              search selected player names from intel logs.
            </li>
            <li>
              <span className="font-semibold text-slate-100">Jump ranges:</span>{" "}
              left click sets primary/secondary anchors, then use the jumprange
              context menu to show or clear overlays.
            </li>
          </ul>
        </HelpSection>
        <HelpSection
          title="Saved settings"
          icon={<Save className="h-4 w-4 text-slate-400" />}
        >
          <p className={sectionTextClass}>
            Map regions, intel log filters, and settings are stored in your
            browser.
          </p>
          <div>
            <p className="text-sm font-medium text-slate-100">Settings page</p>
            <p className="mt-1 text-xs text-slate-300">
              Open the Settings page to adjust map behavior, threat timings,
              alarm options, and appearance.
            </p>
          </div>
        </HelpSection>
      </div>
      <p className="px-1 text-center text-xs italic text-slate-500">
        This map can get laggy at times, so it may not run well on older
        machines.
      </p>
    </>
  );
}

function HelpDialogActions() {
  const { close } = useModalBody();
  const introMode = useSettingsStore((s) => s.settings.introduction);
  const setIntroduction = useSettingsStore((s) => s.setIntroduction);
  const [introCountdown, setIntroCountdown] = useState(3);

  useEffect(() => {
    if (!introMode) {
      setIntroCountdown(0);
      return;
    }
    setIntroCountdown(3);
    const timer = setInterval(() => {
      setIntroCountdown((value) => {
        if (value <= 1) {
          clearInterval(timer);
          return 0;
        }
        return value - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [introMode]);

  const handleClose = () => {
    if (introMode) {
      setIntroduction(false);
    }
    close();
  };

  return (
    <>
      <button
        className="btn btn-sm btn-outline"
        onClick={handleClose}
        disabled={introMode && introCountdown > 0}
      >
        {introMode && introCountdown > 0 ? introCountdown : "Close"}
      </button>
    </>
  );
}

export const AppModalHelp = defineUIDialogModal({
  key: UI_DIALOG.Help,
  useOpen: () => useUIStore((s) => s.dialogs[UI_DIALOG.Help]),
  build: () => ({
    title: useSettingsStore.getState().settings.introduction
      ? "Welcome to Sentinel"
      : "Help",
    body: <HelpDialogBody />,
    actions: <HelpDialogActions />,
    sizeClass: "max-w-none w-[calc(100vw-2rem)] xl:w-[50vw]",
    dismissible: !useSettingsStore.getState().settings.introduction,
  }),
});

export default function HelpDialog() {
  const introMode = useSettingsStore((s) => s.settings.introduction);
  const helpOpen = useUIStore((s) => s.dialogs[UI_DIALOG.Help]);
  const setDialog = useUIStore((s) => s.setModal);
  useEffect(() => {
    if (!introMode || helpOpen) {
      return;
    }
    const timer = window.setTimeout(() => {
      setDialog(UI_DIALOG.Help, true);
    }, 100);
    return () => window.clearTimeout(timer);
  }, [helpOpen, introMode, setDialog]);

  useModal(AppModalHelp);

  return null;
}
