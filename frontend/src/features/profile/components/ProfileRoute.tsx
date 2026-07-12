import Profile from "./Profile";
import NotAuthorized from "@/components/NotAuthorized";
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

  if (!authLoaded) {
    return <LoadingCard subtitle="Preparing profile…" />;
  }

  if (!isAuthenticated) {
    return <NotAuthorized message="Please log in to view your profile." />;
  }

  return <Profile />;
}
