export type SiteAnnouncementVariant = "banner" | "modal";

export type SiteAnnouncement = {
  id: string;
  variant: SiteAnnouncementVariant;
  message: string;
  created?: string;
};
