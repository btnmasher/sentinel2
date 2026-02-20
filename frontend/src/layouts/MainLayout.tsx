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
  const { loaded: configLoaded, standaloneAuth } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      standaloneAuth: s.standaloneAuth,
    })),
  );

  useEffect(() => {
    setNavOpen(false);
  }, [location.pathname]);

  const navItems = useMemo(() => {
    const items = [
      { to: "/", label: "Intel" },
      { to: "/nav", label: "Navigation" },
      { to: "/timers", label: "Timers" },
      { to: "/settings", label: "Settings" },
      { to: "/uploader", label: "Uploader" },
    ];
    if (configLoaded && standaloneAuth) {
      items.splice(3, 0, { to: "/profile", label: "Profile" });
    }
    if (loaded && isStaff && configLoaded && standaloneAuth) {
      items.push({ to: "/staff", label: "Staff" });
    }
    if (loaded && isAdmin && configLoaded && standaloneAuth) {
      items.push({ to: "/admin", label: "Admin" });
    }
    return items;
  }, [configLoaded, isAdmin, isStaff, loaded, standaloneAuth]);

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
                <h1 className="font-display text-base md:text-lg leading-tight">
                  Sentinel 2
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
              {isAuthenticated && configLoaded && standaloneAuth && (
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
