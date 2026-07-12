import Staff from "./Staff";
import NotAuthorized from "@/components/NotAuthorized";
import { useAuthStore } from "@/app/store/authStore";
import LoadingCard from "@/components/LoadingCard";
import { useShallow } from "zustand/shallow";

export default function StaffRoute() {
  const { loaded, isStaff } = useAuthStore(
    useShallow((s) => ({
      loaded: s.loaded,
      isStaff: s.isStaff,
    })),
  );
  if (!loaded) {
    return <LoadingCard subtitle="Preparing staff tools…" />;
  }

  if (!isStaff) {
    return <NotAuthorized message="You don’t have access to this page." />;
  }

  return <Staff />;
}
