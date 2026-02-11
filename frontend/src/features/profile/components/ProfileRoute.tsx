import Profile from "./Profile";
import NotAuthorized from "@/components/NotAuthorized";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import LoadingCard from "@/components/LoadingCard";
import { useShallow } from "zustand/shallow";

export default function ProfileRoute() {
  const { loaded: authLoaded, isAuthenticated } = useAuthStore(
    useShallow((s) => ({
      loaded: s.loaded,
      isAuthenticated: s.isAuthenticated,
    })),
  );
  const { loaded: configLoaded, standaloneAuth } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      standaloneAuth: s.standaloneAuth,
    })),
  );

  if (!authLoaded || !configLoaded) {
    return <LoadingCard subtitle="Preparing profile…" />;
  }

  if (!standaloneAuth) {
    return (
      <NotAuthorized message="Your profile isn’t available with the current login method." />
    );
  }

  if (!isAuthenticated) {
    return <NotAuthorized message="Please log in to view your profile." />;
  }

  return <Profile />;
}
