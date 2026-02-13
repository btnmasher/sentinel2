import { NavLink } from "react-router-dom";

export type ResponsiveNavMenuItem = {
  key?: string;
  label: string;
  to?: string;
  href?: string;
};

type ResponsiveNavMenuProps = {
  open: boolean;
  items: ResponsiveNavMenuItem[];
  onClose: () => void;
  onLogout?: () => void;
  className?: string;
  title?: string;
};

export default function ResponsiveNavMenu({
  open,
  items,
  onClose,
  onLogout,
  className,
  title = "Menu",
}: ResponsiveNavMenuProps) {
  if (!open) {
    return null;
  }

  return (
    <div
      className={`w-56 rounded-lg border border-slate-800 bg-base-200/95 shadow-lg backdrop-blur ${className ?? ""}`.trim()}
    >
      <div className="px-3 py-2 text-[10px] uppercase tracking-[0.2em] text-slate-500">
        {title}
      </div>
      <div className="flex flex-col">
        {items.map((item, idx) => {
          const key = item.key ?? item.to ?? item.href ?? `${item.label}-${idx}`;
          if (item.href) {
            return (
              <a
                key={key}
                href={item.href}
                className="px-3 py-2 text-sm hover:bg-base-300"
                onClick={onClose}
              >
                {item.label}
              </a>
            );
          }
          if (item.to) {
            return (
              <NavLink
                key={key}
                to={item.to}
                className={({ isActive }) =>
                  `px-3 py-2 text-sm hover:bg-base-300 ${
                    isActive ? "nav-active" : ""
                  }`.trim()
                }
                onClick={onClose}
              >
                {item.label}
              </NavLink>
            );
          }
          return null;
        })}
        {onLogout && (
          <button
            className="px-3 py-2 text-sm text-left hover:bg-base-300"
            onClick={() => {
              onClose();
              onLogout();
            }}
          >
            Log out
          </button>
        )}
      </div>
    </div>
  );
}
