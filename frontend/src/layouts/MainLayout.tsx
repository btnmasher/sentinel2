import { ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { NavLink } from "react-router-dom";
import { LogOut } from "lucide-react";
import Dialogs from "@/components/Dialogs";
import GlobalModalHost from "@/components/GlobalModalHost";
import Toast from "@/components/Toast";
import ThemeToggle from "@/components/ThemeToggle";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useShallow } from "zustand/shallow";

const navLink = "text-sm uppercase tracking-[0.2em] nav-link px-1 py-0.5";

export default function MainLayout({ children }: { children: ReactNode }) {
  const location = useLocation();
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
  return (
    <div className="min-h-screen gradient-grid flex flex-col">
      {!fullPageMapMode && (
        <header className="px-6 py-4 h-20 border-b border-slate-800/60 bg-abyss/80 backdrop-blur">
          <div className="grid grid-cols-[1fr_4fr_1fr] items-center gap-6 w-full">
            <div className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg bg-primary/20 border border-primary flex items-center justify-center font-display text-lg">
                S2
              </div>
              <div>
                <h1 className="font-display text-xl">Sentinel 2</h1>
                <p className="text-xs text-slate-400">
                  intel & navigation control
                </p>
              </div>
            </div>
            <nav className="flex items-center justify-center gap-6 text-slate-300">
              <NavLink
                to="/"
                className={({ isActive }) =>
                  `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                }
              >
                Intel
              </NavLink>
              <NavLink
                to="/nav"
                className={({ isActive }) =>
                  `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                }
              >
                Navigation
              </NavLink>
              <NavLink
                to="/settings"
                className={({ isActive }) =>
                  `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                }
              >
                Settings
              </NavLink>
              {configLoaded && standaloneAuth && (
                <NavLink
                  to="/profile"
                  className={({ isActive }) =>
                    `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                  }
                >
                  Profile
                </NavLink>
              )}
              <NavLink
                to="/uploader"
                className={({ isActive }) =>
                  `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                }
              >
                Uploader
              </NavLink>
              {loaded && isStaff && configLoaded && standaloneAuth && (
                <NavLink
                  to="/staff"
                  className={({ isActive }) =>
                    `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                  }
                >
                  Staff
                </NavLink>
              )}
              {loaded && isAdmin && configLoaded && standaloneAuth && (
                <NavLink
                  to="/admin"
                  className={({ isActive }) =>
                    `${navLink} ${isActive ? "nav-active" : ""}`.trim()
                  }
                >
                  Admin
                </NavLink>
              )}
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
        </header>
      )}
      <main
        className={
          fullPageMapMode
            ? "h-screen w-screen relative overflow-hidden"
            : isMapMode
              ? "max-w-[1800px] mx-auto px-6 py-6 flex-1 w-full"
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
