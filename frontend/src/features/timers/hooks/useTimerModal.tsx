import { useEffect, useMemo, useState } from "react";
import { Plus, SquarePen } from "lucide-react";
import useModal from "@/app/hooks/useModal";
import { useUIStore } from "@/app/store/uiStore";
import { api } from "@/config/api";
import AddTimerStepContent from "../components/add-timer/AddTimerStepContent";
import {
  contextOptionsFor,
  moonOnlyStructureTypes,
  planetOnlyStructureTypes,
  timerContextSelectionFromFields,
  timerKindLabels,
} from "../config/timerOptions";
import {
  eveDisplayDateToISO,
  isoToEveDisplayDate,
  nextUTCMidnightISO,
  validISO,
} from "../formatters";
import { useTimerFormStore } from "../store/useTimerFormStore";
import type { ParseTimerResponse, TimerFormStep, TimerRecord } from "../types";
import { TimerKind, TimerStageLabel, TimerStructureType } from "../types";

type UseTimerModalInput = {
  onSaved: () => Promise<void>;
};

export function useTimerModal({ onSaved }: UseTimerModalInput) {
  const setToast = useUIStore((s) => s.setToast);
  const form = useTimerFormStore((s) => s.form);
  const step = useTimerFormStore((s) => s.step);
  const setStep = useTimerFormStore((s) => s.setStep);
  const updateForm = useTimerFormStore((s) => s.updateForm);
  const replaceForm = useTimerFormStore((s) => s.replaceForm);
  const resetForm = useTimerFormStore((s) => s.resetForm);
  const setSystemQuery = useTimerFormStore((s) => s.setSystemQuery);
  const setOwnerQuery = useTimerFormStore((s) => s.setOwnerQuery);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editingTimer, setEditingTimer] = useState<TimerRecord | null>(null);

  const stepMeta: Array<{ value: TimerFormStep; label: string }> = [
    { value: 1, label: "Time" },
    { value: 2, label: "Location" },
    { value: 3, label: "Context" },
    { value: 4, label: "Priority" },
  ];

  const contextOptions = useMemo(
    () => contextOptionsFor(form.structure_type),
    [form.structure_type],
  );
  const requiresPlanet =
    form.structure_type !== "" &&
    planetOnlyStructureTypes.has(form.structure_type);
  const requiresMoon =
    form.structure_type !== "" &&
    moonOnlyStructureTypes.has(form.structure_type);
  const selectedExpiresAt = useMemo(
    () => isoToEveDisplayDate(form.expires_at),
    [form.expires_at],
  );

  useEffect(() => {
    const validContext = new Set(contextOptions.map((item) => item.value));
    if (!form.context_selection || !validContext.has(form.context_selection)) {
      updateForm((state) => ({
        ...state,
        context_selection: "",
        timer_kind: "",
        stage_label: "",
      }));
    }
  }, [form.context_selection, contextOptions, updateForm]);

  const parsePastedText = async () => {
    if (!form.raw_text.trim()) {
      setToast({ text: "Paste timer text first", color: "warning" });
      return;
    }
    try {
      const response = await api.post<ParseTimerResponse>("/timers/parse", {
        text: form.raw_text,
      });
      const data = response.data;
      updateForm((current) => ({
        ...current,
        title: current.title || data.title || "",
        system: current.system || data.system || "",
        system_id: current.system_id || data.system_id || 0,
        timer_kind: data.timer_kind || current.timer_kind,
        standing_type: data.standing_type || current.standing_type,
        context_selection: timerContextSelectionFromFields(
          data.timer_kind || current.timer_kind,
          current.stage_label,
        ),
        expires_at: data.expires_at || "",
      }));
      setStep(2);
      setToast({ text: "Parsed timer text", color: "success" });
    } catch {
      setToast({ text: "Could not parse timer text", color: "error" });
    }
  };

  const resetFormState = () => {
    resetForm();
    updateForm((state) => ({
      ...state,
      expires_at: nextUTCMidnightISO(),
    }));
    setEditingTimer(null);
  };

  const openCreateModal = () => {
    resetFormState();
    setOpen(true);
  };

  const openEditModal = (timer: TimerRecord) => {
    setEditingTimer(timer);
    replaceForm({
      raw_text: timer.raw_text || "",
      expires_at: timer.expires_at || "",
      system_id: timer.system_id || 0,
      system: timer.system_name || "",
      structure_type: timer.structure_type || "",
      planet_id: timer.planet_id || 0,
      planet_name: timer.planet_name || "",
      moon_id: timer.moon_id || 0,
      moon_name: timer.moon_name || "",
      owner_corporation_id: timer.owner_corporation_id || 0,
      owner_corporation_name: timer.owner_corporation_name || "",
      owner_corporation_ticker: timer.owner_corporation_ticker || "",
      owner_alliance_id: timer.owner_alliance_id || 0,
      owner_alliance_name: timer.owner_alliance_name || "",
      owner_alliance_ticker: timer.owner_alliance_ticker || "",
      standing_type: timer.standing_type || "",
      timer_kind: timer.timer_kind || "",
      stage_label: timer.stage_label || "",
      context_selection: timerContextSelectionFromFields(
        timer.timer_kind || "",
        timer.stage_label || "",
      ),
      replacement_action: timer.replacement_action || "",
      skyhook_fullness_pct:
        typeof timer.skyhook_fullness_pct === "number" &&
        Number.isFinite(timer.skyhook_fullness_pct) &&
        timer.skyhook_fullness_pct > 0
          ? String(timer.skyhook_fullness_pct)
          : "",
      severity: timer.severity || "",
      title: timer.title || "",
      notes: timer.notes || "",
      other_structure_note: "",
      timer_kind_note: "",
    });
    setSystemQuery(timer.system_name || "");
    setOwnerQuery(
      timer.owner_alliance_name || timer.owner_corporation_name || "",
    );
    setStep(1);
    setOpen(true);
  };

  const saveTimer = async () => {
    if (!form.expires_at || !form.system_id) {
      setToast({ text: "Time and system are required", color: "warning" });
      return;
    }
    if (!form.structure_type) {
      setToast({ text: "Structure type is required", color: "warning" });
      return;
    }
    if (!form.standing_type) {
      setToast({ text: "Hostility is required", color: "warning" });
      return;
    }
    if (!form.timer_kind) {
      setToast({ text: "Timer type is required", color: "warning" });
      return;
    }
    if (
      form.structure_type !== TimerStructureType.Custom &&
      !form.stage_label
    ) {
      setToast({ text: "Reinforcement context is required", color: "warning" });
      return;
    }
    if (!form.replacement_action) {
      setToast({ text: "Replacement context is required", color: "warning" });
      return;
    }
    if (!form.severity) {
      setToast({ text: "Severity is required", color: "warning" });
      return;
    }
    if (requiresMoon && !form.moon_id) {
      setToast({ text: "Moon selection is required", color: "warning" });
      return;
    }
    if (requiresPlanet && !form.planet_id) {
      setToast({ text: "Planet selection is required", color: "warning" });
      return;
    }

    const expiresAtISO = validISO(form.expires_at);
    if (!expiresAtISO) {
      setToast({ text: "Invalid EVE timestamp", color: "warning" });
      return;
    }
    if (new Date(expiresAtISO).getTime() <= Date.now()) {
      setToast({
        text: "Timer must be in the future (EVE Time)",
        color: "warning",
      });
      return;
    }

    setSaving(true);
    try {
      const combinedNotes = [
        (form.timer_kind === TimerKind.Extraction ||
          form.timer_kind === TimerKind.Custom) &&
        form.timer_kind_note.trim()
          ? `${(form.timer_kind && timerKindLabels[form.timer_kind]) || "Timer Type"}: ${form.timer_kind_note.trim()}`
          : "",
        form.structure_type === TimerStructureType.Custom &&
        form.other_structure_note.trim()
          ? `Other/Misc: ${form.other_structure_note.trim()}`
          : "",
        form.notes.trim(),
      ]
        .filter(Boolean)
        .join("\n");

      const payload = {
        title: form.title,
        system_id: form.system_id,
        system: form.system,
        structure_type: form.structure_type,
        planet_id: form.planet_id || 0,
        planet_name: form.planet_name,
        moon_id: form.moon_id || 0,
        moon_name: form.moon_name,
        owner_corporation_id: form.owner_corporation_id || 0,
        owner_corporation_name: form.owner_corporation_name,
        owner_corporation_ticker: form.owner_corporation_ticker,
        owner_alliance_id: form.owner_alliance_id || 0,
        owner_alliance_name: form.owner_alliance_name,
        owner_alliance_ticker: form.owner_alliance_ticker,
        standing_type: form.standing_type,
        timer_kind: form.timer_kind,
        stage_label: form.stage_label || TimerStageLabel.NotApplicable,
        replacement_action: form.replacement_action,
        skyhook_fullness_pct:
          form.skyhook_fullness_pct.trim() === ""
            ? null
            : Number(form.skyhook_fullness_pct),
        severity: form.severity,
        expires_at: expiresAtISO,
        notes: combinedNotes,
        raw_text: form.raw_text,
      };

      if (editingTimer) {
        await api.patch(`/timers/${editingTimer.id}`, payload);
      } else {
        await api.post("/timers", payload);
      }

      setToast({
        text: editingTimer ? "Timer updated" : "Timer added",
        color: "success",
      });
      resetFormState();
      setOpen(false);
      await onSaved();
    } catch {
      setToast({
        text: editingTimer ? "Failed to update timer" : "Failed to add timer",
        color: "error",
      });
    } finally {
      setSaving(false);
    }
  };

  const renderBody = () => (
    <>
      <ol className="steps steps-horizontal w-full text-xs">
        {stepMeta.map((item) => (
          <button
            key={item.value}
            className={`step ${step >= item.value ? "step-primary" : ""} cursor-pointer`}
            onClick={() => setStep(item.value)}
            type="button"
          >
            {item.label}
          </button>
        ))}
      </ol>
      <AddTimerStepContent
        selectedExpiresAt={selectedExpiresAt}
        parsePastedText={parsePastedText}
        eveDisplayDateToISO={eveDisplayDateToISO}
      />
    </>
  );

  const renderActions = () => (
    <>
      <button
        className="btn btn-sm btn-ghost"
        disabled={step === 1 || saving}
        onClick={() => setStep(Math.max(1, step - 1) as TimerFormStep)}
      >
        Back
      </button>
      {step < 4 ? (
        <button
          className="btn btn-sm btn-outline"
          disabled={saving}
          onClick={() => setStep(Math.min(4, step + 1) as TimerFormStep)}
        >
          Next
        </button>
      ) : (
        <button
          className="btn btn-sm btn-primary"
          disabled={saving}
          onClick={saveTimer}
        >
          {editingTimer ? (
            <SquarePen className="h-4 w-4" />
          ) : (
            <Plus className="h-4 w-4" />
          )}{" "}
          {editingTimer ? "Save changes" : "Save timer"}
        </button>
      )}
    </>
  );

  useModal({
    open,
    onDismiss: () => {
      setOpen(false);
      resetFormState();
    },
    build: () => ({
      title: editingTimer ? "Edit Timer" : "Add Timer",
      sizeClass: "w-[min(96vw,44rem)] xl:w-[34vw] max-w-none",
      body: renderBody(),
      actions: renderActions(),
    }),
  });

  return {
    openCreateModal,
    openEditModal,
  };
}
