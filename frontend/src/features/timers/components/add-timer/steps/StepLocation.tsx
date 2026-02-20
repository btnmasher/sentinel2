import { Check } from "lucide-react";
import SelectionDropdown from "@/components/SelectionDropdown";
import { SystemSearchField } from "@/features/map";
import type {
  StructureGroup,
  StructureTone,
} from "../../../config/timerOptions";
import { useTimerFormStore } from "../../../store/useTimerFormStore";
import { TimerStructureType } from "../../../types";

type Props = {
  structureGroups: ReadonlyArray<StructureGroup>;
  celestialOptions: ReadonlyArray<{
    kind: "item" | "section";
    type: "planet" | "moon";
    id: number;
    name: string;
    label: string;
    description?: string;
  }>;
  planets: ReadonlyArray<{ id: number; name: string }>;
  moons: ReadonlyArray<{ id: number; name: string }>;
  requiresPlanet: boolean;
  requiresMoon: boolean;
  toneButtonClass: (tone: StructureTone, active: boolean) => string;
};

export default function StepLocation({
  structureGroups,
  celestialOptions,
  planets,
  moons,
  requiresPlanet,
  requiresMoon,
  toneButtonClass,
}: Props) {
  const form = useTimerFormStore((s) => s.form);
  const systemQuery = useTimerFormStore((s) => s.systemQuery);
  const setSystemQuery = useTimerFormStore((s) => s.setSystemQuery);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  return (
    <div className="space-y-4 rounded-xl border border-slate-700/70 bg-base-300/20 p-4">
      <div className="space-y-2">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Location
        </div>
        <div className="grid gap-3 md:grid-cols-2">
          <label className="form-control">
            <div className="label py-1">
              <span className="label-text text-xs uppercase tracking-wide">
                System
              </span>
            </div>
            <SystemSearchField
              query={systemQuery}
              onQueryChange={(value) => {
                setSystemQuery(value);
                const normalized = value.trim().toLowerCase();
                if (normalized !== form.system.trim().toLowerCase()) {
                  updateForm((state) => ({
                    ...state,
                    system_id: 0,
                    system: "",
                    planet_id: 0,
                    planet_name: "",
                    moon_id: 0,
                    moon_name: "",
                  }));
                }
              }}
              selectionInputMode="set"
              inputClassName={`input input-bordered h-10 ${
                form.system_id > 0 &&
                systemQuery.trim().toLowerCase() ===
                  form.system.trim().toLowerCase()
                  ? "border-emerald-500/80 bg-emerald-500/5 ring-1 ring-emerald-500/25"
                  : ""
              }`}
              onSelect={(system) =>
                updateForm((state) => ({
                  ...state,
                  system_id: system.id,
                  system: system.name,
                }))
              }
            />
            {form.system_id > 0 &&
              systemQuery.trim().toLowerCase() ===
                form.system.trim().toLowerCase() && (
                <div className="mt-1 flex items-center gap-1.5 text-xs text-success">
                  <Check className="h-3.5 w-3.5" />
                  <span>Selected</span>
                </div>
              )}
          </label>

          <label className="form-control">
            <div className="label py-1">
              <span className="label-text text-xs uppercase tracking-wide">
                Celestial
              </span>
            </div>
            <SelectionDropdown
              items={celestialOptions.map((item, index) => ({
                id:
                  item.kind === "section"
                    ? `section-${item.type}-${index}`
                    : `${item.type}:${item.id}`,
                label: item.label,
                description: item.description,
                kind: item.kind,
                disabled: item.kind === "section",
              }))}
              selected={
                form.planet_id
                  ? [`planet:${form.planet_id}`]
                  : form.moon_id
                    ? [`moon:${form.moon_id}`]
                    : []
              }
              onChange={(next) => {
                const value = next[0] || "";
                if (!value) {
                  updateForm((state) => ({
                    ...state,
                    planet_id: 0,
                    planet_name: "",
                    moon_id: 0,
                    moon_name: "",
                  }));
                  return;
                }
                const [type, idToken] = value.split(":");
                const id = Number(idToken);
                if (type === "planet") {
                  const selected = planets.find((planet) => planet.id === id);
                  updateForm((state) => ({
                    ...state,
                    planet_id: selected?.id || 0,
                    planet_name: selected?.name || "",
                    moon_id: 0,
                    moon_name: "",
                  }));
                  return;
                }
                const selected = moons.find((moon) => moon.id === id);
                updateForm((state) => ({
                  ...state,
                  moon_id: selected?.id || 0,
                  moon_name: selected?.name || "",
                  planet_id: 0,
                  planet_name: "",
                }));
              }}
              searchable
              multi={false}
              disabled={form.system_id <= 0}
              label="Celestial"
              placeholder={
                requiresPlanet
                  ? "Select planet"
                  : requiresMoon
                    ? "Select moon"
                    : "Select celestial (optional)"
              }
              buttonClassName="input input-bordered h-10 w-full justify-between"
              menuClassName="z-[12000]"
            />
          </label>
        </div>
      </div>

      <div className="space-y-2">
        <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
          Structure Selection
        </div>
        <div className="space-y-4 rounded-lg border border-slate-700/50 bg-base-200/50 p-3">
          {structureGroups.map((group) => (
            <div key={group.group} className="space-y-2">
              <div className="text-[10px] uppercase tracking-[0.22em] text-slate-500">
                {group.group}
              </div>
              <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                {group.options.map((option) => {
                  const Icon = option.icon;
                  const active = form.structure_type === option.value;
                  return (
                    <button
                      key={option.value}
                      className={`btn btn-sm h-auto min-h-11 justify-start py-2 text-left leading-tight whitespace-normal ${toneButtonClass(option.tone, active)}`}
                      onClick={() =>
                        updateForm((state) => ({
                          ...state,
                          structure_type: option.value,
                        }))
                      }
                    >
                      <Icon className="h-4 w-4" /> {option.label}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
          {form.structure_type === TimerStructureType.Custom && (
            <label className="form-control mt-2">
              <div className="label py-1">
                <span className="label-text text-xs uppercase tracking-wide">
                  Other/Misc note
                </span>
              </div>
              <input
                className="input input-bordered"
                value={form.other_structure_note}
                onChange={(event) =>
                  updateForm((state) => ({
                    ...state,
                    other_structure_note: event.target.value,
                  }))
                }
                placeholder="Short structure description"
              />
            </label>
          )}
        </div>
      </div>
    </div>
  );
}
