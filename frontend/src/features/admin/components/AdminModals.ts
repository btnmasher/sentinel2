import useModal from "@/app/hooks/useModal";
import type { ModalDefinition } from "@/app/hooks/useModal";
import { ADMIN_MODAL, type AdminModalKey } from "../store/adminStore";
import { AdminModalAccess } from "./AccessLevelModal";
import { AdminModalMove } from "./MoveCharacterModal";
import { AdminModalMerge } from "./MergeAccountModal";
import { AdminModalAudit } from "./AuditLogModal";
import { AdminModalAnnouncement } from "./SiteAnnouncementModal";
import { AdminModalAllowedOrganizations } from "./AllowedOrganizationsModal";

export const AdminModals = {
  Access: AdminModalAccess,
  Move: AdminModalMove,
  Merge: AdminModalMerge,
  Audit: AdminModalAudit,
  Announcement: AdminModalAnnouncement,
  AllowedOrganizations: AdminModalAllowedOrganizations,
} as const;

const AdminModalByKey = {
  [ADMIN_MODAL.Access]: AdminModalAccess,
  [ADMIN_MODAL.Move]: AdminModalMove,
  [ADMIN_MODAL.Merge]: AdminModalMerge,
  [ADMIN_MODAL.Audit]: AdminModalAudit,
  [ADMIN_MODAL.Announcement]: AdminModalAnnouncement,
  [ADMIN_MODAL.AllowedOrganizations]: AdminModalAllowedOrganizations,
} as const;

export function useAdminModal(key: AdminModalKey) {
  return useModal(AdminModalByKey[key] as ModalDefinition<AdminModalKey>);
}
