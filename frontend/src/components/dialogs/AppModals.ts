import useModal from "@/app/hooks/useModal";
import type { ModalDefinition } from "@/app/hooks/useModal";
import { UI_DIALOG, type DialogStateKey } from "@/app/store/uiStore";
import { AppModalHelp } from "@/components/dialogs/HelpDialog";
import { AppModalShareLink } from "@/components/dialogs/ShareLinkDialog";
import { AppModalPermissionRequired } from "@/components/dialogs/PermissionRequiredDialog";
import { AppModalAlarmStart } from "@/components/dialogs/AlarmStartDialog";
import { AppModalConfirm } from "@/components/ConfirmDialog";

export const AppModals = {
  Help: AppModalHelp,
  ShareLink: AppModalShareLink,
  PermissionRequired: AppModalPermissionRequired,
  AlarmStart: AppModalAlarmStart,
  Confirm: AppModalConfirm,
} as const;

const AppModalByKey = {
  [UI_DIALOG.Help]: AppModalHelp,
  [UI_DIALOG.ShareLink]: AppModalShareLink,
  [UI_DIALOG.PermissionRequired]: AppModalPermissionRequired,
  [UI_DIALOG.AlarmStart]: AppModalAlarmStart,
  [UI_DIALOG.Confirm]: AppModalConfirm,
} as const;

export function useAppModal(key: DialogStateKey) {
  return useModal(AppModalByKey[key] as ModalDefinition<DialogStateKey>);
}
