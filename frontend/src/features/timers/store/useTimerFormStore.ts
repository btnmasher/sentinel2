import { create } from "zustand";
import type {
  TimerMoonOption,
  TimerPlanetOption,
  TimerForm,
  TimerFormStep,
} from "../types";

type TimerFormState = {
  form: TimerForm;
  step: TimerFormStep;
  systemQuery: string;
  ownerQuery: string;
  planets: TimerPlanetOption[];
  moons: TimerMoonOption[];
  setStep: (step: TimerFormStep) => void;
  setSystemQuery: (value: string) => void;
  setOwnerQuery: (value: string) => void;
  setPlanets: (items: TimerPlanetOption[]) => void;
  setMoons: (items: TimerMoonOption[]) => void;
  replaceForm: (form: TimerForm) => void;
  updateForm: (updater: (current: TimerForm) => TimerForm) => void;
  resetForm: () => void;
};

export function buildEmptyTimerForm(): TimerForm {
  return {
    raw_text: "",
    expires_at: "",
    system_id: 0,
    system: "",
    structure_type: "",
    planet_id: 0,
    planet_name: "",
    moon_id: 0,
    moon_name: "",
    owner_corporation_id: 0,
    owner_corporation_name: "",
    owner_corporation_ticker: "",
    owner_alliance_id: 0,
    owner_alliance_name: "",
    owner_alliance_ticker: "",
    standing_type: "",
    timer_kind: "",
    stage_label: "",
    context_selection: "",
    replacement_action: "",
    skyhook_fullness_pct: "",
    severity: "",
    title: "",
    notes: "",
    other_structure_note: "",
    timer_kind_note: "",
  };
}

export const useTimerFormStore = create<TimerFormState>((set) => ({
  form: buildEmptyTimerForm(),
  step: 1,
  systemQuery: "",
  ownerQuery: "",
  planets: [],
  moons: [],
  setStep: (step) => set({ step }),
  setSystemQuery: (systemQuery) => set({ systemQuery }),
  setOwnerQuery: (ownerQuery) => set({ ownerQuery }),
  setPlanets: (planets) => set({ planets }),
  setMoons: (moons) => set({ moons }),
  replaceForm: (form) => set({ form }),
  updateForm: (updater) =>
    set((state) => ({
      form: updater(state.form),
    })),
  resetForm: () =>
    set({
      form: buildEmptyTimerForm(),
      step: 1,
      systemQuery: "",
      ownerQuery: "",
      planets: [],
      moons: [],
    }),
}));
