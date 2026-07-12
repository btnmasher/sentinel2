import { ReactNode, useEffect, useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { NavLink } from "react-router-dom";
import { LogOut, Menu } from "lucide-react";
import Dialogs from "@/components/Dialogs";
import GlobalModalHost from "@/components/GlobalModalHost";
import Toast from "@/components/Toast";
import ThemeToggle from "@/components/ThemeToggle";
import ResponsiveNavMenu from "@/components/ResponsiveNavMenu";
import { SiteAnnouncementHost } from "@/features/announcements";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useShallow } from "zustand/shallow";

const navLink = "text-sm uppercase tracking-[0.2em] nav-link px-1 py-0.5";
type MainNavItem = {
  to: string;
  label: string;
  timers?: boolean;
  auth?: boolean;
  staff?: boolean;
  admin?: boolean;
};
const MAIN_NAV_ITEMS: MainNavItem[] = [
  { to: "/", label: "Intel" },
  { to: "/nav", label: "Navigation" },
  { to: "/timers", label: "Timers", timers: true },
  { to: "/settings", label: "Settings" },
  { to: "/profile", label: "Profile", auth: true },
  { to: "/uploader", label: "Uploader" },
  { to: "/staff", label: "Staff", staff: true },
  { to: "/admin", label: "Admin", admin: true },
];

export default function MainLayout({ children }: { children: ReactNode }) {
  const location = useLocation();
  const [navOpen, setNavOpen] = useState(false);
  const isMapMode = location.pathname === "/" || location.pathname === "/nav";
  const mapViewMode = useSettingsStore((s) => s.settings.map.viewMode);
  const fullPageMapMode = isMapMode && mapViewMode === "full";
  const { loaded, isStaff, isAdmin } = useAuthStore(
    useShallow((s) => ({
      loaded: s.loaded,
      isStaff: s.isStaff,
      isAdmin: s.isAdmin,
    })),
  );
  const { isAuthenticated, logout } = useAuthStore(
    useShallow((s) => ({
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

  useEffect(() => {
    setNavOpen(false);
  }, [location.pathname]);

  const navItems = useMemo(() => {
    return MAIN_NAV_ITEMS.filter((item) => {
      if (item.timers && !timersEnabled) return false;
      if (item.auth && !(loaded && isAuthenticated)) return false;
      if (item.staff && !(loaded && isStaff)) {
        return false;
      }
      if (item.admin && !(loaded && isAdmin)) {
        return false;
      }
      return true;
    });
  }, [
    isAdmin,
    isAuthenticated,
    isStaff,
    loaded,
    timersEnabled,
  ]);

  return (
    <div className="min-h-screen gradient-grid flex flex-col">
      {!fullPageMapMode && (
        <header className="relative z-50 px-4 md:px-6 py-2 h-16 border-b border-slate-800/60 bg-abyss/80 backdrop-blur">
          <div className="grid grid-cols-[auto_1fr_auto] items-center gap-3 md:gap-5 w-full">
            <div className="flex items-center gap-3">
              <button
                className="btn btn-sm btn-square btn-primary btn-outline xl:hidden"
                onClick={() => setNavOpen((prev) => !prev)}
                aria-label="Toggle navigation"
              >
                <Menu className="h-4 w-4" />
              </button>
              <div className="w-8 h-8 rounded-lg bg-primary/20 border border-primary flex items-center justify-center font-display text-sm">
                S2
              </div>
              <div>
                <h1 className="font-display text-base md:text-lg leading-tight flex items-center gap-2">
                  <span>Sentinel 2</span>
                  <span className="inline-flex h-4 items-center rounded-full border border-amber-400/45 bg-amber-500/12 px-1.5 text-[9px] font-semibold uppercase leading-none tracking-[0.12em] text-amber-300">
                    beta
                  </span>
                </h1>
                <p className="hidden md:block text-[11px] text-slate-400">
                  intel & navigation control
                </p>
              </div>
            </div>
            <nav className="hidden xl:flex items-center justify-center gap-6 text-slate-300">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                  }
                >
                  {item.label}
                </NavLink>
              ))}
            </nav>
            <div className="justify-self-end flex items-center gap-2">
              {isAuthenticated && (
                <button className="btn btn-sm btn-ghost gap-2" onClick={logout}>
                  <LogOut className="h-4 w-4" />
                  Log out
                </button>
              )}
              <ThemeToggle className="btn btn-sm btn-ghost btn-square" />
            </div>
          </div>
          <ResponsiveNavMenu
            open={navOpen}
            items={navItems}
            onClose={() => setNavOpen(false)}
            onLogout={isAuthenticated ? () => void logout() : undefined}
            className="absolute top-full left-4 md:left-6 mt-2 z-[70] xl:hidden"
          />
        </header>
      )}
      <SiteAnnouncementHost />
      <main
        className={
          fullPageMapMode
            ? "h-screen w-screen relative overflow-hidden"
            : isMapMode
              ? "w-full px-6 py-6 flex-1"
              : "max-w-7xl mx-auto px-6 py-6 flex-1 w-full"
        }
      >
        {children}
      </main>
      <GlobalModalHost />
      <Dialogs />
      <Toast />
    </div>
  );
}
