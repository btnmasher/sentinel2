import { useEffect, useState } from "react";
import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import {
  ADMIN_MODAL,
  defineAdminModal,
  useAdminStore,
} from "../store/adminStore";
import { useAdminActionsStore } from "../store/adminActionsStore";

type AnnouncementVariant = "banner" | "modal";

function SiteAnnouncementModalBody() {
  const { close } = useModalBody();
  const publishAnnouncement = useAdminActionsStore(
    (s) => s.publishAnnouncement,
  );
  const archiveLatestAnnouncement = useAdminActionsStore(
    (s) => s.archiveLatestAnnouncement,
  );
  const [variant, setVariant] = useState<AnnouncementVariant>("modal");
  const [message, setMessage] = useState("");

  useEffect(() => {
    setVariant("modal");
    setMessage("");
  }, []);

  return (
    <div className="space-y-4 text-sm text-slate-300">
      <p className="text-xs leading-relaxed text-slate-400">
        Publish a new site-wide message for users.
      </p>
      <label className="form-control gap-1.5">
        <span className="label-text text-xs text-slate-400">Variant</span>
        <select
          className="select select-sm w-full bg-base-300/70"
          value={variant}
          onChange={(event) =>
            setVariant(event.target.value as AnnouncementVariant)
          }
        >
          <option value="modal">Modal (markdown)</option>
          <option value="banner">Banner (plain text)</option>
        </select>
      </label>
      <label className="form-control gap-1.5">
        <span className="label-text text-xs text-slate-400">
          {variant === "modal" ? "Message (markdown)" : "Message (plain text)"}
        </span>
        <textarea
          className="textarea textarea-bordered min-h-40 w-full bg-base-300/70 font-mono text-xs leading-relaxed"
          placeholder="## Release Notes&#10;- Added feature X&#10;- Fixed issue Y"
          value={message}
          onChange={(event) => setMessage(event.target.value)}
        />
      </label>
      <div className="flex flex-wrap items-center gap-2 border-t border-slate-800/70 pt-3">
        <button
          className="btn btn-xs btn-outline"
          onClick={() => void archiveLatestAnnouncement()}
        >
          Archive latest
        </button>
        <div className="ml-auto flex items-center gap-2">
          <button className="btn btn-xs btn-outline" onClick={() => close()}>
            Cancel
          </button>
          <button
            className="btn btn-xs btn-outline"
            onClick={() => void publishAnnouncement(variant, message)}
          >
            Publish
          </button>
        </div>
      </div>
    </div>
  );
}

export const AdminModalAnnouncement = defineAdminModal({
  key: ADMIN_MODAL.Announcement,
  useOpen: () => useAdminStore((s) => s.modals[ADMIN_MODAL.Announcement]),
  build: () => ({
    title: "Publish Announcement",
    sizeClass: "max-w-2xl",
    body: <SiteAnnouncementModalBody />,
  }),
});

export default function SiteAnnouncementModal() {
  useModal(AdminModalAnnouncement);
  return null;
}
