import ConfirmDialog from "./ConfirmDialog";
import HelpDialog from "./dialogs/HelpDialog";
import ShareLinkDialog from "./dialogs/ShareLinkDialog";
import PermissionRequiredDialog from "./dialogs/PermissionRequiredDialog";
import AlarmStartDialog from "./dialogs/AlarmStartDialog";

export default function Dialogs() {
  return (
    <>
      <ConfirmDialog />
      <HelpDialog />
      <ShareLinkDialog />
      <PermissionRequiredDialog />
      <AlarmStartDialog />
    </>
  );
}
