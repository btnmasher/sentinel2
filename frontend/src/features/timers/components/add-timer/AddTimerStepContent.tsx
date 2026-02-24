import {
  moonOnlyStructureTypes,
  planetOnlyStructureTypes,
} from "../../config/timerOptions";
import { useTimerCelestials } from "../../hooks/useTimerCelestials";
import { useTimerFormOptions } from "../../hooks/useTimerFormOptions";
import { useTimerOwnerSuggestions } from "../../hooks/useTimerOwnerSuggestions";
import { useTimerFormStore } from "../../store/useTimerFormStore";
import {
  contextToneClass,
  replacementToneClass,
  severityToneClass,
  toneButtonClass,
} from "../../utils/timerToneButtons";
import StepContext from "./steps/StepContext";
import StepLocation from "./steps/StepLocation";
import StepOwner from "./steps/StepOwner";
import StepPriority from "./steps/StepPriority";
import StepReplacement from "./steps/StepReplacement";
import StepTime from "./steps/StepTime";

type Props = {
  selectedExpiresAt: Date | null;
  parsePastedText: () => void;
  eveDisplayDateToISO: (value: Date) => string;
};

export default function AddTimerStepContent({
  selectedExpiresAt,
  parsePastedText,
  eveDisplayDateToISO,
}: Props) {
  const form = useTimerFormStore((s) => s.form);
  const planets = useTimerFormStore((s) => s.planets);
  const setPlanets = useTimerFormStore((s) => s.setPlanets);
  const moons = useTimerFormStore((s) => s.moons);
  const setMoons = useTimerFormStore((s) => s.setMoons);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  const step = useTimerFormStore((s) => s.step);
  const requiresPlanet =
    form.structure_type !== "" &&
    planetOnlyStructureTypes.has(form.structure_type);
  const requiresMoon =
    form.structure_type !== "" &&
    moonOnlyStructureTypes.has(form.structure_type);
  const allowsPlanet = !requiresMoon;
  const allowsMoon = !requiresPlanet;
  const {
    contextOptions,
    hostilityOptions,
    replacementOptions,
    severityOptions,
    structureGroups,
  } = useTimerFormOptions(form.structure_type);
  const celestialOptions = useTimerCelestials({
    form,
    allowsPlanet,
    allowsMoon,
    planets,
    moons,
    setPlanets,
    setMoons,
    updateForm,
  });
  const loadOwnerSuggestions = useTimerOwnerSuggestions();

  if (step === 1) {
    return (
      <StepTime
        selectedExpiresAt={selectedExpiresAt}
        parsePastedText={parsePastedText}
        eveDisplayDateToISO={eveDisplayDateToISO}
      />
    );
  }
  if (step === 2) {
    return (
      <StepLocation
        structureGroups={structureGroups}
        celestialOptions={celestialOptions}
        planets={planets}
        moons={moons}
        requiresPlanet={requiresPlanet}
        requiresMoon={requiresMoon}
        toneButtonClass={toneButtonClass}
      />
    );
  }
  if (step === 3) {
    return (
      <div className="space-y-4">
        <StepOwner
          loadOwnerSuggestions={loadOwnerSuggestions}
          toneButtonClass={toneButtonClass}
          hostilityOptions={hostilityOptions}
        />
        <StepContext
          contextOptions={contextOptions}
          contextToneClass={contextToneClass}
        />
        <StepReplacement
          replacementOptions={replacementOptions}
          replacementToneClass={replacementToneClass}
        />
      </div>
    );
  }
  if (step === 4) {
    return (
      <StepPriority
        severityOptions={severityOptions}
        severityToneClass={severityToneClass}
      />
    );
  }
  return null;
}
