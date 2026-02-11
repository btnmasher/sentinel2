import { useMemo } from "react";

import SelectionDropdown from "@/components/SelectionDropdown";

import { MapLayout } from "../types";
import { useMapStore } from "../store/mapStore";
import { REGIONS } from "../types/regions";

type RegionSelectProps = {
  multi?: boolean;
  label?: string;
  searchable?: boolean;
  showTags?: boolean;
  selected?: string[];
  onChange?: (next: string[]) => void;
};

type MapLayoutSelectProps = {
  label?: string;
  inlineLabel?: string;
  value?: MapLayout;
  onChange?: (next: MapLayout) => void;
};

export default function RegionSelect({
  multi = false,
  label,
  searchable = true,
  showTags,
  selected,
  onChange,
}: RegionSelectProps) {
  const mapRegions = useMapStore((s) => s.mapRegions);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const regionOptions = useMemo(
    () =>
      REGIONS.map((region) => ({
        id: String(region.region),
        label: region.name,
      })),
    [],
  );
  const currentSelected = selected ?? mapRegions;
  const handleChange =
    onChange ??
    ((next: string[]) => {
      const nextValue = multi ? next : next.slice(0, 1);
      updateMapConfig({ mapRegions: nextValue });
    });

  return (
    <SelectionDropdown
      items={regionOptions}
      selected={currentSelected}
      onChange={(next) => handleChange(multi ? next : next.slice(0, 1))}
      multi={multi}
      showTags={showTags ?? multi}
      searchable={searchable}
      label={label ?? (multi ? "Regions" : "Region")}
    />
  );
}

export function MapLayoutSelect({
  value,
  onChange,
  inlineLabel,
  label = "Layout",
}: MapLayoutSelectProps) {
  const mapLayout = useMapStore((s) => s.mapLayout);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const layoutOptions = useMemo(
    () => [
      { id: "dotlan", label: "Dotlan" },
      { id: "eve2d", label: "EVE Flat 2D" },
      { id: "metro", label: "Metro" },
      { id: "real", label: "Real" },
    ],
    [],
  );
  const currentValue = value ?? mapLayout;
  const handleChange =
    onChange ?? ((next: MapLayout) => updateMapConfig({ mapLayout: next }));

  const dropdown = (
    <SelectionDropdown
      items={layoutOptions}
      selected={[currentValue]}
      onChange={(next) => handleChange((next[0] ?? "dotlan") as MapLayout)}
      label={label}
    />
  );

  if (!inlineLabel) {
    return dropdown;
  }

  return (
    <div className="flex items-center gap-2">
      <span>{inlineLabel}</span>
      {dropdown}
    </div>
  );
}
