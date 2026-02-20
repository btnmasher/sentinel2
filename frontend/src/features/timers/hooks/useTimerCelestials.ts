import { useEffect, useMemo } from "react";
import { api } from "@/config/api";
import type { TimerForm, TimerMoonOption, TimerPlanetOption } from "../types";

type UseTimerCelestialsInput = {
  form: TimerForm;
  allowsPlanet: boolean;
  allowsMoon: boolean;
  setPlanets: (items: TimerPlanetOption[]) => void;
  setMoons: (items: TimerMoonOption[]) => void;
  updateForm: (updater: (current: TimerForm) => TimerForm) => void;
  planets: TimerPlanetOption[];
  moons: TimerMoonOption[];
};

type CelestialOption = {
  kind: "item" | "section";
  type: "planet" | "moon";
  id: number;
  name: string;
  label: string;
  description?: string;
};

export function useTimerCelestials({
  form,
  allowsPlanet,
  allowsMoon,
  setPlanets,
  setMoons,
  updateForm,
  planets,
  moons,
}: UseTimerCelestialsInput): CelestialOption[] {
  useEffect(() => {
    if (!allowsPlanet || form.system_id <= 0) {
      setPlanets([]);
      return;
    }
    api
      .get<{ planets: TimerPlanetOption[] }>(
        `/timers/planets?system_id=${form.system_id}`,
      )
      .then((response) => setPlanets(response.data.planets || []))
      .catch(() => setPlanets([]));
  }, [allowsPlanet, form.system_id, setPlanets]);

  useEffect(() => {
    if (!allowsMoon || form.system_id <= 0) {
      setMoons([]);
      return;
    }
    api
      .get<{ moons: TimerMoonOption[] }>(
        `/timers/moons?system_id=${form.system_id}`,
      )
      .then((response) => setMoons(response.data.moons || []))
      .catch(() => setMoons([]));
  }, [allowsMoon, form.system_id, setMoons]);

  useEffect(() => {
    if (!allowsPlanet && form.planet_id) {
      updateForm((state) => ({
        ...state,
        planet_id: 0,
        planet_name: "",
      }));
    }
    if (!allowsMoon && form.moon_id) {
      updateForm((state) => ({
        ...state,
        moon_id: 0,
        moon_name: "",
      }));
    }
  }, [allowsMoon, allowsPlanet, form.moon_id, form.planet_id, updateForm]);

  return useMemo(() => {
    const options: CelestialOption[] = [];

    if (allowsPlanet) {
      options.push({
        kind: "section",
        type: "planet",
        id: 0,
        name: "",
        label: "Planets",
      });
      planets.forEach((planet) => {
        options.push({
          kind: "item",
          type: "planet",
          id: planet.id,
          name: planet.name,
          label: planet.name,
          description: "Planet",
        });
      });
    }
    if (allowsMoon) {
      options.push({
        kind: "section",
        type: "moon",
        id: 0,
        name: "",
        label: "Moons",
      });
      moons.forEach((moon) => {
        options.push({
          kind: "item",
          type: "moon",
          id: moon.id,
          name: moon.name,
          label: moon.name,
          description: "Moon",
        });
      });
    }

    return options;
  }, [allowsMoon, allowsPlanet, moons, planets]);
}
