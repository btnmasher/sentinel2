import { useEffect, useRef, type ReactNode } from "react";
import { LayoutGrid, Maximize2, Menu } from "lucide-react";
import { useAuthStore } from "@/app/store/authStore";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useShallow } from "zustand/shallow";
import ThemeToggle from "@/components/ThemeToggle";
import ResponsiveNavMenu, {
  type ResponsiveNavMenuItem,
} from "@/components/ResponsiveNavMenu";
import { useSettingsStore } from "@/app/store/settingsStore";

export type MapNavItem = {
  label: string;
  to: string;
  external?: boolean;
  auth?: boolean;
  staff?: boolean;
  admin?: boolean;
  timers?: boolean;
};

type MapShellProps = {
  navItems: MapNavItem[];
  navOpen: boolean;
  onToggleNav: () => void;
  onCloseNav: () => void;
  pageBadge?: {
    icon: ReactNode;
    label: string;
  };
  leftControls?: ReactNode;
  rightControls?: ReactNode;
  panel?: ReactNode;
  panelOpen?: boolean;
  panelClassName?: string;
  onAutoHidePanel?: () => void;
  children?: ReactNode;
};

const panelModeMinViewportWidth = 1024;

function MapTopBar({
  className = "",
  showNavToggle,
  onToggleNav,
  pageBadge,
  leftControls,
  rightControls,
  viewMode,
  onToggleViewMode,
  showThemeToggle,
}: {
  className?: string;
  showNavToggle: boolean;
  onToggleNav: () => void;
  pageBadge?: {
    icon: ReactNode;
    label: string;
  };
  leftControls?: ReactNode;
  rightControls?: ReactNode;
  viewMode: "full" | "panel";
  onToggleViewMode: () => void;
  showThemeToggle: boolean;
}) {
  return (
    <div
      className={`grid grid-cols-[auto_1fr_auto] items-center gap-3 ${className}`.trim()}
    >
      <div className="flex flex-wrap items-center gap-2">
        {showNavToggle && (
          <button
            className="btn btn-sm btn-square btn-primary btn-outline"
            onClick={onToggleNav}
            aria-label="Toggle navigation"
          >
            <Menu className="h-4 w-4" />
          </button>
        )}
        {pageBadge && (
          <span className="inline-flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-[10px] uppercase tracking-[0.2em] text-slate-300">
            {pageBadge.icon}
            {pageBadge.label}
          </span>
        )}
        {leftControls}
      </div>
      <div />
      <div className="flex items-center gap-2 justify-end">
        {rightControls}
        <button
          className="btn btn-xs btn-ghost btn-square"
          onClick={onToggleViewMode}
          aria-label={
            viewMode === "full"
              ? "Switch to panel view"
              : "Switch to full-screen view"
          }
          title={
            viewMode === "full"
              ? "Switch to panel view"
              : "Switch to full-screen view"
          }
        >
          {viewMode === "full" ? (
            <LayoutGrid className="h-4 w-4" />
          ) : (
            <Maximize2 className="h-4 w-4" />
          )}
        </button>
        {showThemeToggle && <ThemeToggle />}
      </div>
    </div>
  );
}

export default function MapShell({
  navItems,
  navOpen,
  onToggleNav,
  onCloseNav,
  pageBadge,
  leftControls,
  rightControls,
  panel,
  panelOpen,
  panelClassName,
  onAutoHidePanel,
  children,
}: MapShellProps) {
  const autoSwitchedToFullRef = useRef(false);
  const { mapViewMode, setMapViewMode } = useSettingsStore(
    useShallow((s) => ({
      mapViewMode: s.settings.map.viewMode,
      setMapViewMode: s.apply,
    })),
  );
  const {
    loaded: authLoaded,
    isStaff,
    isAdmin,
    isAuthenticated,
    logout,
  } = useAuthStore(
    useShallow((s) => ({
      loaded: s.loaded,
      isStaff: s.isStaff,
      isAdmin: s.isAdmin,
      isAuthenticated: s.isAuthenticated,
      logout: s.logout,
    })),
  );
  const {
    timersEnabled,
  } = useAppConfigStore(
    useShallow((s) => ({
      timersEnabled: s.timersEnabled,
    })),
  );
  const isPanelOpen = panelOpen ?? Boolean(panel);
  const showPanelMode = mapViewMode === "panel";
  const showNavMenu = !showPanelMode;
  const responsiveNavItems: ResponsiveNavMenuItem[] = navItems
    .filter((item) => {
      if (item.staff && !(authLoaded && isStaff)) return false;
      if (item.admin && !(authLoaded && isAdmin)) return false;
      if (item.auth && !(authLoaded && isAuthenticated)) return false;
      if (item.timers && !timersEnabled) return false;
      return true;
    })
    .map((item) => {
      return {
        key: item.to,
        label: item.label,
        to: item.to,
      };
    });
  const toggleViewMode = () => {
    autoSwitchedToFullRef.current = false;
    setMapViewMode("map", "viewMode", showPanelMode ? "full" : "panel");
  };

  useEffect(() => {
    const syncResponsiveViewMode = () => {
      if (
        mapViewMode === "panel" &&
        window.innerWidth < panelModeMinViewportWidth
      ) {
        autoSwitchedToFullRef.current = true;
        setMapViewMode("map", "viewMode", "full");
        onAutoHidePanel?.();
        onCloseNav();
        return;
      }
      if (
        mapViewMode === "full" &&
        autoSwitchedToFullRef.current &&
        window.innerWidth >= panelModeMinViewportWidth
      ) {
        autoSwitchedToFullRef.current = false;
        setMapViewMode("map", "viewMode", "panel");
      }
    };
    syncResponsiveViewMode();
    window.addEventListener("resize", syncResponsiveViewMode);
    return () => window.removeEventListener("resize", syncResponsiveViewMode);
  }, [mapViewMode, onAutoHidePanel, onCloseNav, setMapViewMode]);

  if (showPanelMode) {
    return (
      <div
        className={`grid gap-6 h-[calc(100vh-4rem-3rem)] items-stretch ${
          panel
            ? "grid-cols-[minmax(0,1fr)_minmax(18rem,24rem)]"
            : "grid-cols-1"
        }`.trim()}
      >
        <section className="card bg-base-200/70 border border-slate-800 h-full min-h-0 overflow-hidden">
          <div className="card-body h-full min-h-0 grid grid-rows-[auto_1fr] gap-4 p-4">
            <MapTopBar
              showNavToggle={false}
              onToggleNav={onToggleNav}
              pageBadge={pageBadge}
              leftControls={leftControls}
              rightControls={rightControls}
              viewMode={mapViewMode}
              onToggleViewMode={toggleViewMode}
              showThemeToggle={false}
              className="rounded-lg border border-slate-800/60 bg-base-200/70 px-3 py-2"
            />
            <div className="relative min-h-0 overflow-hidden rounded-lg border border-slate-800 bg-base-300/40">
              <div className="absolute inset-0">{children}</div>
            </div>
          </div>
        </section>
        {panel && (
          <section className="card bg-base-200/70 border border-slate-800 h-full min-h-0 overflow-hidden">
            <div className="card-body h-full min-h-0 overflow-hidden p-4">
              {panel}
            </div>
          </section>
        )}
      </div>
    );
  }

  return (
    <div className="w-full h-full relative">
      <div className="absolute inset-0">{children}</div>

      <div className="absolute inset-4 z-30 pointer-events-none flex min-h-0 flex-col gap-3">
        <div className="pointer-events-auto shrink-0">
          <MapTopBar
            className="rounded-lg border border-slate-800 bg-base-200/90 px-3 py-2 shadow-lg backdrop-blur text-xs"
            showNavToggle={showNavMenu}
            onToggleNav={onToggleNav}
            pageBadge={pageBadge}
            leftControls={leftControls}
            rightControls={rightControls}
            viewMode={mapViewMode}
            onToggleViewMode={toggleViewMode}
            showThemeToggle
          />
        </div>

        <div className="min-h-0 flex flex-1 gap-4">
          <div className="min-w-0 flex-1 relative">
            <ResponsiveNavMenu
              open={showNavMenu && navOpen}
              items={responsiveNavItems}
              onClose={onCloseNav}
              onLogout={isAuthenticated ? () => void logout() : undefined}
              className="absolute top-0 left-0 pointer-events-auto bg-base-200/90"
            />
          </div>

          {panel && isPanelOpen && (
            <div
              className={`map-full-panel shrink-0 self-stretch h-full min-h-0 max-h-full overflow-hidden rounded-xl border border-slate-800 bg-base-200/90 shadow-lg backdrop-blur pointer-events-auto ${
                panelClassName ?? "w-80"
              }`}
            >
              <div className="min-h-0 max-h-full overflow-hidden p-3 flex flex-col">
                {panel}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
