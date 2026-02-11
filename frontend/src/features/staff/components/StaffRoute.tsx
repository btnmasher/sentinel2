import Staff from "./Staff";
import NotAuthorized from "@/components/NotAuthorized";
import { useAppConfigStore } from "@/app/store/appConfigStore";
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
  const { loaded: configLoaded, standaloneAuth } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      standaloneAuth: s.standaloneAuth,
    })),
  );

  if (!loaded || !configLoaded) {
    return <LoadingCard subtitle="Preparing staff tools…" />;
  }

  if (!standaloneAuth) {
    return (
      <NotAuthorized message="This page isn’t available with the current login method." />
    );
  }

  if (!isStaff) {
    return <NotAuthorized message="You don’t have access to this page." />;
  }

  return <Staff />;
}
