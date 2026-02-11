import type { ReactNode } from "react";
import { NavLink } from "react-router-dom";
import { LayoutGrid, Maximize2, Menu } from "lucide-react";
import { useAuthStore } from "@/app/store/authStore";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useShallow } from "zustand/shallow";
import ThemeToggle from "@/components/ThemeToggle";
import { useSettingsStore } from "@/app/store/settingsStore";

export type MapNavItem = {
  label: string;
  to: string;
  external?: boolean;
  auth?: boolean;
  staff?: boolean;
  admin?: boolean;
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
  children?: ReactNode;
};

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
  children,
}: MapShellProps) {
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
    loaded: configLoaded,
    standaloneAuth,
    authBackend,
    oidcPortalUrl,
  } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      standaloneAuth: s.standaloneAuth,
      authBackend: s.authBackend,
      oidcPortalUrl: s.oidcPortalUrl,
    })),
  );
  const isPanelOpen = panelOpen ?? Boolean(panel);
  const showPanelMode = mapViewMode === "panel";
  const showNavMenu = !showPanelMode;
  const toggleViewMode = () =>
    setMapViewMode("map", "viewMode", showPanelMode ? "full" : "panel");

  if (showPanelMode) {
    return (
      <div className="grid gap-6 lg:grid-cols-[3fr_1fr] h-[calc(100vh-5rem-3rem)] items-stretch">
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
            <div className="card-body h-full min-h-0 overflow-auto p-4">
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

      {showNavMenu && navOpen && (
        <div className="absolute top-16 left-4 z-40 w-56 rounded-lg border border-slate-800 bg-base-200/90 shadow-lg backdrop-blur">
          <div className="px-3 py-2 text-[10px] uppercase tracking-[0.2em] text-slate-500">
            Menu
          </div>
          <div className="flex flex-col">
            {navItems.map((item) => {
              if (item.staff && !(authLoaded && isStaff)) return null;
              if (item.admin && !(authLoaded && isAdmin)) return null;
              if (item.auth && !(authLoaded && isAuthenticated)) {
                return null;
              }
              if (
                item.label === "Profile" &&
                configLoaded &&
                authBackend !== "eve"
              ) {
                return (
                  <a
                    key={item.to}
                    href={oidcPortalUrl}
                    className="px-3 py-2 text-sm hover:bg-base-300"
                    onClick={onCloseNav}
                  >
                    Profile
                  </a>
                );
              }
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    `px-3 py-2 text-sm hover:bg-base-300 ${
                      isActive ? "nav-active" : ""
                    }`.trim()
                  }
                  onClick={onCloseNav}
                >
                  {item.label}
                </NavLink>
              );
            })}
            {isAuthenticated && configLoaded && standaloneAuth && (
              <button
                className="px-3 py-2 text-sm text-left hover:bg-base-300"
                onClick={() => {
                  onCloseNav();
                  void logout();
                }}
              >
                Log out
              </button>
            )}
          </div>
        </div>
      )}

      <div className="absolute left-4 top-4 right-4 z-30">
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

      {panel && isPanelOpen && (
        <div
          className={`absolute top-16 right-4 z-20 max-h-[calc(100vh-7rem)] overflow-hidden rounded-xl border border-slate-800 bg-base-200/90 shadow-lg backdrop-blur mt-3 ${
            panelClassName ?? "w-80"
          }`}
        >
          <div className="max-h-[calc(100vh-7rem)] overflow-auto p-3">
            {panel}
          </div>
        </div>
      )}
    </div>
  );
}
