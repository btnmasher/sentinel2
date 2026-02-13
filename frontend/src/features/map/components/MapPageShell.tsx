import { useEffect, useState } from "react";
import type { ReactNode } from "react";

import MapShell, { MapNavItem } from "./MapShell";
import { useMapStore } from "../store/mapStore";
import CharacterLocationRefresher from "./CharacterLocationRefresher";

const MAP_NAV_ITEMS: MapNavItem[] = [
  { label: "Intel", to: "/" },
  { label: "Navigation", to: "/nav" },
  { label: "Settings", to: "/settings" },
  { label: "Profile", to: "/profile", auth: true },
  { label: "Uploader", to: "/uploader" },
  { label: "Staff", to: "/staff", staff: true },
  { label: "Admin", to: "/admin", admin: true },
];

type MapPageShellProps = {
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

export default function MapPageShell({
  pageBadge,
  leftControls,
  rightControls,
  panel,
  panelOpen,
  panelClassName,
  onAutoHidePanel,
  children,
}: MapPageShellProps) {
  const [navOpen, setNavOpen] = useState(false);
  const loadCharacters = useMapStore((s) => s.loadCharacters);

  useEffect(() => {
    loadCharacters();
  }, [loadCharacters]);

  return (
    <>
      <CharacterLocationRefresher />
      <MapShell
        navItems={MAP_NAV_ITEMS}
        navOpen={navOpen}
        onToggleNav={() => setNavOpen((prev) => !prev)}
        onCloseNav={() => setNavOpen(false)}
        pageBadge={pageBadge}
        leftControls={leftControls}
        rightControls={rightControls}
        panel={panel}
        panelOpen={panelOpen}
        panelClassName={panelClassName}
        onAutoHidePanel={onAutoHidePanel}
      >
        {children}
      </MapShell>
    </>
  );
}
